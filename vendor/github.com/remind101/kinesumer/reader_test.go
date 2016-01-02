package kinesumer

import (
	"bytes"
	"io"
	"testing"

	k "github.com/remind101/kinesumer/interface"
	"github.com/stretchr/testify/assert"
)

func TestReader_Read_NoData(t *testing.T) {
	ch := make(chan k.Record)
	r := NewReader(ch)

	b := make([]byte, 1)
	n, err := r.Read(b)
	assert.Nil(t, err)
	assert.Equal(t, 0, n)
}

func TestReader_Read_SingleByte(t *testing.T) {
	ch := make(chan k.Record, 1)
	checkpointC := make(chan k.Record, 1)
	r := NewReader(ch)

	// Check that we can read a single byte into a byte slice of size 1.
	record := &Record{data: []byte{0x01}, checkpointC: checkpointC}
	ch <- record
	b := make([]byte, 1)

	n, err := r.Read(b)
	assert.Nil(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, []byte{0x01}, b)
	assertCheckpointed(t, checkpointC, record)
}

func TestReader_Read_SmallBuffer(t *testing.T) {
	ch := make(chan k.Record, 1)
	checkpointC := make(chan k.Record, 1)
	r := NewReader(ch)

	// Check that, if the record has more data than the size of the buffer
	// we're provided, we buffer the data.
	record := &Record{data: []byte{0x01, 0x02}, checkpointC: checkpointC}
	ch <- record
	b := make([]byte, 1)

	n, err := r.Read(b)
	assert.Nil(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, []byte{0x01}, b)
	assertNotCheckpointed(t, checkpointC)

	n, err = r.Read(b)
	assert.Nil(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, []byte{0x02}, b)
	assertCheckpointed(t, checkpointC, record)
}

func TestReader_Read_LargeBuffer(t *testing.T) {
	ch := make(chan k.Record, 2)
	checkpointC := make(chan k.Record, 2)
	r := NewReader(ch)

	record := &Record{data: []byte{0x01}, checkpointC: checkpointC}
	ch <- record
	b := make([]byte, 2)

	n, err := r.Read(b)
	assert.Nil(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, []byte{0x01, 0x00}, b)
	assertCheckpointed(t, checkpointC, record)

	record = &Record{data: []byte{0x01}, checkpointC: checkpointC}
	ch <- record
	n, err = r.Read(b)
	assert.Nil(t, err)
	assert.Equal(t, 1, n)
	assert.Equal(t, []byte{0x01, 0x00}, b)
	assertCheckpointed(t, checkpointC, record)
}

func TestReader_Read_MultipleRecords(t *testing.T) {
	ch := make(chan k.Record, 2)
	checkpointC := make(chan k.Record, 2)
	r := NewReader(ch)

	record1 := &Record{data: []byte{0x01}, checkpointC: checkpointC}
	ch <- record1
	record2 := &Record{data: []byte{0x02}, checkpointC: checkpointC}
	ch <- record2

	b := make([]byte, 2)
	n, err := r.Read(b)
	assert.Nil(t, err)
	assert.Equal(t, 2, n)
	assert.Equal(t, []byte{0x01, 0x02}, b)

	assertCheckpointed(t, checkpointC, record1)
	assertCheckpointed(t, checkpointC, record2)
}

func TestReader_Read_Copy(t *testing.T) {
	ch := make(chan k.Record, 2)
	checkpointC := make(chan k.Record, 2)
	r := NewReader(ch)

	ch <- &Record{data: []byte{'a'}, checkpointC: checkpointC}
	ch <- &Record{data: []byte{'b'}, checkpointC: checkpointC}
	close(ch)

	b := new(bytes.Buffer)

	_, err := io.Copy(b, r)
	assert.Nil(t, err)
	assert.Equal(t, "ab", b.String())
}

func assertCheckpointed(t testing.TB, checkpointC chan k.Record, record k.Record) {
	select {
	case r := <-checkpointC:
		assert.Equal(t, record, r)
	default:
		t.Fatalf("Expected Done to be called on record: %v", record)
	}
}

func assertNotCheckpointed(t testing.TB, checkpointC chan k.Record) {
	select {
	case <-checkpointC:
		t.Fatal("Expected no checkpoint")
	default:
	}
}
