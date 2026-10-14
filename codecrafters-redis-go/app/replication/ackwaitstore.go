package replication

import "sync"

type AckWaitStore struct {
	mu   sync.Mutex
	data map[string]AckWait
}

func NewAckWaitStore() *AckWaitStore {
	return &AckWaitStore{
		data: make(map[string]AckWait),
	}
}

func (s *AckWaitStore) Set(id string, ack AckWait) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[id] = ack
}

func (s *AckWaitStore) Get(id string) (AckWait, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ack, ok := s.data[id]
	return ack, ok
}

func (s *AckWaitStore) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, id)
}

func (s *AckWaitStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = make(map[string]AckWait)
}

func (s *AckWaitStore) CountSuccessful() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	acks := 0
	for _, ackWait := range s.data {
		if ackWait.IsSuccess {
			acks++
		}
	}
	return acks
}

type AckWait struct {
	IsSuccess      bool
	ExpectedOffset int
}
