package commands

import (
	"context"
	"net"

	"github.com/codecrafters-io/redis-starter-go/app/config"
	"github.com/codecrafters-io/redis-starter-go/app/protocol"
	"github.com/codecrafters-io/redis-starter-go/app/replication"
	"github.com/codecrafters-io/redis-starter-go/app/storage"
)

type HandleResult struct {
	CommandError error
	Msg          []byte
}

type Handler interface {
	Handle(ctx context.Context, conn net.Conn, command protocol.Command) (HandleResult, error)
}

type CommandDispatcher interface {
	Register(commandName string, handler Handler)
	Dispatch(ctx context.Context, conn net.Conn, command protocol.Command) (HandleResult, error)
}

type DefaultCommandDispatcher struct {
	handlers         map[string]Handler
	pubsubController PubSubController
}

func (d *DefaultCommandDispatcher) Register(commandName string, handler Handler) {
	d.handlers[commandName] = handler
}

func (d *DefaultCommandDispatcher) Dispatch(ctx context.Context, conn net.Conn, command protocol.Command) (HandleResult, error) {
	if d.pubsubController != nil {
		clientID := conn.RemoteAddr().String()
		err := d.pubsubController.CheckCommandAllowed(clientID, command)
		if err != nil {
			msg := protocol.Error(err.Error())
			sendMsg(ctx, conn, msg)
			return HandleResult{}, err
		}
	}
	handler, ok := d.handlers[command.Name]
	if !ok {
		msg := protocol.Error("unknown command '" + command.Name + "'")
		sendMsg(ctx, conn, msg)
		return HandleResult{}, nil
	}
	return handler.Handle(ctx, conn, command)
}

func NewMasterCommandDispatcher(
	replicasInfo *replication.ReplicasInfo,
	replicator Replicator,
	storage storage.Storage,
	config *config.Config,
) *DefaultCommandDispatcher {
	pubSubController := NewPubSubController()
	dispatcher := &DefaultCommandDispatcher{
		handlers:         make(map[string]Handler),
		pubsubController: pubSubController,
	}
	ackWaits := replication.NewAckWaitStore()
	listWaits := NewListWaitStore()
	transactionsController := NewDefaultTransactionsController()
	dispatcher.Register(protocol.WAIT, NewWaitHandler(replicasInfo, replicator, ackWaits))
	dispatcher.Register(protocol.GET, NewGetHandler(storage, transactionsController))
	dispatcher.Register(protocol.SET, NewSetHandler(storage, transactionsController))
	dispatcher.Register(protocol.PING, NewPingHandler(false, pubSubController))
	dispatcher.Register(protocol.ECHO, NewEchoHandler())
	dispatcher.Register(protocol.INFO, NewMasterInfoHandler(replicasInfo))
	dispatcher.Register(protocol.REPLCONF, NewMasterReplConfHandler(ackWaits))
	dispatcher.Register(protocol.PSYNC, NewPsyncHandler(replicasInfo))
	dispatcher.Register(protocol.CONFIG, NewConfigHandler(config))

	// List commands
	dispatcher.Register(protocol.RPUSH, NewRPushHandler(storage, listWaits))
	dispatcher.Register(protocol.LPUSH, NewLPushHandler(storage))
	dispatcher.Register(protocol.LRANGE, NewLRangeHandler(storage))
	dispatcher.Register(protocol.LLEN, NewLLenHandler(storage))
	dispatcher.Register(protocol.LPOP, NewLPopHandler(storage))
	dispatcher.Register(protocol.BLPOP, NewBLPopHandler(storage, listWaits))

	// Transaction commands
	dispatcher.Register(protocol.MULTI, NewMultiHandler(transactionsController))
	dispatcher.Register(protocol.INCR, NewIncrHandler(storage, transactionsController))
	execCommandDispatcher := NewExecCommandDispatcher(storage)
	dispatcher.Register(protocol.EXEC, NewExecHandler(transactionsController, execCommandDispatcher))
	dispatcher.Register(protocol.DISCARD, NewDiscardHandler(transactionsController))

	// Pub/Sub commands
	dispatcher.Register(protocol.SUBSCRIBE, NewSubscribeHandler(pubSubController))
	dispatcher.Register(protocol.UNSUBSCRIBE, NewUnsubscribeHandler(pubSubController))
	dispatcher.Register(protocol.PUBLISH, NewPublishHandler(pubSubController))
	dispatcher.Register(protocol.TYPE, NewTypeHandler(storage))

	// Sorted Set commands
	dispatcher.Register(protocol.ZADD, NewZAddHandler(storage))
	dispatcher.Register(protocol.ZRANK, NewZRankHandler(storage))
	dispatcher.Register(protocol.ZRANGE, NewZRangeHandler(storage))
	dispatcher.Register(protocol.ZCARD, NewZCardHandler(storage))
	dispatcher.Register(protocol.ZSCORE, NewZScoreHandler(storage))
	dispatcher.Register(protocol.ZREM, NewZRemHandler(storage))

	// Stream commands
	dispatcher.Register(protocol.XADD, NewXAddHandler(storage))
	return dispatcher
}

func NewReplicaCommandDispatcher(
	storage storage.Storage,
	config *config.Config,
) *DefaultCommandDispatcher {
	dispatcher := &DefaultCommandDispatcher{
		handlers: make(map[string]Handler),
	}
	transactionsController := NewDefaultTransactionsController()

	dispatcher.Register(protocol.GET, NewGetHandler(storage, transactionsController))
	dispatcher.Register(protocol.PING, NewPingHandler(false, nil))
	dispatcher.Register(protocol.ECHO, NewEchoHandler())
	dispatcher.Register(protocol.INFO, NewReplicaInfoHandler())
	dispatcher.Register(protocol.CONFIG, NewConfigHandler(config))
	return dispatcher
}

func NewReplicaReplicatedCommandDispatcher(
	masterInfo *replication.MasterInfo,
	storage storage.Storage,
) *DefaultCommandDispatcher {
	dispatcher := &DefaultCommandDispatcher{
		handlers: make(map[string]Handler),
	}
	dispatcher.Register(protocol.SET, NewReplicatedSetHandler(storage))
	dispatcher.Register(protocol.PING, NewPingHandler(true, nil))
	dispatcher.Register(protocol.REPLCONF, NewReplicaReplConfHandler(masterInfo))
	dispatcher.Register(protocol.FULLRESYNC, NewFullResyncHandler(masterInfo))
	return dispatcher
}

func NewExecCommandDispatcher(
	storage storage.Storage,
) *DefaultCommandDispatcher {
	dispatcher := &DefaultCommandDispatcher{
		handlers: make(map[string]Handler),
	}
	dispatcher.Register(protocol.SET, NewExecSetHandler(storage))
	dispatcher.Register(protocol.INCR, NewExecIncrHandler(storage))
	dispatcher.Register(protocol.GET, NewExecGetHandler(storage))
	return dispatcher
}
