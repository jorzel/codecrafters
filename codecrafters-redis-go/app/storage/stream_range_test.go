package storage_test

import (
	"math"
	"testing"

	"github.com/codecrafters-io/redis-starter-go/app/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func concreteID(ms, seq uint64) storage.StreamID {
	return storage.StreamID{Type: storage.IDConcrete, Ms: ms, Seq: seq}
}

func newStreamWithIDs(t *testing.T, ids ...string) *storage.StreamRecord {
	t.Helper()
	stream := storage.NewStreamRecord()
	for _, id := range ids {
		_, err := stream.AddItem(id, []storage.Field{{"f", "k"}})
		require.NoError(t, err)
	}
	return stream
}

func itemIDs(items []storage.StreamItem) []storage.StreamID {
	ids := []storage.StreamID{}
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func TestRangeReturnsItemsWithinBoundsInclusive(t *testing.T) {
	stream := newStreamWithIDs(t, "1-0", "1-1", "2-0", "3-0")

	items := stream.Range(concreteID(1, 1), concreteID(2, 0))

	assert.Equal(t, []storage.StreamID{concreteID(1, 1), concreteID(2, 0)}, itemIDs(items))
}

func TestRangeWithStartAfterEndReturnsNoItems(t *testing.T) {
	stream := newStreamWithIDs(t, "1-0", "2-0")

	items := stream.Range(concreteID(2, 0), concreteID(1, 0))

	assert.Empty(t, items)
}

func TestParseRangeStartSuccess(t *testing.T) {
	tc := []struct {
		name  string
		input string
		want  storage.StreamID
	}{
		{name: "full id", input: "1-2", want: concreteID(1, 2)},
		{name: "ms only defaults seq to 0", input: "1", want: concreteID(1, 0)},
		{name: "minus is the smallest id", input: "-", want: concreteID(0, 0)},
		{name: "plus is the largest id", input: "+", want: concreteID(math.MaxUint64, math.MaxUint64)},
	}
	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			got, err := storage.ParseRangeStart(tt.input)

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseRangeStartFailure(t *testing.T) {
	tc := []struct {
		name  string
		input string
	}{
		{name: "auto id", input: "*"},
		{name: "auto sequence", input: "1-*"},
		{name: "not a number", input: "abc"},
	}
	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			_, err := storage.ParseRangeStart(tt.input)

			require.Error(t, err)
		})
	}
}

func TestParseRangeEndSuccess(t *testing.T) {
	tc := []struct {
		name  string
		input string
		want  storage.StreamID
	}{
		{name: "full id", input: "1-2", want: concreteID(1, 2)},
		{name: "explicit zero seq stays zero", input: "1-0", want: concreteID(1, 0)},
		{name: "ms only defaults seq to max", input: "1", want: concreteID(1, math.MaxUint64)},
		{name: "plus is the largest id", input: "+", want: concreteID(math.MaxUint64, math.MaxUint64)},
		{name: "minus is the smallest id", input: "-", want: concreteID(0, 0)},
	}
	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			got, err := storage.ParseRangeEnd(tt.input)

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseRangeEndFailure(t *testing.T) {
	tc := []struct {
		name  string
		input string
	}{
		{name: "auto id", input: "*"},
		{name: "auto sequence", input: "1-*"},
		{name: "not a number", input: "abc"},
	}
	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			_, err := storage.ParseRangeEnd(tt.input)

			require.Error(t, err)
		})
	}
}
