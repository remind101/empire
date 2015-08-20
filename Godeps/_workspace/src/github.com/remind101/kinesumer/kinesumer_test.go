package kinesumer

import (
	"math/rand"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/remind101/kinesumer/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func makeTestKinesumer(t *testing.T) (*Kinesumer, *mocks.Kinesis, *mocks.Checkpointer,
	*mocks.Provisioner) {
	kin := new(mocks.Kinesis)
	sssm := new(mocks.Checkpointer)
	prov := new(mocks.Provisioner)
	k, err := New(
		kin,
		sssm,
		prov,
		rand.NewSource(0),
		"TestStream",
		nil,
	)
	if err != nil {
		t.Error(err)
	}
	return k, kin, sssm, prov
}

func TestKinesumerGetStreams(t *testing.T) {
	k, kin, _, _ := makeTestKinesumer(t)
	kin.On("ListStreamsPages", mock.Anything, mock.Anything).Return(nil)
	streams, err := k.GetStreams()
	assert.Nil(t, err)
	kin.AssertNumberOfCalls(t, "ListStreamsPages", 1)
	assert.Equal(t, 3, len(streams))
	assert.Equal(t, streams[2], "c")
}

func TestKinesumerStreamExists(t *testing.T) {
	k, kin, _, _ := makeTestKinesumer(t)
	k.Stream = "c"
	kin.On("ListStreamsPages", mock.Anything, mock.Anything).Return(nil)
	e, err := k.StreamExists()
	assert.Nil(t, err)
	kin.AssertNumberOfCalls(t, "ListStreamsPages", 1)
	assert.True(t, e)
}

func TestKinesumerGetShards(t *testing.T) {
	k, kin, _, _ := makeTestKinesumer(t)
	k.Stream = "c"
	kin.On("DescribeStreamPages", mock.Anything, mock.Anything).Return(nil)
	shards, err := k.GetShards()
	assert.Nil(t, err)
	kin.AssertNumberOfCalls(t, "DescribeStreamPages", 1)
	assert.Equal(t, 2, len(shards))
	assert.Equal(t, "shard1", *shards[1].ShardId)
}

func TestKinesumerBeginEnd(t *testing.T) {
	k, kin, sssm, prov := makeTestKinesumer(t)
	k.Stream = "c"

	kin.On("DescribeStreamPages", mock.Anything, mock.Anything).Return(awserr.New("bad", "bad", nil)).Once()
	_, err := k.Begin()
	assert.Error(t, err)

	prov.On("TTL").Return(time.Millisecond * 10)
	prov.On("TryAcquire", mock.Anything).Return(nil)
	prov.On("Heartbeat", mock.Anything).Return(nil)
	prov.On("Release", mock.Anything).Return(nil)
	kin.On("DescribeStreamPages", mock.Anything, mock.Anything).Return(awserr.Error(nil))
	sssm.On("Begin", mock.Anything).Return(nil)
	sssm.On("GetStartSequence", mock.Anything).Return("0").Once()
	sssm.On("GetStartSequence", mock.Anything).Return("")
	sssm.On("TryAcquire", mock.Anything).Return(nil)
	kin.On("GetShardIterator", mock.Anything).Return(&kinesis.GetShardIteratorOutput{
		ShardIterator: aws.String("0"),
	}, awserr.Error(nil))
	kin.On("GetRecords", mock.Anything).Return(&kinesis.GetRecordsOutput{
		MillisBehindLatest: aws.Int64(0),
		NextShardIterator:  aws.String("AAAAA"),
		Records:            []*kinesis.Record{},
	}, awserr.Error(nil))
	sssm.On("End").Return()
	_, err = k.Begin()
	assert.Nil(t, err)
	assert.Equal(t, 2, k.nRunning)
	k.End()
}
