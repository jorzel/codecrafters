package commands

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/protocol"
	"github.com/codecrafters-io/redis-starter-go/app/storage"
)

type ListWaitStore struct {
	Waits map[string][]chan bool
	mu    sync.Mutex
}

func NewListWaitStore() *ListWaitStore {
	return &ListWaitStore{
		Waits: make(map[string][]chan bool),
	}
}

func (s *ListWaitStore) AddWait(key string, wait chan bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Waits[key] = append(s.Waits[key], wait)
}

func (s *ListWaitStore) NotifyWaits(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if waits, ok := s.Waits[key]; ok {
		waits[0] <- true         // Notify the first wait channel
		s.Waits[key] = waits[1:] // Remove the notified wait
		if len(s.Waits[key]) == 0 {
			delete(s.Waits, key) // Clean up if no waits left
		}
	}
}

func (s *ListWaitStore) CleanWait(key string, ch chan bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if waits, ok := s.Waits[key]; ok {
		for i, wait := range waits {
			if wait == ch {
				s.Waits[key] = append(waits[:i], waits[i+1:]...)
				if len(s.Waits[key]) == 0 {
					delete(s.Waits, key) // Clean up if no waits left
				}
				close(ch) // Close the channel to avoid leaks
				return
			}
		}
	}
}

type RPushHandler struct {
	storage storage.Storage
	waits   *ListWaitStore
}

func NewRPushHandler(storage storage.Storage, waits *ListWaitStore) *RPushHandler {
	return &RPushHandler{
		storage: storage,
		waits:   waits,
	}
}

func (h *RPushHandler) Handle(
	ctx context.Context, conn net.Conn, command protocol.Command,
) (HandleResult, error) {
	msg, commandErr := h.apply(ctx, command)
	err := sendMsg(ctx, conn, msg)
	return HandleResult{
		CommandError: commandErr,
	}, err
}

func (h *RPushHandler) apply(ctx context.Context, command protocol.Command) ([]byte, error) {
	if len(command.Args) < 2 {
		errMsg := "RPush command requires at least 2 arguments"
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	key := command.Args[0]
	record, err := h.storage.GetList(key)
	if err != nil {
		errMsg := "Failed to get record: " + err.Error()
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}

	if record == nil {
		// If the key does not exist, create a new list
		values := make([]string, 0, len(command.Args)-1)
		record = &storage.ListRecord{
			Values: values,
		}
	}

	for _, value := range command.Args[1:] {
		record.Values = append(record.Values, value)
		if h.waits != nil {
			// Notify any waiting clients for this key
			h.waits.NotifyWaits(key)
		}

	}

	if err := h.storage.SetList(key, record); err != nil {
		errMsg := "Failed to set list record: " + err.Error()
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	return protocol.SimpleInteger(len(record.Values)), nil
}

type LPushHandler struct {
	storage storage.Storage
}

func NewLPushHandler(storage storage.Storage) *LPushHandler {
	return &LPushHandler{
		storage: storage,
	}
}

func (h *LPushHandler) Handle(
	ctx context.Context, conn net.Conn, command protocol.Command,
) (HandleResult, error) {
	msg, commandErr := h.apply(ctx, command)
	err := sendMsg(ctx, conn, msg)
	return HandleResult{
		CommandError: commandErr,
	}, err
}

func (h *LPushHandler) apply(_ context.Context, command protocol.Command) ([]byte, error) {
	if len(command.Args) < 2 {
		errMsg := "LPush command requires at least 2 arguments"
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	key := command.Args[0]
	record, err := h.storage.GetList(key)
	if err != nil {
		errMsg := "Failed to get record: " + err.Error()
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}

	if record == nil {
		// If the key does not exist, create a new list
		values := make([]string, 0, len(command.Args)-1)
		record = &storage.ListRecord{
			Values: values,
		}
	}

	for _, value := range command.Args[1:] {
		record.Values = append([]string{value}, record.Values...)
	}

	if err := h.storage.SetList(key, record); err != nil {
		errMsg := "Failed to set list record: " + err.Error()
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	return protocol.SimpleInteger(len(record.Values)), nil
}

type LRangeHandler struct {
	storage storage.Storage
}

func NewLRangeHandler(storage storage.Storage) *LRangeHandler {
	return &LRangeHandler{
		storage: storage,
	}
}

func (h *LRangeHandler) Handle(
	ctx context.Context, conn net.Conn, command protocol.Command,
) (HandleResult, error) {
	msg, commandErr := h.apply(ctx, command)
	err := sendMsg(ctx, conn, msg)
	return HandleResult{
		CommandError: commandErr,
	}, err
}

func (h *LRangeHandler) apply(_ context.Context, command protocol.Command) ([]byte, error) {
	if len(command.Args) < 3 {
		errMsg := "LRange command requires at least 3 arguments"
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}

	startIndex, err := strconv.Atoi(command.Args[1])
	if err != nil {
		errMsg := "Invalid start index: " + err.Error()
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	stopIndex, err := strconv.Atoi(command.Args[2])
	if err != nil {
		errMsg := "Invalid stop index: " + err.Error()
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	if startIndex > stopIndex && stopIndex >= 0 {
		return protocol.BulkArray([]string{}), nil
	}

	key := command.Args[0]
	record, err := h.storage.GetList(key)
	if err != nil {
		errMsg := "Failed to get record: " + err.Error()
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}

	if record == nil {
		return protocol.BulkArray([]string{}), nil
	}

	results := make([]string, 0)
	end := stopIndex
	if stopIndex < 0 {
		end = len(record.Values) + stopIndex // Adjust for negative indexing
	}
	start := startIndex
	if startIndex < 0 {
		start = max(len(record.Values)+startIndex, 0) // Adjust for negative indexing
	}
	for i := start; i <= end && i < len(record.Values); i++ {
		results = append(results, record.Values[i])
	}

	return protocol.BulkArray(results), nil
}

type LLenHandler struct {
	storage storage.Storage
}

func NewLLenHandler(storage storage.Storage) *LLenHandler {
	return &LLenHandler{
		storage: storage,
	}
}

func (h *LLenHandler) Handle(
	ctx context.Context, conn net.Conn, command protocol.Command,
) (HandleResult, error) {
	msg, commandErr := h.apply(ctx, command)
	err := sendMsg(ctx, conn, msg)
	return HandleResult{
		CommandError: commandErr,
	}, err
}

func (h *LLenHandler) apply(_ context.Context, command protocol.Command) ([]byte, error) {
	if len(command.Args) != 1 {
		errMsg := "LLen command requires 1 argument"
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}

	key := command.Args[0]
	record, err := h.storage.GetList(key)
	if err != nil {
		errMsg := "Failed to get record: " + err.Error()
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}

	if record == nil {
		return protocol.SimpleInteger(0), nil
	}

	return protocol.SimpleInteger(len(record.Values)), nil
}

type LPopHandler struct {
	storage storage.Storage
}

func NewLPopHandler(storage storage.Storage) *LPopHandler {
	return &LPopHandler{
		storage: storage,
	}
}

func (h *LPopHandler) Handle(
	ctx context.Context, conn net.Conn, command protocol.Command,
) (HandleResult, error) {
	msg, commandErr := h.apply(ctx, command)
	err := sendMsg(ctx, conn, msg)
	return HandleResult{
		CommandError: commandErr,
	}, err
}

func (h *LPopHandler) apply(_ context.Context, command protocol.Command) ([]byte, error) {
	if len(command.Args) > 2 {
		errMsg := "LPop command requires at most 2 arguments"
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}

	key := command.Args[0]
	record, err := h.storage.GetList(key)
	if err != nil {
		errMsg := "Failed to get record: " + err.Error()
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}

	if record == nil || len(record.Values) == 0 {
		return protocol.Nil(), nil
	}

	var toReturn []byte
	if len(command.Args) == 2 {
		count, err := strconv.Atoi(command.Args[1])
		if err != nil || count <= 0 {
			errMsg := "Invalid count value: " + err.Error()
			return protocol.Error(errMsg), fmt.Errorf(errMsg)
		}
		values := make([]string, 0, count)
		for i := 0; i < count && len(record.Values) > 0; i++ {
			value := record.Values[0]
			record.Values = record.Values[1:]
			values = append(values, value)
		}
		toReturn = protocol.BulkArray(values)
	} else {
		value := record.Values[0]
		record.Values = record.Values[1:]
		toReturn = protocol.BulkString(value)
	}

	if err := h.storage.SetList(key, record); err != nil {
		errMsg := "Failed to set list record: " + err.Error()
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	return toReturn, nil
}

type BLPopHandler struct {
	storage storage.Storage
	waits   *ListWaitStore
}

func NewBLPopHandler(storage storage.Storage, waits *ListWaitStore) *BLPopHandler {
	return &BLPopHandler{
		storage: storage,
		waits:   waits,
	}
}

func (h *BLPopHandler) Handle(
	ctx context.Context, conn net.Conn, command protocol.Command,
) (HandleResult, error) {
	msg, commandErr := h.apply(ctx, command)
	err := sendMsg(ctx, conn, msg)
	return HandleResult{
		CommandError: commandErr,
	}, err
}

func (h *BLPopHandler) apply(ctx context.Context, command protocol.Command) ([]byte, error) {
	if len(command.Args) > 3 {
		errMsg := "BLPop command requires at most 2 arguments"
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}

	key := command.Args[0]
	record, err := h.storage.GetList(key)
	if err != nil {
		errMsg := "Failed to get record: " + err.Error()
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	timeout, err := strconv.ParseFloat(command.Args[1], 64)
	if err != nil {
		errMsg := "Invalid timeout value: " + err.Error()
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	if timeout == 0 {
		timeout = 24 * 3600 // 24h - indefinite wait
	}
	var value string
	if record == nil || len(record.Values) == 0 {
		ch := make(chan bool)
		h.waits.AddWait(key, ch)
		select {
		case <-time.After(time.Duration(timeout * float64(time.Second))):
			h.waits.CleanWait(key, ch)
			return protocol.NilArray(), nil
		case <-ch:
			record, err = h.storage.GetList(key)
			if err != nil || record == nil || len(record.Values) == 0 {
				errMsg := "Failed to get record: "
				return protocol.Error(errMsg), fmt.Errorf(errMsg)
			}
			value = record.Values[0]
			record.Values = record.Values[1:]
		}
	} else {
		value = record.Values[0]
		record.Values = record.Values[1:]
		return protocol.BulkArray([]string{key, value}), nil
	}

	if err := h.storage.SetList(key, record); err != nil {
		errMsg := "Failed to set list record: " + err.Error()
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	return protocol.BulkArray([]string{key, value}), nil
}
