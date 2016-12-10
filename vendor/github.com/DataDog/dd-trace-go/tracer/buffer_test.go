package tracer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSpanBufferMaxSize(t *testing.T) {
	assert := assert.New(t)

	sb := newSpansBuffer(2)
	assert.Equal(sb.Len(), 0)
	sb.Push(&Span{})
	assert.Equal(sb.Len(), 1)
	sb.Push(&Span{})
	assert.Equal(sb.Len(), 2)
	sb.Push(&Span{})
	assert.Equal(sb.Len(), 2)
}

func TestSpanBufferPushPop(t *testing.T) {
	assert := assert.New(t)

	sb := newSpansBuffer(10)

	s1 := &Span{Name: "1"}
	s2 := &Span{Name: "2"}
	s3 := &Span{Name: "3"}

	sb.Push(s1)
	sb.Push(s2)

	out := sb.Pop()
	assert.Equal([]*Span{s1, s2}, out)

	sb.Push(s3)
	out = sb.Pop()
	assert.Equal([]*Span{s3}, out)
}
