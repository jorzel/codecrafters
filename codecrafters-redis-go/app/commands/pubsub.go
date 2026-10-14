package commands

import (
	"context"
	"fmt"
	"net"
	"slices"
	"strings"
	"sync"

	"github.com/codecrafters-io/redis-starter-go/app/protocol"
)

type PubSubController interface {
	SubscribeClient(conn net.Conn, channel string) (int, error)
	UnsubscribeClient(conn net.Conn, channel string) (int, error)
	GetSubscribedClients(channel string) ([]net.Conn, error)
	CheckCommandAllowed(connID string, command protocol.Command) error
	IsSubscribedMode(connID string) bool
}

type DefaultPubSubController struct {
	clients  map[string]map[string]struct{}   // Map to track channels by connection ID
	channels map[string]map[net.Conn]struct{} // Map to track clients by channel
	mu       sync.Mutex
}

func NewPubSubController() PubSubController {
	return &DefaultPubSubController{
		clients:  make(map[string]map[string]struct{}),   // Initialize the map to store channels
		channels: make(map[string]map[net.Conn]struct{}), // Initialize the map to store clients
	}
}

func (h *DefaultPubSubController) IsSubscribedMode(connID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.isSubscribedMode(connID)
}

func (h *DefaultPubSubController) isSubscribedMode(connID string) bool {
	if connID == "" {
		return false
	}
	_, exists := h.clients[connID]
	return exists
}

func (h *DefaultPubSubController) CheckCommandAllowed(connID string, command protocol.Command) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if connID == "" {
		return fmt.Errorf("connection ID cannot be empty")
	}

	if !h.isSubscribedMode(connID) {
		return nil
	}

	// Check if the command is allowed in the context of Pub/Sub
	if err := h.checkCommandAllowed(command); err != nil {
		return err
	}

	// If the command is allowed, return nil
	return nil
}

func (h *DefaultPubSubController) checkCommandAllowed(command protocol.Command) error {
	allowed_commands := []string{
		"SUBSCRIBE",
		"UNSUBSCRIBE",
		"PSUBSCRIBE",
		"PUNSUBSCRIBE",
		"PING",
		"QUIT",
		"RESET",
	}
	if !slices.Contains(allowed_commands, command.Name) {
		return fmt.Errorf(
			`Can't execute '%s': only (P|S)SUBSCRIBE / (P|S)UNSUBSCRIBE / PING / QUIT / RESET are allowed in this context`,
			strings.ToLower(command.Name),
		)
	}
	return nil
}

func (h *DefaultPubSubController) SubscribeClient(conn net.Conn, channel string) (int, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	connID := conn.RemoteAddr().String()
	if connID == "" || channel == "" {
		return 0, fmt.Errorf("connection ID and channel cannot be empty")
	}

	// Add the channel to the connection's list of channels
	if _, exists := h.clients[connID]; !exists {
		h.clients[connID] = make(map[string]struct{})
	}
	h.clients[connID][channel] = struct{}{}
	if _, exists := h.channels[channel]; !exists {
		h.channels[channel] = make(map[net.Conn]struct{})
	}
	h.channels[channel][conn] = struct{}{}
	return len(h.clients[connID]), nil
}

func (h *DefaultPubSubController) UnsubscribeClient(conn net.Conn, channel string) (int, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	connID := conn.RemoteAddr().String()
	if connID == "" || channel == "" {
		return 0, fmt.Errorf("connection ID and channel cannot be empty")
	}
	// Check if the client is subscribed to the channel
	_, exists := h.clients[connID]
	if !exists {
		return 0, fmt.Errorf("client %s is not subscribed to channel %s", connID, channel)
	}

	delete(h.clients[connID], channel)

	_, exists = h.channels[channel]
	if exists {
		delete(h.channels[channel], conn)
	}

	if len(h.channels[channel]) == 0 {
		delete(h.channels, channel)
	}

	if len(h.clients[connID]) == 0 {
		delete(h.clients, connID)
		return 0, nil
	}
	return len(h.clients[connID]), nil
}

func (h *DefaultPubSubController) GetSubscribedClients(channel string) ([]net.Conn, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if channel == "" {
		return nil, fmt.Errorf("channel cannot be empty")
	}

	clientsSet, exists := h.channels[channel]
	if !exists {
		return []net.Conn{}, nil
	}

	var clients []net.Conn
	for client := range clientsSet {
		clients = append(clients, client)
	}
	return clients, nil
}

type SubscribeHandler struct {
	pubsubController PubSubController
}

func NewSubscribeHandler(pubsubController PubSubController) *SubscribeHandler {
	return &SubscribeHandler{
		pubsubController: pubsubController,
	}
}

func (h *SubscribeHandler) Handle(
	ctx context.Context, conn net.Conn, command protocol.Command,
) (HandleResult, error) {
	msg, commandErr := h.apply(conn, command)
	err := sendMsg(ctx, conn, msg)
	return HandleResult{
		CommandError: commandErr,
		Msg:          msg,
	}, err
}

func (h *SubscribeHandler) apply(conn net.Conn, command protocol.Command) ([]byte, error) {
	if len(command.Args) != 1 {
		errMsg := "SUBSCRIBE command requires exactly 1 argument"
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	channel := command.Args[0]
	clientID := conn.RemoteAddr().String()
	clientChannelsCount, err := h.pubsubController.SubscribeClient(conn, channel)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to add channel %s for client %s: %v", channel, clientID, err)
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	return protocol.Array([]string{
		string(protocol.BulkString("subscribe")),
		string(protocol.BulkString(channel)),
		string(protocol.SimpleInteger(clientChannelsCount)),
	}), nil
}

type PublishHandler struct {
	pubsubController PubSubController
}

func NewPublishHandler(pubsubController PubSubController) *PublishHandler {
	return &PublishHandler{
		pubsubController: pubsubController,
	}
}

func (h *PublishHandler) Handle(
	ctx context.Context, conn net.Conn, command protocol.Command,
) (HandleResult, error) {
	msg, commandErr := h.apply(ctx, conn, command)
	err := sendMsg(ctx, conn, msg)
	return HandleResult{
		CommandError: commandErr,
		Msg:          msg,
	}, err
}

func (h *PublishHandler) apply(ctx context.Context, conn net.Conn, command protocol.Command) ([]byte, error) {
	if len(command.Args) != 2 {
		errMsg := "PUBLISH command requires exactly 2 argument"
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	channel := command.Args[0]
	content := command.Args[1]
	clients, err := h.pubsubController.GetSubscribedClients(channel)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to get subscribed clients for channel %s: %v", channel, err)
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	for _, client := range clients {
		if client == nil {
			continue // Skip nil clients
		}
		msg := protocol.BulkArray([]string{
			"message",
			channel,
			content,
		})
		if err := sendMsg(ctx, client, msg); err != nil {
			return protocol.Error("Failed to publish message"), fmt.Errorf("failed to send message to client: %v", err)
		}
	}
	return protocol.SimpleInteger(len(clients)), nil

}

type UnsubscribeHandler struct {
	pubsubController PubSubController
}

func NewUnsubscribeHandler(pubsubController PubSubController) *UnsubscribeHandler {
	return &UnsubscribeHandler{
		pubsubController: pubsubController,
	}
}

func (h *UnsubscribeHandler) Handle(
	ctx context.Context, conn net.Conn, command protocol.Command,
) (HandleResult, error) {
	msg, commandErr := h.apply(conn, command)
	err := sendMsg(ctx, conn, msg)
	return HandleResult{
		CommandError: commandErr,
		Msg:          msg,
	}, err
}

func (h *UnsubscribeHandler) apply(conn net.Conn, command protocol.Command) ([]byte, error) {
	if len(command.Args) != 1 {
		errMsg := "UNSUBSCRIBE command requires exactly 1 argument"
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	channel := command.Args[0]
	clientID := conn.RemoteAddr().String()
	clientChannelsCount, err := h.pubsubController.UnsubscribeClient(conn, channel)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to unsubscribe channel %s for client %s: %v", channel, clientID, err)
		return protocol.Error(errMsg), fmt.Errorf(errMsg)
	}
	return protocol.Array([]string{
		string(protocol.BulkString("unsubscribe")),
		string(protocol.BulkString(channel)),
		string(protocol.SimpleInteger(clientChannelsCount)),
	}), nil
}
