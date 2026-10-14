package skiplist

type node struct {
	score    float64
	member   string
	forward  []*node // forward[i] points to next node at level i
	span     []int   // span[i] = #nodes between this node and forward[i]
	backward *node   // level 0 backward pointer for reverse iteration
}

func newNode(level int, score float64, member string) *node {
	return &node{
		score:    score,
		member:   member,
		forward:  make([]*node, level),
		span:     make([]int, level),
		backward: nil,
	}
}

func (n *node) ForwardNode(level int) *node {
	if level < 0 || level >= len(n.forward) {
		return nil
	}
	return n.forward[level]
}

func (n *node) Member() string {
	return n.member
}

func (n *node) Score() float64 {
	return n.score
}
