package kinesumer

import (
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/kinesis"
	k "github.com/remind101/kinesumer/interface"
	"github.com/remind101/kinesumer/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func makeTestShardWorker() (*ShardWorker, *mocks.Kinesis, *mocks.Checkpointer, *mocks.Provisioner,
	chan Unit, chan Unit, chan k.Record) {
	kin := new(mocks.Kinesis)
	sssm := new(mocks.Checkpointer)
	prov := new(mocks.Provisioner)
	stop := make(chan Unit, 1)
	stopped := make(chan Unit, 1)
	c := make(chan k.Record, 100)

	return &ShardWorker{
		kinesis: kin,
		shard: &kinesis.Shard{
			AdjacentParentShardId: nil,
			HashKeyRange: &kinesis.HashKeyRange{
				StartingHashKey: aws.String("0"),
				EndingHashKey:   aws.String("7f"),
			},
			ParentShardId: nil,
			SequenceNumberRange: &kinesis.SequenceNumberRange{
				StartingSequenceNumber: aws.String("0"),
				EndingSequenceNumber:   aws.String("100"),
			},
			ShardId: aws.String("shard0"),
		},
		checkpointer:    sssm,
		stream:          "TestStream",
		sequence:        "123",
		stop:            stop,
		stopped:         stopped,
		c:               c,
		provisioner:     prov,
		errHandler:      DefaultErrHandler,
		GetRecordsLimit: 123,
	}, kin, sssm, prov, stop, stopped, c
}

func TestShardWorkerGetShardIterator(t *testing.T) {
	s, kin, _, _, _, _, _ := makeTestShardWorker()

	kin.On("GetShardIterator", mock.Anything).Return(&kinesis.GetShardIteratorOutput{
		ShardIterator: aws.String("AAAAA"),
	}, awserr.Error(nil))
	res, err := s.GetShardIterator("TYPE", "123")
	assert.Nil(t, err)
	assert.Equal(t, "AAAAA", res)
}

func TestShardWorkerTryGetShardIterator(t *testing.T) {
	s, kin, _, _, _, _, _ := makeTestShardWorker()

	kin.On("GetShardIterator", mock.Anything).Return(nil, awserr.New("bad", "bad", errors.New("bad")))
	assert.Panics(t, func() {
		s.TryGetShardIterator("TYPE", "123")
	})
}

func TestShardWorkerGetRecords(t *testing.T) {
	s, kin, _, _, _, _, _ := makeTestShardWorker()

	kin.On("GetRecords", mock.Anything).Return(&kinesis.GetRecordsOutput{
		MillisBehindLatest: aws.Int64(0),
		NextShardIterator:  aws.String("AAAA"),
		Records:            []*kinesis.Record{},
	}, awserr.Error(nil))

	records, nextIt, mills, err := s.GetRecords("AAAA")
	assert.Nil(t, err)
	assert.Equal(t, 0, len(records))
	assert.Equal(t, "AAAA", nextIt)
	assert.Equal(t, int64(0), mills)
}

func TestShardWorkerGetRecordsAndProcess(t *testing.T) {
	s, kin, sssm, prov, stp, _, c := makeTestShardWorker()

	prov.On("Heartbeat", mock.Anything).Return(nil)

	record1 := kinesis.Record{
		Data:           []byte("help I'm trapped"),
		PartitionKey:   aws.String("aaaa"),
		SequenceNumber: aws.String("123"),
	}
	kin.On("GetRecords", mock.Anything).Return(&kinesis.GetRecordsOutput{
		MillisBehindLatest: aws.Int64(0),
		NextShardIterator:  aws.String("AAAA"),
		Records:            []*kinesis.Record{&record1},
	}, awserr.Error(nil)).Once()
	doneC := make(chan k.Record)
	sssm.On("DoneC").Return(doneC)
	brk, nextIt, nextSeq := s.GetRecordsAndProcess("AAAA", "123")
	rec := <-c
	assert.Equal(t, record1.Data, rec.Data())
	assert.False(t, brk)
	assert.Equal(t, "AAAA", nextIt)
	assert.Equal(t, "123", nextSeq)

	err := awserr.New("bad", "bad", nil)
	stp <- Unit{}
	kin.On("GetRecords", mock.Anything).Return(&kinesis.GetRecordsOutput{
		MillisBehindLatest: aws.Int64(0),
		NextShardIterator:  aws.String("AAAA"),
		Records:            []*kinesis.Record{},
	}, err)
	kin.On("GetShardIterator", mock.Anything).Return(&kinesis.GetShardIteratorOutput{
		ShardIterator: aws.String("AAAA"),
	}, awserr.Error(nil))
	brk, nextIt, nextSeq = s.GetRecordsAndProcess("AAAA", "123")
	kin.AssertNumberOfCalls(t, "GetShardIterator", 1)
	assert.True(t, brk)
}

func TestShardWorkerRun(t *testing.T) {
	s, kin, sssm, prov, stp, stpd, c := makeTestShardWorker()

	prov.On("Heartbeat", mock.Anything).Return(nil)
	prov.On("Release", mock.Anything).Return(nil)
	sssm.On("GetStartSequence", mock.Anything).Return("AAAA")

	record1 := kinesis.Record{
		Data:           []byte("help I'm trapped"),
		PartitionKey:   aws.String("aaaa"),
		SequenceNumber: aws.String("123"),
	}
	kin.On("GetRecords", mock.Anything).Return(&kinesis.GetRecordsOutput{
		MillisBehindLatest: aws.Int64(0),
		NextShardIterator:  aws.String("AAAA"),
		Records:            []*kinesis.Record{&record1},
	}, awserr.Error(nil)).Once()
	kin.On("GetRecords", mock.Anything).Return(&kinesis.GetRecordsOutput{
		MillisBehindLatest: aws.Int64(0),
		NextShardIterator:  aws.String("AAAA"),
		Records:            []*kinesis.Record{},
	}, awserr.Error(nil))
	doneC := make(chan k.Record)
	sssm.On("DoneC").Return(doneC)
	kin.On("GetShardIterator", mock.Anything).Return(&kinesis.GetShardIteratorOutput{
		ShardIterator: aws.String("AAAA"),
	}, awserr.Error(nil))
	go func() {
		time.Sleep(10 * time.Millisecond)
		stp <- Unit{}
	}()
	s.RunWorker()
	<-stpd
	rec := <-c
	assert.Equal(t, record1.Data, rec.Data())
}
