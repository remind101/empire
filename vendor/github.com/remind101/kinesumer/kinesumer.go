package kinesumer

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/remind101/kinesumer/checkpointers/empty"
	k "github.com/remind101/kinesumer/interface"
	"github.com/remind101/kinesumer/provisioners/empty"
)

type Kinesumer struct {
	Kinesis      k.Kinesis
	Checkpointer k.Checkpointer
	Provisioner  k.Provisioner
	Stream       string
	Options      *Options
	records      chan k.Record
	stop         chan Unit
	stopped      chan Unit
	nRunning     int
	rand         *rand.Rand
}

type Options struct {
	ListStreamsLimit    int64
	DescribeStreamLimit int64
	GetRecordsLimit     int64

	// Amount of time to poll of records if consumer lag is minimal
	PollTime            int
	MaxShardWorkers     int
	ErrHandler          func(k.Error)
	DefaultIteratorType string

	// How long to try and get shard iterator
	ShardAcquisitionTimeout time.Duration

	// ShardIteratorTimestamp is used when DefaultIteratorType is "AT_TIMESTAMP"
	ShardIteratorTimestamp time.Time
}

var DefaultOptions = Options{
	// These values are the hard limits set by Amazon
	ListStreamsLimit:        1000,
	DescribeStreamLimit:     10000,
	GetRecordsLimit:         10000,
	PollTime:                2000,
	MaxShardWorkers:         50,
	ErrHandler:              DefaultErrHandler,
	DefaultIteratorType:     "LATEST",
	ShardAcquisitionTimeout: 90 * time.Second,
}

func NewDefault(stream string, duration time.Duration) (*Kinesumer, error) {
	return New(
		kinesis.New(session.New()),
		nil,
		nil,
		nil,
		stream,
		nil,
		duration,
	)
}

func New(kinesis k.Kinesis, checkpointer k.Checkpointer, provisioner k.Provisioner,
	randSource rand.Source, stream string, opt *Options, duration time.Duration) (*Kinesumer, error) {

	if kinesis == nil {
		return nil, NewError(ECrit, "Kinesis object must not be nil", nil)
	}

	if checkpointer == nil {
		checkpointer = emptycheckpointer.Checkpointer{}
	}

	if provisioner == nil {
		provisioner = emptyprovisioner.Provisioner{}
	}

	if randSource == nil {
		randSource = rand.NewSource(time.Now().UnixNano())
	}

	if len(stream) == 0 {
		return nil, NewError(ECrit, "Stream name can't be empty", nil)
	}

	if opt == nil {
		tmp := DefaultOptions
		opt = &tmp
	}

	if opt.ErrHandler == nil {
		opt.ErrHandler = DefaultErrHandler
	}

	if duration != 0 {
		opt.DefaultIteratorType = "AT_TIMESTAMP"
		opt.ShardIteratorTimestamp = time.Now().Add(duration * -1)
	}

	return &Kinesumer{
		Kinesis:      kinesis,
		Checkpointer: checkpointer,
		Provisioner:  provisioner,
		Stream:       stream,
		Options:      opt,
		records:      make(chan k.Record, opt.GetRecordsLimit*2+10),
		rand:         rand.New(randSource),
	}, nil
}

func (kin *Kinesumer) GetStreams() (streams []string, err error) {
	streams = make([]string, 0)
	err = kin.Kinesis.ListStreamsPages(&kinesis.ListStreamsInput{
		Limit: &kin.Options.ListStreamsLimit,
	}, func(sts *kinesis.ListStreamsOutput, _ bool) bool {
		streams = append(streams, aws.StringValueSlice(sts.StreamNames)...)
		return true
	})
	return
}

func (kin *Kinesumer) StreamExists() (found bool, err error) {
	streams, err := kin.GetStreams()
	if err != nil {
		return
	}
	for _, stream := range streams {
		if stream == kin.Stream {
			return true, nil
		}
	}
	return
}

func (kin *Kinesumer) GetShards() (shards []*kinesis.Shard, err error) {
	for {
		retry := false
		shards = make([]*kinesis.Shard, 0)
		err = kin.Kinesis.DescribeStreamPages(&kinesis.DescribeStreamInput{
			Limit:      &kin.Options.DescribeStreamLimit,
			StreamName: &kin.Stream,
		}, func(desc *kinesis.DescribeStreamOutput, _ bool) bool {
			if desc == nil || desc.StreamDescription == nil {
				err = errors.New("Stream could not be described")
				return false
			}
			switch aws.StringValue(desc.StreamDescription.StreamStatus) {
			case "CREATING":
				retry = true
				return false
			case "DELETING":
				err = errors.New("Stream is being deleted")
				return false
			}
			shards = append(shards, desc.StreamDescription.Shards...)
			return true
		})
		if retry {
			time.Sleep(time.Second)
		} else {
			return
		}
	}
}

func (kin *Kinesumer) LaunchShardWorker(shards []*kinesis.Shard) (int, *ShardWorker, error) {
	perm := kin.rand.Perm(len(shards))
	for _, j := range perm {
		err := kin.Provisioner.TryAcquire(aws.StringValue(shards[j].ShardId))
		if err == nil {
			worker := &ShardWorker{
				kinesis:                kin.Kinesis,
				shard:                  shards[j],
				checkpointer:           kin.Checkpointer,
				stream:                 kin.Stream,
				pollTime:               kin.Options.PollTime,
				stop:                   kin.stop,
				stopped:                kin.stopped,
				c:                      kin.records,
				provisioner:            kin.Provisioner,
				errHandler:             kin.Options.ErrHandler,
				defaultIteratorType:    kin.Options.DefaultIteratorType,
				shardIteratorTimestamp: kin.Options.ShardIteratorTimestamp,
				GetRecordsLimit:        kin.Options.GetRecordsLimit,
			}
			kin.nRunning++
			go worker.RunWorker()
			return j, worker, nil
		}
	}
	return 0, nil, errors.New("No unlocked keys")
}

func (kin *Kinesumer) Begin() (int, error) {
	shards, err := kin.GetShards()
	if err != nil {
		return 0, err
	}

	err = kin.Checkpointer.Begin()
	if err != nil {
		return 0, err
	}

	n := kin.Options.MaxShardWorkers
	if n <= 0 || len(shards) < n {
		n = len(shards)
	}

	tryTime := kin.Options.ShardAcquisitionTimeout
	if tryTime < 2*kin.Provisioner.TTL()+time.Second {
		tryTime = 2*kin.Provisioner.TTL() + time.Second
	}

	start := time.Now()

	kin.stop = make(chan Unit, n)
	kin.stopped = make(chan Unit, n)

	workers := make([]*ShardWorker, 0)
	for kin.nRunning < n && len(shards) > 0 && time.Now().Sub(start) < tryTime {
		for i := kin.nRunning; i < n; i++ {
			j, worker, err := kin.LaunchShardWorker(shards)
			if err != nil {
				kin.Options.ErrHandler(NewError(EWarn, "Could not start shard worker", err))
			} else {
				workers = append(workers, worker)
				shards = append(shards[:j], shards[j+1:]...)
			}
		}
		time.Sleep(time.Duration(500+rand.Intn(1500)) * time.Millisecond)
	}

	kin.Options.ErrHandler(NewError(EInfo, fmt.Sprintf("%v/%v workers started", kin.nRunning, n), nil))

	if len(workers) < 1 {
		return len(workers), NewError(EWarn, "0 shard workers started", nil)
	}

	return len(workers), nil
}

func (kin *Kinesumer) End() {
	for kin.nRunning > 0 {
		select {
		case <-kin.stopped:
			kin.nRunning--
		case kin.stop <- Unit{}:
		}
	}
	kin.Checkpointer.End()
}

func (kin *Kinesumer) Records() <-chan k.Record {
	return kin.records
}
