package mocks

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/service"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/stretchr/testify/mock"
)

type Kinesis struct {
	mock.Mock
}

func (m *Kinesis) AddTagsToStreamRequest(_a0 *kinesis.AddTagsToStreamInput) (*service.Request, *kinesis.AddTagsToStreamOutput) {
	ret := m.Called(_a0)

	var r0 *service.Request
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*service.Request)
	}
	var r1 *kinesis.AddTagsToStreamOutput
	if ret.Get(1) != nil {
		r1 = ret.Get(1).(*kinesis.AddTagsToStreamOutput)
	}

	return r0, r1
}
func (m *Kinesis) AddTagsToStream(_a0 *kinesis.AddTagsToStreamInput) (*kinesis.AddTagsToStreamOutput, error) {
	ret := m.Called(_a0)

	var r0 *kinesis.AddTagsToStreamOutput
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*kinesis.AddTagsToStreamOutput)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Kinesis) CreateStreamRequest(_a0 *kinesis.CreateStreamInput) (*service.Request, *kinesis.CreateStreamOutput) {
	ret := m.Called(_a0)

	var r0 *service.Request
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*service.Request)
	}
	var r1 *kinesis.CreateStreamOutput
	if ret.Get(1) != nil {
		r1 = ret.Get(1).(*kinesis.CreateStreamOutput)
	}

	return r0, r1
}
func (m *Kinesis) CreateStream(_a0 *kinesis.CreateStreamInput) (*kinesis.CreateStreamOutput, error) {
	ret := m.Called(_a0)

	var r0 *kinesis.CreateStreamOutput
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*kinesis.CreateStreamOutput)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Kinesis) DeleteStreamRequest(_a0 *kinesis.DeleteStreamInput) (*service.Request, *kinesis.DeleteStreamOutput) {
	ret := m.Called(_a0)

	var r0 *service.Request
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*service.Request)
	}
	var r1 *kinesis.DeleteStreamOutput
	if ret.Get(1) != nil {
		r1 = ret.Get(1).(*kinesis.DeleteStreamOutput)
	}

	return r0, r1
}
func (m *Kinesis) DeleteStream(_a0 *kinesis.DeleteStreamInput) (*kinesis.DeleteStreamOutput, error) {
	ret := m.Called(_a0)

	var r0 *kinesis.DeleteStreamOutput
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*kinesis.DeleteStreamOutput)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Kinesis) DescribeStreamRequest(_a0 *kinesis.DescribeStreamInput) (*service.Request, *kinesis.DescribeStreamOutput) {
	ret := m.Called(_a0)

	var r0 *service.Request
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*service.Request)
	}
	var r1 *kinesis.DescribeStreamOutput
	if ret.Get(1) != nil {
		r1 = ret.Get(1).(*kinesis.DescribeStreamOutput)
	}

	return r0, r1
}
func (m *Kinesis) DescribeStream(_a0 *kinesis.DescribeStreamInput) (*kinesis.DescribeStreamOutput, error) {
	ret := m.Called(_a0)

	var r0 *kinesis.DescribeStreamOutput
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*kinesis.DescribeStreamOutput)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Kinesis) DescribeStreamPages(_a0 *kinesis.DescribeStreamInput, _a1 func(*kinesis.DescribeStreamOutput, bool) bool) error {
	ret := m.Called(_a0, _a1)

	a := kinesis.Shard{
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
	}
	b := kinesis.Shard{
		AdjacentParentShardId: nil,
		HashKeyRange: &kinesis.HashKeyRange{
			StartingHashKey: aws.String("80"),
			EndingHashKey:   aws.String("ff"),
		},
		ParentShardId: nil,
		SequenceNumberRange: &kinesis.SequenceNumberRange{
			StartingSequenceNumber: aws.String("101"),
			EndingSequenceNumber:   aws.String("200"),
		},
		ShardId: aws.String("shard1"),
	}
	cont := _a1(
		&kinesis.DescribeStreamOutput{
			StreamDescription: &kinesis.StreamDescription{
				HasMoreShards: aws.Bool(true),
				Shards:        []*kinesis.Shard{&a},
				StreamName:    aws.String("TestStream"),
				StreamStatus:  aws.String("ACTIVE"),
			},
		}, true)
	if cont {
		_a1(
			&kinesis.DescribeStreamOutput{
				StreamDescription: &kinesis.StreamDescription{
					HasMoreShards: aws.Bool(true),
					Shards:        []*kinesis.Shard{&b},
					StreamName:    aws.String("TestStream"),
					StreamStatus:  aws.String("ACTIVE"),
				},
			}, false)
	}
	r0 := ret.Error(0)

	return r0
}
func (m *Kinesis) GetRecordsRequest(_a0 *kinesis.GetRecordsInput) (*service.Request, *kinesis.GetRecordsOutput) {
	ret := m.Called(_a0)

	var r0 *service.Request
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*service.Request)
	}
	var r1 *kinesis.GetRecordsOutput
	if ret.Get(1) != nil {
		r1 = ret.Get(1).(*kinesis.GetRecordsOutput)
	}

	return r0, r1
}
func (m *Kinesis) GetRecords(_a0 *kinesis.GetRecordsInput) (*kinesis.GetRecordsOutput, error) {
	ret := m.Called(_a0)

	var r0 *kinesis.GetRecordsOutput
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*kinesis.GetRecordsOutput)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Kinesis) GetShardIteratorRequest(_a0 *kinesis.GetShardIteratorInput) (*service.Request, *kinesis.GetShardIteratorOutput) {
	ret := m.Called(_a0)

	var r0 *service.Request
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*service.Request)
	}
	var r1 *kinesis.GetShardIteratorOutput
	if ret.Get(1) != nil {
		r1 = ret.Get(1).(*kinesis.GetShardIteratorOutput)
	}

	return r0, r1
}
func (m *Kinesis) GetShardIterator(_a0 *kinesis.GetShardIteratorInput) (*kinesis.GetShardIteratorOutput, error) {
	ret := m.Called(_a0)

	var r0 *kinesis.GetShardIteratorOutput
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*kinesis.GetShardIteratorOutput)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Kinesis) ListStreamsRequest(_a0 *kinesis.ListStreamsInput) (*service.Request, *kinesis.ListStreamsOutput) {
	ret := m.Called(_a0)

	var r0 *service.Request
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*service.Request)
	}
	var r1 *kinesis.ListStreamsOutput
	if ret.Get(1) != nil {
		r1 = ret.Get(1).(*kinesis.ListStreamsOutput)
	}

	return r0, r1
}
func (m *Kinesis) ListStreams(_a0 *kinesis.ListStreamsInput) (*kinesis.ListStreamsOutput, error) {
	ret := m.Called(_a0)

	var r0 *kinesis.ListStreamsOutput
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*kinesis.ListStreamsOutput)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Kinesis) ListStreamsPages(_a0 *kinesis.ListStreamsInput, _a1 func(*kinesis.ListStreamsOutput, bool) bool) error {
	ret := m.Called(_a0, _a1)

	cont := _a1(&kinesis.ListStreamsOutput{
		HasMoreStreams: aws.Bool(true),
		StreamNames:    []*string{aws.String("a"), aws.String("b")},
	}, true)
	if cont {
		_a1(&kinesis.ListStreamsOutput{
			HasMoreStreams: aws.Bool(false),
			StreamNames:    []*string{aws.String("c")},
		}, false)
	}
	r0 := ret.Error(0)

	return r0
}
func (m *Kinesis) ListTagsForStreamRequest(_a0 *kinesis.ListTagsForStreamInput) (*service.Request, *kinesis.ListTagsForStreamOutput) {
	ret := m.Called(_a0)

	var r0 *service.Request
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*service.Request)
	}
	var r1 *kinesis.ListTagsForStreamOutput
	if ret.Get(1) != nil {
		r1 = ret.Get(1).(*kinesis.ListTagsForStreamOutput)
	}

	return r0, r1
}
func (m *Kinesis) ListTagsForStream(_a0 *kinesis.ListTagsForStreamInput) (*kinesis.ListTagsForStreamOutput, error) {
	ret := m.Called(_a0)

	var r0 *kinesis.ListTagsForStreamOutput
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*kinesis.ListTagsForStreamOutput)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Kinesis) MergeShardsRequest(_a0 *kinesis.MergeShardsInput) (*service.Request, *kinesis.MergeShardsOutput) {
	ret := m.Called(_a0)

	var r0 *service.Request
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*service.Request)
	}
	var r1 *kinesis.MergeShardsOutput
	if ret.Get(1) != nil {
		r1 = ret.Get(1).(*kinesis.MergeShardsOutput)
	}

	return r0, r1
}
func (m *Kinesis) MergeShards(_a0 *kinesis.MergeShardsInput) (*kinesis.MergeShardsOutput, error) {
	ret := m.Called(_a0)

	var r0 *kinesis.MergeShardsOutput
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*kinesis.MergeShardsOutput)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Kinesis) PutRecordRequest(_a0 *kinesis.PutRecordInput) (*service.Request, *kinesis.PutRecordOutput) {
	ret := m.Called(_a0)

	var r0 *service.Request
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*service.Request)
	}
	var r1 *kinesis.PutRecordOutput
	if ret.Get(1) != nil {
		r1 = ret.Get(1).(*kinesis.PutRecordOutput)
	}

	return r0, r1
}
func (m *Kinesis) PutRecord(_a0 *kinesis.PutRecordInput) (*kinesis.PutRecordOutput, error) {
	ret := m.Called(_a0)

	var r0 *kinesis.PutRecordOutput
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*kinesis.PutRecordOutput)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Kinesis) PutRecordsRequest(_a0 *kinesis.PutRecordsInput) (*service.Request, *kinesis.PutRecordsOutput) {
	ret := m.Called(_a0)

	var r0 *service.Request
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*service.Request)
	}
	var r1 *kinesis.PutRecordsOutput
	if ret.Get(1) != nil {
		r1 = ret.Get(1).(*kinesis.PutRecordsOutput)
	}

	return r0, r1
}
func (m *Kinesis) PutRecords(_a0 *kinesis.PutRecordsInput) (*kinesis.PutRecordsOutput, error) {
	ret := m.Called(_a0)

	var r0 *kinesis.PutRecordsOutput
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*kinesis.PutRecordsOutput)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Kinesis) RemoveTagsFromStreamRequest(_a0 *kinesis.RemoveTagsFromStreamInput) (*service.Request, *kinesis.RemoveTagsFromStreamOutput) {
	ret := m.Called(_a0)

	var r0 *service.Request
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*service.Request)
	}
	var r1 *kinesis.RemoveTagsFromStreamOutput
	if ret.Get(1) != nil {
		r1 = ret.Get(1).(*kinesis.RemoveTagsFromStreamOutput)
	}

	return r0, r1
}
func (m *Kinesis) RemoveTagsFromStream(_a0 *kinesis.RemoveTagsFromStreamInput) (*kinesis.RemoveTagsFromStreamOutput, error) {
	ret := m.Called(_a0)

	var r0 *kinesis.RemoveTagsFromStreamOutput
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*kinesis.RemoveTagsFromStreamOutput)
	}
	r1 := ret.Error(1)

	return r0, r1
}
func (m *Kinesis) SplitShardRequest(_a0 *kinesis.SplitShardInput) (*service.Request, *kinesis.SplitShardOutput) {
	ret := m.Called(_a0)

	var r0 *service.Request
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*service.Request)
	}
	var r1 *kinesis.SplitShardOutput
	if ret.Get(1) != nil {
		r1 = ret.Get(1).(*kinesis.SplitShardOutput)
	}

	return r0, r1
}
func (m *Kinesis) SplitShard(_a0 *kinesis.SplitShardInput) (*kinesis.SplitShardOutput, error) {
	ret := m.Called(_a0)

	var r0 *kinesis.SplitShardOutput
	if ret.Get(0) != nil {
		r0 = ret.Get(0).(*kinesis.SplitShardOutput)
	}
	r1 := ret.Error(1)

	return r0, r1
}
