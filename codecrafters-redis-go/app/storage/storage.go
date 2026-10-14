package storage

import (
	"fmt"
	"sync"
)

var _ Storage = (*DefaultStorage)(nil)

type Storage interface {
	Get(key string) (any, error)
	GetString(key string) (*StringRecord, error)
	SetString(key string, value *StringRecord) error
	GetList(key string) (*ListRecord, error)
	SetList(key string, value *ListRecord) error
	GetSortedSet(key string) (*SortedSetRecord, error)
	SetSortedSet(key string, value *SortedSetRecord) error
	GetStream(key string) (*StreamRecord, error)
	SetStream(key string, value *StreamRecord) error
}

type DefaultStorage struct {
	db sync.Map
}

func NewStorage() *DefaultStorage {
	return &DefaultStorage{
		db: sync.Map{},
	}
}

func (s *DefaultStorage) Get(key string) (any, error) {
	record, ok := s.db.Load(key)
	if !ok {
		return nil, nil
	}
	return record, nil
}

func (s *DefaultStorage) GetString(key string) (*StringRecord, error) {
	record, ok := s.db.Load(key)
	if !ok {
		return nil, nil
	}
	deserializedRecord, ok := record.(*StringRecord)
	if !ok {
		return nil, fmt.Errorf("failed to deserialize string record for key=%s, record=%v", key, record)
	}
	return deserializedRecord, nil
}

func (s *DefaultStorage) SetList(key string, value *ListRecord) error {
	s.db.Store(key, value)
	return nil
}

func (s *DefaultStorage) GetList(key string) (*ListRecord, error) {
	record, ok := s.db.Load(key)
	if !ok {
		return nil, nil
	}
	deserializedRecord, ok := record.(*ListRecord)
	if !ok {
		return nil, fmt.Errorf("failed to deserialize list record for key=%s, record=%v", key, record)
	}
	return deserializedRecord, nil
}

func (s *DefaultStorage) SetString(key string, value *StringRecord) error {
	s.db.Store(key, value)
	return nil
}

func (s *DefaultStorage) GetSortedSet(key string) (*SortedSetRecord, error) {
	record, ok := s.db.Load(key)
	if !ok {
		return nil, nil
	}
	deserializedRecord, ok := record.(*SortedSetRecord)
	if !ok {
		return nil, fmt.Errorf("failed to deserialize sorted set record for key=%s, record=%v", key, record)
	}
	return deserializedRecord, nil
}

func (s *DefaultStorage) SetSortedSet(key string, value *SortedSetRecord) error {
	s.db.Store(key, value)
	return nil
}

func (s *DefaultStorage) GetStream(key string) (*StreamRecord, error) {
	record, ok := s.db.Load(key)
	if !ok {
		return nil, nil
	}
	deserializedRecord, ok := record.(*StreamRecord)
	if !ok {
		return nil, fmt.Errorf("failed to deserialize stream record for key=%s, record=%v", key, record)
	}
	return deserializedRecord, nil
}

func (s *DefaultStorage) SetStream(key string, value *StreamRecord) error {
	s.db.Store(key, value)
	return nil
}
