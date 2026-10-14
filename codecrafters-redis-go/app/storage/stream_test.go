package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStreamFailedAddItem(t *testing.T) {

	tc := []struct {
		name string
		id   string
	}{
		{
			name: "lower ms",
			id:   "1-0",
		},
		{
			name: "same ms, lower seq",
			id:   "2-0",
		},
	}
	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			stream := NewStreamRecord()
			stream.AddItem("2-1", map[string]string{"field1": "value1"})
			id, err := stream.AddItem(tt.id, map[string]string{"field1": "value1"})
			require.Error(t, err)
			assert.Equal(t, StreamID{}, id)
		})
	}
}

func TestStreamAddItem(t *testing.T) {

	tc := []struct {
		name     string
		id       string
		expected StreamID
	}{
		{
			name:     "higher sequence",
			id:       "2-2",
			expected: StreamID{Type: IDConcrete, Ms: 2, Seq: 2},
		},
		{
			name:     "auto sequence",
			id:       "2-*",
			expected: StreamID{Type: IDConcrete, Ms: 2, Seq: 2},
		},
	}
	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			stream := NewStreamRecord()
			stream.AddItem("2-1", map[string]string{"field1": "value1"})
			id, err := stream.AddItem(tt.id, map[string]string{"field1": "value1"})
			require.NoError(t, err)
			assert.Equal(t, tt.expected, id)
		})
	}
}
