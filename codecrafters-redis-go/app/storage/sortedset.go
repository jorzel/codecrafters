package storage

import "github.com/codecrafters-io/redis-starter-go/app/datastructures/skiplist"

type SortedSetRecord struct {
	dict map[string]float64
	sl   *skiplist.Skiplist
}

func NewSortedSetRecord() *SortedSetRecord {
	return &SortedSetRecord{sl: skiplist.New(), dict: make(map[string]float64)}
}

func (s *SortedSetRecord) Add(member string, score float64) bool {
	isNewMember := true
	if old, ok := s.dict[member]; ok {
		if old == score {
			return false
		} // no change
		s.sl.Remove(old, member)
		isNewMember = false
	}
	s.sl.Insert(score, member)
	s.dict[member] = score
	return isNewMember
}

func (s *SortedSetRecord) Remove(member string) bool {
	score, ok := s.dict[member]
	if !ok {
		return false
	}
	s.sl.Remove(score, member)
	delete(s.dict, member)
	return true
}

func (s *SortedSetRecord) Rank(member string) *int {
	score, ok := s.dict[member]
	if !ok {
		return nil
	}

	rank := s.sl.GetRank(score, member)
	return &rank
}

func (s *SortedSetRecord) Range(start, stop int) []string {
	members := []string{}
	size := s.sl.Length()
	if start < 0 {
		start = size + start
	}
	if stop < 0 {
		stop = size + stop
	}
	if start < 0 {
		start = 0
	}
	if stop < 0 {
		return []string{}
	}
	if start > stop || start >= size {
		return []string{}
	}
	if stop >= size {
		stop = size - 1
	}

	node := s.sl.GetByRank(start)
	for i := start; i <= stop && node != nil; i++ {
		members = append(members, node.Member())
		node = node.ForwardNode(0)
	}
	return members
}

func (s *SortedSetRecord) Card() int {
	return s.sl.Length()
}

func (s *SortedSetRecord) Score(member string) *float64 {
	score, ok := s.dict[member]
	if !ok {
		return nil
	}
	return &score
}
