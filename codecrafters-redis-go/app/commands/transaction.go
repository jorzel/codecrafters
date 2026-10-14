package commands

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/codecrafters-io/redis-starter-go/app/protocol"
	"github.com/rs/zerolog"
)

type Transaction struct {
	queue    []protocol.Command
	IsActive bool
}

type TransactionsController interface {
	BeginTransaction(connID string) error
	EndTransaction(connID string) ([]protocol.Command, error)
	IsTransactionActive(connID string) bool
	QueueCommand(connID string, command protocol.Command) error
}

type DefaultTransactionsController struct {
	transactions map[string]Transaction // Map to track active transactions by connection ID
	mu           sync.Mutex
}

func NewDefaultTransactionsController() TransactionsController {
	return &DefaultTransactionsController{
		transactions: make(map[string]Transaction), // Initialize the map to store transactions
	}
}

func (c *DefaultTransactionsController) IsTransactionActive(connID string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.isTransactionActive(connID)
}

func (c *DefaultTransactionsController) isTransactionActive(connID string) bool {
	transaction, exists := c.transactions[connID]
	return exists && transaction.IsActive
}

func (c *DefaultTransactionsController) BeginTransaction(connID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.isTransactionActive(connID) {
		return fmt.Errorf("transaction already active for connection %s", connID)
	}
	c.transactions[connID] = Transaction{
		queue:    []protocol.Command{},
		IsActive: true,
	}
	return nil
}

func (c *DefaultTransactionsController) QueueCommand(connID string, command protocol.Command) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	transaction, exists := c.transactions[connID]
	if !exists || !transaction.IsActive {
		return fmt.Errorf("no active transaction for connection %s", connID)
	}
	transaction.queue = append(transaction.queue, command)
	c.transactions[connID] = transaction
	return nil
}

func (c *DefaultTransactionsController) EndTransaction(connID string) ([]protocol.Command, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	transaction, exists := c.transactions[connID]
	if !exists || !transaction.IsActive {
		return []protocol.Command{}, fmt.Errorf("no active transaction for connection %s", connID)
	}
	delete(c.transactions, connID)
	return transaction.queue, nil
}

type MultiHandler struct {
	transactionsController TransactionsController
}

func NewMultiHandler(
	transactionsController TransactionsController,
) *MultiHandler {
	return &MultiHandler{
		transactionsController: transactionsController,
	}
}

func (h *MultiHandler) Handle(
	ctx context.Context, conn net.Conn, command protocol.Command,
) (HandleResult, error) {
	msg, commandErr := h.apply(ctx, conn, command)
	err := sendMsg(ctx, conn, msg)
	return HandleResult{
		CommandError: commandErr,
	}, err
}

func (h *MultiHandler) apply(ctx context.Context, conn net.Conn, command protocol.Command) ([]byte, error) {
	if len(command.Args) > 0 {
		errMsg := "MULTI command requires no argument"
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}

	connID := conn.RemoteAddr().String()
	err := h.transactionsController.BeginTransaction(connID)
	if err != nil {
		errMsg := "Failed to begin transaction: " + err.Error()
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	return protocol.SimpleString("OK"), nil
}

type ExecHandler struct {
	transactionsController TransactionsController
	commandDispatcher      CommandDispatcher
}

func NewExecHandler(
	transactionsController TransactionsController, commandDispatcher CommandDispatcher,
) *ExecHandler {
	return &ExecHandler{
		transactionsController: transactionsController,
		commandDispatcher:      commandDispatcher,
	}
}

func (h *ExecHandler) Handle(
	ctx context.Context, conn net.Conn, command protocol.Command,
) (HandleResult, error) {
	msg, commandErr := h.apply(ctx, conn, command)
	err := sendMsg(ctx, conn, msg)
	return HandleResult{
		CommandError: commandErr,
	}, err
}

func (h *ExecHandler) apply(ctx context.Context, conn net.Conn, command protocol.Command) ([]byte, error) {
	if len(command.Args) > 0 {
		errMsg := "EXEC command requires no argument"
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}

	connID := conn.RemoteAddr().String()
	if !h.transactionsController.IsTransactionActive(connID) {
		errMsg := "EXEC without MULTI"
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	commands, err := h.transactionsController.EndTransaction(connID)
	if err != nil {
		errMsg := "Failed to end transaction: " + err.Error()
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	var results []string
	logger := zerolog.Ctx(ctx)
	for _, command := range commands {
		logger.Info().
			Str("remote_addr", conn.RemoteAddr().String()).
			Str("command", command.Name).
			Str("args", strings.Join(command.Args, ", ")).
			Msg("Executing transaction command")
		result, err := h.commandDispatcher.Dispatch(ctx, conn, command)
		if err != nil {
			errMsg := "Failed to execute command: " + err.Error()
			return protocol.Error(errMsg), fmt.Errorf(errMsg)
		}
		results = append(results, string(result.Msg))
	}
	logger.Info().
		Str("remote_addr", conn.RemoteAddr().String()).
		Interface("results", results).
		Str("resp", string(protocol.Array(results))).
		Msg("Executed transaction commands")
	return protocol.Array(results), nil
}

type DiscardHandler struct {
	transactionsController TransactionsController
}

func NewDiscardHandler(
	transactionsController TransactionsController,
) *DiscardHandler {
	return &DiscardHandler{
		transactionsController: transactionsController,
	}
}

func (h *DiscardHandler) Handle(
	ctx context.Context, conn net.Conn, command protocol.Command,
) (HandleResult, error) {
	msg, commandErr := h.apply(ctx, conn, command)
	err := sendMsg(ctx, conn, msg)
	return HandleResult{
		CommandError: commandErr,
	}, err
}

func (h *DiscardHandler) apply(ctx context.Context, conn net.Conn, command protocol.Command) ([]byte, error) {
	if len(command.Args) > 0 {
		errMsg := "DISCARD command requires no argument"
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}

	connID := conn.RemoteAddr().String()
	if !h.transactionsController.IsTransactionActive(connID) {
		errMsg := "DISCARD without MULTI"
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	_, err := h.transactionsController.EndTransaction(connID)
	if err != nil {
		errMsg := "Failed to discard transaction: " + err.Error()
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	return protocol.SimpleString("OK"), nil
}
