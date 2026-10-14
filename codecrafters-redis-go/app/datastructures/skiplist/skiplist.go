package skiplist

import (
	"math/rand"
	"time"
)

const (
	maxLevel = 32
	p        = 0.25
)

// node comparison: first by score, then by member
func less(aScore float64, aMember string, bScore float64, bMember string) bool {
	if aScore < bScore {
		return true
	}
	if aScore > bScore {
		return false
	}
	return aMember < bMember
}

type Skiplist struct {
	header *node
	level  int
	length int
	rnd    *rand.Rand
}

func New() *Skiplist {
	s := &Skiplist{
		header: newNode(maxLevel, 0, ""), // header with maxLevel arrays
		level:  1,
		rnd:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	return s
}

func (s *Skiplist) Header() *node {
	return s.header
}

func (s *Skiplist) Length() int {
	return s.length
}

// randomLevel generates a random level for a new node
// with probability p for each level
// the lower the p, the less likely to have high levels
func (s *Skiplist) randomLevel() int {
	lvl := 1
	for lvl < maxLevel && s.rnd.Float64() < p {
		lvl++
	}
	return lvl
}

// Insert inserts a new (score, member) pair into the skiplist.
func (s *Skiplist) Insert(score float64, member string) *node {
	insertionPoints, ranks := s.findInsertionPoints(score, member)

	lvl := s.pickLevelAndMaybeGrow(insertionPoints, ranks)

	x := s.createAndSpliceNode(score, member, lvl, insertionPoints, ranks)

	s.fixBackwardPointers(x, insertionPoints)

	s.length++
	return x
}

// findInsertionPoints walks the skiplist top-down to find where
// the new node should be inserted. Returns:
// - insertionPoints: the last node before the insertion point at each level
// - ranks: cumulative counts used for span calculation
func (s *Skiplist) findInsertionPoints(score float64, member string) ([]*node, []int) {
	insertionPoints := make([]*node, maxLevel)
	ranks := make([]int, maxLevel)

	x := s.header
	for i := s.level - 1; i >= 0; i-- {
		if i == s.level-1 {
			ranks[i] = 0
		} else {
			ranks[i] = ranks[i+1]
		}
		for x.forward[i] != nil && less(x.forward[i].score, x.forward[i].member, score, member) {
			ranks[i] += x.span[i]
			x = x.forward[i]
		}
		insertionPoints[i] = x
	}
	return insertionPoints, ranks
}

// pickLevelAndMaybeGrow chooses a random level and extends the skiplist if needed.
func (s *Skiplist) pickLevelAndMaybeGrow(insertionPoints []*node, ranks []int) int {
	lvl := s.randomLevel()
	if lvl > s.level {
		for i := s.level; i < lvl; i++ {
			insertionPoints[i] = s.header
			insertionPoints[i].span[i] = s.length
			ranks[i] = 0
		}
		s.level = lvl
	}
	return lvl
}

// createAndSpliceNode creates the new node and links it at each level.
// Also updates span values for affected nodes.
func (s *Skiplist) createAndSpliceNode(score float64, member string, lvl int, insertionPoints []*node, ranks []int) *node {
	x := newNode(lvl, score, member)

	for i := 0; i < lvl; i++ {
		// forward pointers
		x.forward[i] = insertionPoints[i].forward[i]
		insertionPoints[i].forward[i] = x

		// spans
		x.span[i] = insertionPoints[i].span[i] - (ranks[0] - ranks[i])
		insertionPoints[i].span[i] = (ranks[0] - ranks[i]) + 1
	}

	// untouched higher levels: increment span
	for i := lvl; i < s.level; i++ {
		insertionPoints[i].span[i]++
	}
	return x
}

// fixBackwardPointers updates level-0 backward links for doubly-linked traversal.
func (s *Skiplist) fixBackwardPointers(x *node, insertionPoints []*node) {
	if insertionPoints[0] == s.header {
		x.backward = nil
	} else {
		x.backward = insertionPoints[0]
	}
	if x.forward[0] != nil {
		x.forward[0].backward = x
	}
}

// Remove deletes the node with the given (score, member) from the skiplist.
// Returns true if the node was found and removed, false otherwise.
func (s *Skiplist) Remove(score float64, member string) bool {
	removalPoints := make([]*node, maxLevel)
	x := s.findRemovalPoints(score, member, removalPoints)
	if x == nil {
		return false // not found
	}

	s.unlinkNode(x, removalPoints)
	s.fixBackwardPointersAfterRemove(x)
	s.length--

	// shrink level if needed
	for s.level > 1 && s.header.forward[s.level-1] == nil {
		s.level--
	}
	return true
}

// findRemovalPoints locates the node to remove and records the nodes
// whose forward[i] pointers must be updated.
func (s *Skiplist) findRemovalPoints(score float64, member string, removalPoints []*node) *node {
	x := s.header
	for i := s.level - 1; i >= 0; i-- {
		for x.forward[i] != nil && less(x.forward[i].score, x.forward[i].member, score, member) {
			x = x.forward[i]
		}
		removalPoints[i] = x
	}

	// candidate node at level 0
	x = x.forward[0]
	if x != nil && x.score == score && x.member == member {
		return x
	}
	return nil
}

// unlinkNode adjusts forward pointers and spans to bypass x.
func (s *Skiplist) unlinkNode(x *node, removalPoints []*node) {
	for i := 0; i < s.level; i++ {
		if removalPoints[i].forward[i] == x {
			// bypass x
			removalPoints[i].span[i] += x.span[i] - 1
			removalPoints[i].forward[i] = x.forward[i]
		} else {
			// node not present at this level → just decrement span
			removalPoints[i].span[i]--
		}
	}
}

// fixBackwardPointersAfterRemove updates doubly-linked list pointers at level 0.
func (s *Skiplist) fixBackwardPointersAfterRemove(x *node) {
	if x.forward[0] != nil {
		x.forward[0].backward = x.backward
	} else {
		// last element: nothing to fix
	}
}

// GetRank returns the rank (0-based index) of the node with the given (score, member).
func (s *Skiplist) GetRank(score float64, member string) int {
	x := s.header
	rank := 0
	for i := s.level - 1; i >= 0; i-- {
		for x.forward[i] != nil && less(x.forward[i].score, x.forward[i].member, score, member) {
			rank += x.span[i]
			x = x.forward[i]
		}
	}
	x = x.forward[0]
	if x != nil && x.score == score && x.member == member {
		return rank
	}
	return -1
}

// GetByRank returns the node at the given rank (0-based index).
func (s *Skiplist) GetByRank(rank int) *node {
	if rank < 0 || rank >= s.length {
		return nil
	}
	x := s.header
	traversed := 0
	for i := s.level - 1; i >= 0; i-- {
		for x.forward[i] != nil && traversed+x.span[i] <= rank {
			traversed += x.span[i]
			x = x.forward[i]
		}
	}
	// x is the node before desired node
	return x.forward[0]
}
