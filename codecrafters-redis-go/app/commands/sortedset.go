package commands

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/codecrafters-io/redis-starter-go/app/protocol"
	"github.com/codecrafters-io/redis-starter-go/app/storage"
)

type ZAddHandler struct {
	storage storage.Storage
}

func NewZAddHandler(storage storage.Storage) *ZAddHandler {
	return &ZAddHandler{
		storage: storage,
	}
}

func (h *ZAddHandler) Handle(
	ctx context.Context, conn net.Conn, command protocol.Command,
) (HandleResult, error) {
	msg, commandErr := h.apply(command)
	err := sendMsg(ctx, conn, msg)
	return HandleResult{
		CommandError: commandErr,
		Msg:          msg,
	}, err
}

func (h *ZAddHandler) apply(command protocol.Command) ([]byte, error) {
	// check whether number of argument is uneven
	if len(command.Args) < 3 || len(command.Args)%2 == 0 {
		errMsg := "ZADD command requires at least 3 arguments (key, score, member) and an even number of arguments"
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	key := command.Args[0]
	record, err := h.storage.GetSortedSet(key)
	if err != nil {
		errMsg := "Failed to get record: " + err.Error()
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	if record == nil {
		record = storage.NewSortedSetRecord()
	}
	addedMembers := 0
	for i := 1; i < len(command.Args); i += 2 {
		member := command.Args[i+1]
		score, err := strconv.ParseFloat(command.Args[i], 64)
		if err != nil {
			errMsg := fmt.Sprintf("Invalid score '%s': %v", command.Args[i], err)
			return protocol.Error(errMsg), fmt.Errorf(errMsg)
		}
		wasAdded := record.Add(member, score)
		if wasAdded {
			addedMembers++
		}
	}
	if err := h.storage.SetSortedSet(key, record); err != nil {
		errMsg := "Failed to set sorted set record: " + err.Error()
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	return protocol.SimpleInteger(addedMembers), nil
}

type ZRankHandler struct {
	storage storage.Storage
}

func NewZRankHandler(storage storage.Storage) *ZRankHandler {
	return &ZRankHandler{
		storage: storage,
	}
}

func (h *ZRankHandler) Handle(
	ctx context.Context, conn net.Conn, command protocol.Command,
) (HandleResult, error) {
	msg, commandErr := h.apply(command)
	err := sendMsg(ctx, conn, msg)
	return HandleResult{
		CommandError: commandErr,
		Msg:          msg,
	}, err
}

func (h *ZRankHandler) apply(command protocol.Command) ([]byte, error) {
	if len(command.Args) != 2 {
		errMsg := "ZRANK command requires at least 2 arguments (key, member)"
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	key := command.Args[0]
	record, err := h.storage.GetSortedSet(key)
	if err != nil {
		errMsg := "Failed to get record: " + err.Error()
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	if record == nil {
		return protocol.Nil(), nil
	}
	rank := record.Rank(command.Args[1])
	if rank == nil {
		return protocol.Nil(), nil
	}
	return protocol.SimpleInteger(*rank), nil
}

type ZRangeHandler struct {
	storage storage.Storage
}

func NewZRangeHandler(storage storage.Storage) *ZRangeHandler {
	return &ZRangeHandler{
		storage: storage,
	}
}

func (h *ZRangeHandler) Handle(
	ctx context.Context, conn net.Conn, command protocol.Command,
) (HandleResult, error) {
	msg, commandErr := h.apply(command)
	err := sendMsg(ctx, conn, msg)
	return HandleResult{
		CommandError: commandErr,
		Msg:          msg,
	}, err
}

func (h *ZRangeHandler) apply(command protocol.Command) ([]byte, error) {
	if len(command.Args) != 3 {
		errMsg := "ZRANGE command requires 3 arguments (key, start, end)"
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	key := command.Args[0]
	record, err := h.storage.GetSortedSet(key)
	if err != nil {
		errMsg := "Failed to get record: " + err.Error()
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	if record == nil {
		return protocol.BulkArray([]string{}), nil
	}
	start, err := strconv.Atoi(command.Args[1])
	if err != nil {
		errMsg := fmt.Sprintf("Invalid start index '%s': %v", command.Args[1], err)
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	end, err := strconv.Atoi(command.Args[2])
	if err != nil {
		errMsg := fmt.Sprintf("Invalid end index '%s': %v", command.Args[2], err)
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	results := record.Range(start, end)
	return protocol.BulkArray(results), nil
}

type ZCardHandler struct {
	storage storage.Storage
}

func NewZCardHandler(storage storage.Storage) *ZCardHandler {
	return &ZCardHandler{
		storage: storage,
	}
}

func (h *ZCardHandler) Handle(
	ctx context.Context, conn net.Conn, command protocol.Command,
) (HandleResult, error) {
	msg, commandErr := h.apply(command)
	err := sendMsg(ctx, conn, msg)
	return HandleResult{
		CommandError: commandErr,
		Msg:          msg,
	}, err
}

func (h *ZCardHandler) apply(command protocol.Command) ([]byte, error) {
	if len(command.Args) != 1 {
		errMsg := "ZCARD command requires 1 argument key"
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	key := command.Args[0]
	record, err := h.storage.GetSortedSet(key)
	if err != nil {
		errMsg := "Failed to get record: " + err.Error()
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	if record == nil {
		return protocol.SimpleInteger(0), nil
	}
	return protocol.SimpleInteger(record.Card()), nil
}

type ZScoreHandler struct {
	storage storage.Storage
}

func NewZScoreHandler(storage storage.Storage) *ZScoreHandler {
	return &ZScoreHandler{
		storage: storage,
	}
}

func (h *ZScoreHandler) Handle(
	ctx context.Context, conn net.Conn, command protocol.Command,
) (HandleResult, error) {
	msg, commandErr := h.apply(command)
	err := sendMsg(ctx, conn, msg)
	return HandleResult{
		CommandError: commandErr,
		Msg:          msg,
	}, err
}

func (h *ZScoreHandler) apply(command protocol.Command) ([]byte, error) {
	if len(command.Args) != 2 {
		errMsg := "ZSCORE command requires 1 argument key"
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	key := command.Args[0]
	record, err := h.storage.GetSortedSet(key)
	if err != nil {
		errMsg := "Failed to get record: " + err.Error()
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	if record == nil {
		return protocol.Nil(), nil
	}
	member := command.Args[1]
	score := record.Score(member)
	if score == nil {
		return protocol.Nil(), nil
	}
	return protocol.BulkString(strconv.FormatFloat(*score, 'g', -1, 64)), nil
}

type ZRemHandler struct {
	storage storage.Storage
}

func NewZRemHandler(storage storage.Storage) *ZRemHandler {
	return &ZRemHandler{
		storage: storage,
	}
}

func (h *ZRemHandler) Handle(
	ctx context.Context, conn net.Conn, command protocol.Command,
) (HandleResult, error) {
	msg, commandErr := h.apply(command)
	err := sendMsg(ctx, conn, msg)
	return HandleResult{
		CommandError: commandErr,
		Msg:          msg,
	}, err
}

func (h *ZRemHandler) apply(command protocol.Command) ([]byte, error) {
	key := command.Args[0]
	record, err := h.storage.GetSortedSet(key)
	if err != nil {
		errMsg := "Failed to get record: " + err.Error()
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	if record == nil {
		return protocol.Nil(), nil
	}
	removedMembers := 0
	for i := 1; i < len(command.Args); i++ {
		member := command.Args[i]

		wasRemoved := record.Remove(member)
		if wasRemoved {
			removedMembers++
		}
	}
	if err := h.storage.SetSortedSet(key, record); err != nil {
		errMsg := "Failed to set sorted set record: " + err.Error()
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	return protocol.SimpleInteger(removedMembers), nil
}
