package skiplist_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/codecrafters-io/redis-starter-go/app/skiplist" // adjust import path
)

// assertOrder walks level-0 nodes and returns their members.
func assertOrder(t *testing.T, s *skiplist.Skiplist) []string {
	t.Helper()

	var got []string
	x := s.Header().ForwardNode(0)
	for x != nil {
		got = append(got, x.Member())
		x = x.ForwardNode(0)
	}
	return got
}

// checkOrder validates that skiplist has correct length and order.
func checkOrder(t *testing.T, s *skiplist.Skiplist, expected []string) {
	t.Helper()
	require.Equal(t, len(expected), s.Length())
	assert.Equal(t, expected, assertOrder(t, s))
}

func TestInsertSingle(t *testing.T) {
	sl := skiplist.New()
	sl.Insert(10, "A")

	checkOrder(t, sl, []string{"A"})
	assert.Equal(t, 1, sl.Length())
	assert.Equal(t, sl.GetRank(10, "A"), 0)
}

func TestInsertInOrder(t *testing.T) {
	sl := skiplist.New()
	sl.Insert(10, "A")
	sl.Insert(20, "B")

	checkOrder(t, sl, []string{"A", "B"})
	assert.Equal(t, 2, sl.Length())
	assert.Equal(t, sl.GetRank(10, "A"), 0)
	assert.Equal(t, sl.GetRank(20, "B"), 1)
}

func TestInsertOutOfOrder(t *testing.T) {
	sl := skiplist.New()
	sl.Insert(10, "A")
	sl.Insert(30, "C")
	sl.Insert(20, "B")

	checkOrder(t, sl, []string{"A", "B", "C"})
	assert.Equal(t, 3, sl.Length())
	assert.Equal(t, sl.GetRank(10, "A"), 0)
	assert.Equal(t, sl.GetRank(20, "B"), 1)
	assert.Equal(t, sl.GetRank(30, "C"), 2)
}

func TestInsertTieBreaking(t *testing.T) {
	sl := skiplist.New()
	sl.Insert(10, "A")
	sl.Insert(10, "B")  // same score, "A" < "B" => tie-break lexicographically
	sl.Insert(10, "AA") // between "A" and "B"

	checkOrder(t, sl, []string{"A", "AA", "B"})
	assert.Equal(t, 3, sl.Length())
	assert.Equal(t, sl.GetRank(10, "A"), 0)
	assert.Equal(t, sl.GetRank(10, "AA"), 1)
	assert.Equal(t, sl.GetRank(10, "B"), 2)
}

func TestSkiplistRemove(t *testing.T) {
	sl := skiplist.New()

	// Insert A, B, C
	sl.Insert(10, "A")
	sl.Insert(20, "B")
	sl.Insert(30, "C")
	require.Equal(t, 3, sl.Length())

	// Remove middle element
	ok := sl.Remove(20, "B")
	require.True(t, ok)
	require.Equal(t, 2, sl.Length())

	// Ensure A and C remain in order
	cur := sl.Header().ForwardNode(0)
	require.Equal(t, "A", cur.Member())
	cur = cur.ForwardNode(0)
	require.Equal(t, "C", cur.Member())
	assert.Equal(t, sl.GetRank(30, "C"), 1)

	// Remove non-existent
	ok = sl.Remove(999, "X")
	require.False(t, ok)
	require.Equal(t, 2, sl.Length())

	// Remove head
	ok = sl.Remove(10, "A")
	require.True(t, ok)
	require.Equal(t, 1, sl.Length())

	// Remove tail
	ok = sl.Remove(30, "C")
	require.True(t, ok)
	require.Equal(t, 0, sl.Length())
}

func TestSkiplistGetNodeByRank(t *testing.T) {
	sl := skiplist.New()
	sl.Insert(10, "A")
	sl.Insert(30, "C")

	node := sl.GetByRank(0)
	require.NotNil(t, node)
	assert.Equal(t, "A", node.Member())

	node = sl.GetByRank(1)
	require.NotNil(t, node)
	assert.Equal(t, "C", node.Member())

	sl.Insert(20, "B")

	node = sl.GetByRank(1)
	assert.Equal(t, "B", node.Member())
}
