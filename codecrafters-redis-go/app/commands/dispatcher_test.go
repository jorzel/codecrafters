package commands

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/codecrafters-io/redis-starter-go/app/config"
	"github.com/codecrafters-io/redis-starter-go/app/protocol"
	"github.com/codecrafters-io/redis-starter-go/app/replication"
	"github.com/codecrafters-io/redis-starter-go/app/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMasterDispatcherHandleWait(t *testing.T) {
	command := protocol.Command{
		Name: "WAIT",
		Args: []string{"0", "6000"},
	}
	dispatcher, conn := setupMasterDispatcher(t)

	_, err := dispatcher.Dispatch(context.Background(), conn, command)

	require.NoError(t, err, "Expected no error when handling WAIT command")
	require.Len(t, conn.writes, 1, "Expected single writes to the connection")
	assert.Equal(t,
		string([]byte(":0\r\n")),
		string(conn.writes[0]),
		"Expected WAIT response for master",
	)
}

func TestMasterDispatcherHandleGet(t *testing.T) {
	getCommand := protocol.Command{
		Name: "GET",
		Args: []string{"key"},
	}
	dispatcher, conn := setupMasterDispatcher(t)
	// First, we need to set the value for the key
	setCommand := protocol.Command{
		Name: "SET",
		Args: []string{"key", "value"},
	}

	dispatcher.Dispatch(context.Background(), conn, setCommand)

	_, err := dispatcher.Dispatch(context.Background(), conn, getCommand)

	require.NoError(t, err, "Expected no error when handling GET command")
	require.Len(t, conn.writes, 2, "Expected one write to the connection")
	assert.Equal(
		t,
		string([]byte("$5\r\nvalue\r\n")),
		string(conn.writes[1]),
		"Expected GET command to return the value for the key",
	)
}

func TestMasterDispatcherHandleGetExpiredRecord(t *testing.T) {
	getCommand := protocol.Command{
		Name: "GET",
		Args: []string{"key"},
	}

	dispatcher, conn := setupMasterDispatcher(t)
	// First, we need to set the value for the key
	setCommand := protocol.Command{
		Name: "SET",
		Args: []string{"key", "value", "px", "1"},
	}
	dispatcher.Dispatch(context.Background(), conn, setCommand)
	time.Sleep(2 * time.Millisecond) // Wait for the key to expire

	_, err := dispatcher.Dispatch(context.Background(), conn, getCommand)

	require.NoError(t, err, "Expected no error when handling GET command for expired key")
	require.Len(t, conn.writes, 2, "Expected two writes to the connection")
	assert.Equal(t, string([]byte("$-1\r\n")), string(conn.writes[1]), "Expected GET command to return nil")
}

func TestMasterDispatcherHandleSet(t *testing.T) {
	command := protocol.Command{
		Name: "SET",
		Args: []string{"key", "value"},
	}
	dispatcher, conn := setupMasterDispatcher(t)

	_, err := dispatcher.Dispatch(context.Background(), conn, command)

	require.NoError(t, err, "Expected no error when handling SET command")
	require.Len(t, conn.writes, 1, "Expected one write to the connection")
	assert.Equal(t, string([]byte("+OK\r\n")), string(conn.writes[0]), "Expected SET command to return OK")
}

func TestMasterDispatcherHandleSetWithPX(t *testing.T) {
	command := protocol.Command{
		Name: "SET",
		Args: []string{"key", "value", "px", "1000"},
	}
	dispatcher, conn := setupMasterDispatcher(t)

	_, err := dispatcher.Dispatch(context.Background(), conn, command)

	require.NoError(t, err, "Expected no error when handling SET command with PX")
	require.Len(t, conn.writes, 1, "Expected one write to the connection")
	assert.Equal(t, string([]byte("+OK\r\n")), string(conn.writes[0]), "Expected SET command to return OK")
}

func TestMasterDispatcherHandlePing(t *testing.T) {
	command := protocol.Command{
		Name: "PING",
	}
	dispatcher, conn := setupMasterDispatcher(t)

	_, err := dispatcher.Dispatch(context.Background(), conn, command)

	require.NoError(t, err, "Expected no error when handling PING command")
	require.Len(t, conn.writes, 1, "Expected one write to the connection")
	assert.Equal(t, string([]byte("+PONG\r\n")), string(conn.writes[0]), "Expected PING command to return PONG")
}

func TestMasterDispatcherHandleEcho(t *testing.T) {
	command := protocol.Command{
		Name: "ECHO",
		Args: []string{"as"},
	}
	dispatcher, conn := setupMasterDispatcher(t)

	_, err := dispatcher.Dispatch(context.Background(), conn, command)

	require.NoError(t, err, "Expected no error when handling ECHO command")
	require.Len(t, conn.writes, 1, "Expected one write to the connection")
	assert.Equal(t, string([]byte("$2\r\nas\r\n")), string(conn.writes[0]), "Expected ECHO command to return the same string")
}

func TestMasterDispatcherHandleInfo(t *testing.T) {
	command := protocol.Command{
		Name: "INFO",
	}
	dispatcher, conn := setupMasterDispatcher(t)

	_, err := dispatcher.Dispatch(context.Background(), conn, command)

	require.NoError(t, err, "Expected no error when handling INFO command")
	require.Len(t, conn.writes, 1, "Expected one write to the connection")
	assert.Contains(t, string(conn.writes[0]), "#Replication\r\n", "Expected INFO command to include replication section")
	require.Contains(t, string(conn.writes[0]), "role:master\r\n", "Expected INFO command to include role as master")
	require.Contains(t, string(conn.writes[0]), "master_replid:testreplid\r\n", "Expected INFO command to include master_replid")
	require.Contains(t, string(conn.writes[0]), "master_repl_offset:0\r\n", "Expected INFO command to include master_repl_offset")
}

func TestDispatcherHandleReplConfListeningPort(t *testing.T) {
	command := protocol.Command{
		Name: "REPLCONF",
		Args: []string{"listening-port", "6379"},
	}
	dispatcher, conn := setupMasterDispatcher(t)

	_, err := dispatcher.Dispatch(context.Background(), conn, command)

	require.NoError(t, err, "Expected no error when handling REPLCONF command")
	require.Len(t, conn.writes, 1, "Expected one write to the connection")
	assert.Equal(t, string([]byte("+OK\r\n")), string(conn.writes[0]), "Expected REPLCONF command to return OK")
}

func TestMasterDispatcherHandleReplConfCapa(t *testing.T) {
	command := protocol.Command{
		Name: "REPLCONF",
		Args: []string{"capa", "psync2"},
	}
	dispatcher, conn := setupMasterDispatcher(t)

	_, err := dispatcher.Dispatch(context.Background(), conn, command)

	require.NoError(t, err, "Expected no error when handling REPLCONF command")
	require.Len(t, conn.writes, 1, "Expected one write to the connection")
	assert.Equal(t, string([]byte("+OK\r\n")), string(conn.writes[0]), "Expected REPLCONF command to return OK")
}

func TestReplicaDispatcherHandleReplConfGetAckWithOffset(t *testing.T) {
	command := protocol.Command{
		Name: "REPLCONF",
		Args: []string{"getack", "*"},
	}
	info := &replication.MasterInfo{
		ReplID: "testreplid",
		Conn:   &MockConn{},
	}
	info.AddReplicaOffset(5)
	dispatcher, conn := setupReplicatedCommandDispatcher(t, info)

	_, err := dispatcher.Dispatch(context.Background(), conn, command)

	require.NoError(t, err, "Expected no error when handling REPLCONF getack command")
	require.Len(t, conn.writes, 1, "Expected one write to the connection")
	assert.Equal(t,
		string([]byte("*3\r\n$8\r\nREPLCONF\r\n$3\r\nACK\r\n$1\r\n5\r\n")),
		string(conn.writes[0]),
		"Expected REPLCONF getack command to return nil",
	)
}

func TestMasterDispatcherHandleReplConfInvalid(t *testing.T) {
	command := protocol.Command{
		Name: "REPLCONF",
		Args: []string{"invalid", "argument"},
	}

	dispatcher, conn := setupMasterDispatcher(t)

	_, err := dispatcher.Dispatch(context.Background(), conn, command)

	require.NoError(t, err, "Expected REPLCONF command to return an error response")
	require.Len(t, conn.writes, 1, "Expected one write to the connection")
	assert.Equal(t,
		string([]byte("-ERR REPLCONF command only supports 'listening-port <PORT>', ack, getack, and 'capa psync2' arguments, got: invalid, argument\r\n")),
		string(conn.writes[0]),
		"Expected REPLCONF command to return an error",
	)
}

func TestMasterDispatcherHandlePSync(t *testing.T) {
	command := protocol.Command{
		Name: "PSYNC",
		Args: []string{"?", "-1"},
	}

	dispatcher, conn := setupMasterDispatcher(t)
	_, err := dispatcher.Dispatch(context.Background(), conn, command)

	require.NoError(t, err, "Expected no error when handling PSYNC command")
	require.Len(t, conn.writes, 2, "Expected two writes to the connection")
	assert.Equal(t,
		string([]byte("+FULLRESYNC testreplid 0\r\n")),
		string(conn.writes[0]),
		"Expected PSYNC command to return FULLRESYNC response for master",
	)
	assert.NotNil(t, conn.writes[1], "Expected second write to contain DB file content")
}

func TestMasterDispatcherHandleConfigGetDir(t *testing.T) {
	command := protocol.Command{
		Name: "CONFIG",
		Args: []string{"GET", "dir"},
	}

	dispatcher, conn := setupMasterDispatcher(t)

	_, err := dispatcher.Dispatch(context.Background(), conn, command)

	require.NoError(t, err, "Expected no error when handling CONFIG GET command")
	require.Len(t, conn.writes, 1, "Expected one write to the connection")
	assert.Equal(t,
		string([]byte("*2\r\n$3\r\ndir\r\n$8\r\n/tmp/dir\r\n")),
		string(conn.writes[0]),
		"Expected CONFIG GET command to return the directory",
	)
}

func TestMasterDispatcherHandlerRPush(t *testing.T) {
	command := protocol.Command{
		Name: "RPUSH",
		Args: []string{"list", "value"},
	}

	dispatcher, conn := setupMasterDispatcher(t)

	_, err := dispatcher.Dispatch(context.Background(), conn, command) // should create a list with sngle element
	require.NoError(t, err, "Expected no error when handling RPUSH command")
	_, err = dispatcher.Dispatch(context.Background(), conn, command) // should append
	require.NoError(t, err, "Expected no error when handling RPUSH command")
	require.Len(t, conn.writes, 2, "Expected two writes to the connection")

	assert.Equal(t,
		string([]byte(":1\r\n")),
		string(conn.writes[0]),
		"Expected RPUSH command to return the length of the list after adding the value",
	)
	assert.Equal(t,
		string([]byte(":2\r\n")),
		string(conn.writes[1]),
		"Expected RPUSH command to return the length of the list after adding the value",
	)
}

func TestMasterDispatcherHandlerRPushMultipleElements(t *testing.T) {
	command := protocol.Command{
		Name: "RPUSH",
		Args: []string{"list", "value", "value2"},
	}

	dispatcher, conn := setupMasterDispatcher(t)

	_, err := dispatcher.Dispatch(context.Background(), conn, command) // should create a list with sngle element
	require.NoError(t, err, "Expected no error when handling RPUSH command")

	assert.Equal(t,
		string([]byte(":2\r\n")),
		string(conn.writes[0]),
		"Expected RPUSH command to return the length of the list after adding the value",
	)
}

func TestMasterDispatcherHandlerLRange(t *testing.T) {
	//setup a list with multiple elements
	pushCommand := protocol.Command{
		Name: "RPUSH",
		Args: []string{"list", "value1", "value2", "value3"},
	}
	dispatcher, conn := setupMasterDispatcher(t)
	_, err := dispatcher.Dispatch(context.Background(), conn, pushCommand)
	require.NoError(t, err, "Expected no error when handling RPUSH command")

	command := protocol.Command{
		Name: "LRANGE",
		Args: []string{"list", "0", "2"},
	}

	_, err = dispatcher.Dispatch(context.Background(), conn, command) // should return empty list
	require.NoError(t, err, "Expected no error when handling LRANGE command")
	require.Len(t, conn.writes, 2, "Expected one write to the connection")
	assert.Equal(t,
		string([]byte("*3\r\n$6\r\nvalue1\r\n$6\r\nvalue2\r\n$6\r\nvalue3\r\n")),
		string(conn.writes[1]),
		"Expected LRANGE command to return an empty list",
	)
}

func TestMasterDispatcherHandlerLRangeNegitiveIndex(t *testing.T) {
	//setup a list with multiple elements
	pushCommand := protocol.Command{
		Name: "RPUSH",
		Args: []string{"mango", "grape", "apple", "pineapple", "pear", "blueberry"},
	}
	dispatcher, conn := setupMasterDispatcher(t)
	_, err := dispatcher.Dispatch(context.Background(), conn, pushCommand)
	require.NoError(t, err, "Expected no error when handling RPUSH command")

	tt := []struct {
		startIndex string
		endIndex   string
		expected   string
	}{
		{"-3", "-1", "*3\r\n$9\r\npineapple\r\n$4\r\npear\r\n$9\r\nblueberry\r\n"},
		{"-1", "-1", "*1\r\n$9\r\nblueberry\r\n"},
		{"0", "-2", "*4\r\n$5\r\ngrape\r\n$5\r\napple\r\n$9\r\npineapple\r\n$4\r\npear\r\n"},
	}
	for i, tc := range tt {
		command := protocol.Command{
			Name: "LRANGE",
			Args: []string{"mango", tc.startIndex, tc.endIndex},
		}

		_, err = dispatcher.Dispatch(context.Background(), conn, command) // should return empty list
		require.NoError(t, err, "Expected no error when handling LRANGE command")
		require.Len(t, conn.writes, i+2, "Expected one write to the connection")
		assert.Equal(t,
			string([]byte(tc.expected)),
			string(conn.writes[i+1]),
			"Expected LRANGE command to return the expected elements for startIndex=%s, endIndex=%s",
		)
	}
}

func TestMasterDispatcherHandlerLPushMultipleElements(t *testing.T) {
	command := protocol.Command{
		Name: "LPUSH",
		Args: []string{"list", "value", "value2"},
	}

	dispatcher, conn := setupMasterDispatcher(t)

	_, err := dispatcher.Dispatch(context.Background(), conn, command) // should create a list with sngle element
	require.NoError(t, err, "Expected no error when handling LPUSH command")

	assert.Equal(t,
		string([]byte(":2\r\n")),
		string(conn.writes[0]),
		"Expected LPUSH command to return the length of the list after adding the value",
	)

	lrangeCommand := protocol.Command{
		Name: "LRANGE",
		Args: []string{"list", "0", "-1"},
	}
	_, err = dispatcher.Dispatch(context.Background(), conn, lrangeCommand) // should return the list with two elements
	require.NoError(t, err, "Expected no error when handling LRANGE command")

	require.Len(t, conn.writes, 2, "Expected two writes to the connection")
	assert.Equal(t,
		string([]byte("*2\r\n$6\r\nvalue2\r\n$5\r\nvalue\r\n")),
		string(conn.writes[1]),
		"Expected LRANGE command to return the list with two elements",
	)
}

func TestMasterDispatcherHandleLlen(t *testing.T) {
	// Setup a list with multiple elements
	pushCommand := protocol.Command{
		Name: "RPUSH",
		Args: []string{"list", "value1", "value2", "value3"},
	}
	dispatcher, conn := setupMasterDispatcher(t)
	_, err := dispatcher.Dispatch(context.Background(), conn, pushCommand)
	require.NoError(t, err, "Expected no error when handling RPUSH command")

	command := protocol.Command{
		Name: "LLEN",
		Args: []string{"list"},
	}

	_, err = dispatcher.Dispatch(context.Background(), conn, command) // should return the length of the list
	require.NoError(t, err, "Expected no error when handling LLEN command")
	require.Len(t, conn.writes, 2, "Expected two writes to the connection")
	assert.Equal(t,
		string([]byte(":3\r\n")),
		string(conn.writes[1]),
		"Expected LLEN command to return the length of the list",
	)
}

func TestMasterDispatcherHandleLPop(t *testing.T) {
	// First, we need to set the value for the key
	pushCommand := protocol.Command{
		Name: "RPUSH",
		Args: []string{"list", "value1", "value2", "value3"},
	}
	dispatcher, conn := setupMasterDispatcher(t)
	dispatcher.Dispatch(context.Background(), conn, pushCommand)

	command := protocol.Command{
		Name: "LPOP",
		Args: []string{"list"},
	}

	_, err := dispatcher.Dispatch(context.Background(), conn, command)

	require.NoError(t, err, "Expected no error when handling LPOP command")
	require.Len(t, conn.writes, 2, "Expected two writes to the connection")
	assert.Equal(t,
		string([]byte("$6\r\nvalue1\r\n")),
		string(conn.writes[1]),
		"Expected LPOP command to return the first element of the list",
	)
}

func TestMasterDispatcherHandleLPopMultipleElements(t *testing.T) {
	// First, we need to set the value for the key
	pushCommand := protocol.Command{
		Name: "RPUSH",
		Args: []string{"list", "value1", "value2", "value3"},
	}
	dispatcher, conn := setupMasterDispatcher(t)
	dispatcher.Dispatch(context.Background(), conn, pushCommand)

	command := protocol.Command{
		Name: "LPOP",
		Args: []string{"list", "2"},
	}

	_, err := dispatcher.Dispatch(context.Background(), conn, command)

	require.NoError(t, err, "Expected no error when handling LPOP command")
	require.Len(t, conn.writes, 2, "Expected two writes to the connection")
	assert.Equal(t,
		string([]byte("*2\r\n$6\r\nvalue1\r\n$6\r\nvalue2\r\n")),
		string(conn.writes[1]),
		"Expected LPOP command to return removed elements of the list",
	)
}

func TestMasterDispatcherHandleBLPop(t *testing.T) {
	key := "mylist"
	value := "hello"

	// Prepare the BLPOP command with a timeout
	blpopCommand := protocol.Command{
		Name: "BLPOP",
		Args: []string{key, "1"}, // 1 second timeout
	}

	dispatcher, conn := setupMasterDispatcher(t)

	// Run BLPOP in a separate goroutine
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, err := dispatcher.Dispatch(context.Background(), conn, blpopCommand)
		require.NoError(t, err, "Expected no error when handling BLPOP command")
	}()

	// Wait to ensure BLPOP is blocking
	time.Sleep(100 * time.Millisecond)

	// Now push a value to unblock the BLPOP
	rpushCommand := protocol.Command{
		Name: "RPUSH",
		Args: []string{key, value},
	}
	_, err := dispatcher.Dispatch(context.Background(), conn, rpushCommand)
	require.NoError(t, err, "Expected no error when handling RPUSH command")

	// Wait for BLPOP goroutine to complete
	<-done

	// Validate the response from BLPOP
	require.Len(t, conn.writes, 2, "Expected two writes to the connection")
	assert.Equal(t,
		string([]byte("*2\r\n$6\r\nmylist\r\n$5\r\nhello\r\n")),
		string(conn.writes[1]),
		"Expected BLPOP to return the key and value pushed",
	)
}

func TestMasterDispatcherHandleIncr(t *testing.T) {
	command := protocol.Command{
		Name: "INCR",
		Args: []string{"counter"},
	}

	dispatcher, conn := setupMasterDispatcher(t)

	// First increment should create the key with value 1
	_, err := dispatcher.Dispatch(context.Background(), conn, command)
	require.NoError(t, err, "Expected no error when handling INCR command")
	require.Len(t, conn.writes, 1, "Expected one write to the connection")
	assert.Equal(t,
		string([]byte(":1\r\n")),
		string(conn.writes[0]),
		"Expected INCR command to return 1 for the first increment",
	)

	// Second increment should return 2
	_, err = dispatcher.Dispatch(context.Background(), conn, command)
	require.NoError(t, err, "Expected no error when handling INCR command")
	require.Len(t, conn.writes, 2, "Expected two writes to the connection")
	assert.Equal(t,
		string([]byte(":2\r\n")),
		string(conn.writes[1]),
		"Expected INCR command to return 2 for the second increment",
	)
}

func TestMultiExecEmptyQueue(t *testing.T) {
	command := protocol.Command{
		Name: "MULTI",
	}

	dispatcher, conn := setupMasterDispatcher(t)

	// Start a transaction
	_, err := dispatcher.Dispatch(context.Background(), conn, command)
	require.NoError(t, err, "Expected no error when handling MULTI command")
	require.Len(t, conn.writes, 1, "Expected one write to the connection")
	assert.Equal(t,
		string([]byte("+OK\r\n")),
		string(conn.writes[0]),
		"Expected MULTI command to return OK",
	)

	// Execute the transaction
	execCommand := protocol.Command{
		Name: "EXEC",
	}
	_, err = dispatcher.Dispatch(context.Background(), conn, execCommand)
	require.NoError(t, err, "Expected no error when handling EXEC command")
	require.Len(t, conn.writes, 2, "Expected four writes to the connection")

	assert.Equal(t,
		string([]byte("*0\r\n")),
		string(conn.writes[1]),
		"Expected EXEC command to return results of the transaction",
	)
}

func TestMultiDiscard(t *testing.T) {
	dispatcher, conn := setupMasterDispatcher(t)
	command := protocol.Command{
		Name: "MULTI",
	}

	// Start a transaction
	_, err := dispatcher.Dispatch(context.Background(), conn, command)
	require.NoError(t, err, "Expected no error when handling MULTI command")
	require.Len(t, conn.writes, 1, "Expected one write to the connection")
	assert.Equal(t,
		string([]byte("+OK\r\n")),
		string(conn.writes[0]),
		"Expected MULTI command to return OK",
	)

	// Queue a SET command
	setCommand := protocol.Command{
		Name: "SET",
		Args: []string{"key", "value"},
	}
	_, err = dispatcher.Dispatch(context.Background(), conn, setCommand)
	require.NoError(t, err, "Expected no error when queuing SET command")
	assert.Equal(t,
		string([]byte("+QUEUED\r\n")),
		string(conn.writes[1]),
		"Expected SET command to return QUEUED",
	)

	// DISCARD
	discardCommand := protocol.Command{
		Name: "DISCARD",
	}
	_, err = dispatcher.Dispatch(context.Background(), conn, discardCommand)
	require.NoError(t, err, "Expected no error when handling EXEC command")
	require.Len(t, conn.writes, 3, "Expected four writes to the connection")

	assert.Equal(t,
		string([]byte("+OK\r\n")),
		string(conn.writes[2]),
		"Expected EXEC command to return results of the transaction",
	)

	// Verify that the transaction was discarded
	getCommand := protocol.Command{
		Name: "GET",
		Args: []string{"key"},
	}
	_, err = dispatcher.Dispatch(context.Background(), conn, getCommand)
	require.NoError(t, err, "Expected no error when handling GET command after DISCARD")
	require.Len(t, conn.writes, 4, "Expected four writes to the connection")
	assert.Equal(t,
		string([]byte("$-1\r\n")),
		string(conn.writes[3]),
		"Expected GET command to return nil after DISCARD",
	)
}

func TestMultiExecWithQueuedCommands(t *testing.T) {
	dispatcher, conn := setupMasterDispatcher(t)

	command := protocol.Command{
		Name: "MULTI",
	}
	// Start a transaction
	_, err := dispatcher.Dispatch(context.Background(), conn, command)
	require.NoError(t, err, "Expected no error when handling MULTI command")
	require.Len(t, conn.writes, 1, "Expected one write to the connection")
	assert.Equal(t,
		string([]byte("+OK\r\n")),
		string(conn.writes[0]),
		"Expected MULTI command to return OK",
	)

	// Queue a SET command
	setCommand := protocol.Command{
		Name: "SET",
		Args: []string{"key", "value"},
	}
	_, err = dispatcher.Dispatch(context.Background(), conn, setCommand)
	require.NoError(t, err, "Expected no error when queuing SET command")
	assert.Equal(t,
		string([]byte("+QUEUED\r\n")),
		string(conn.writes[1]),
		"Expected SET command to return QUEUED",
	)

	// Execute the transaction
	execCommand := protocol.Command{
		Name: "EXEC",
	}
	_, err = dispatcher.Dispatch(context.Background(), conn, execCommand)
	require.NoError(t, err, "Expected no error when handling EXEC command")
	require.Len(t, conn.writes, 3, "Expected two writes to the connection")

	assert.Equal(t,
		string([]byte("*1\r\n+OK\r\n")),
		string(conn.writes[2]),
		"Expected EXEC command to return results of the transaction",
	)
}

func TestMasterDispatcherHandleSubscribe(t *testing.T) {
	command := protocol.Command{
		Name: "SUBSCRIBE",
		Args: []string{"channel1"},
	}

	dispatcher, conn := setupMasterDispatcher(t)

	_, err := dispatcher.Dispatch(context.Background(), conn, command)

	require.NoError(t, err, "Expected no error when handling SUBSCRIBE command")
	require.Len(t, conn.writes, 1, "Expected one write to the connection")
	assert.Equal(t,
		string([]byte("*3\r\n$9\r\nsubscribe\r\n$8\r\nchannel1\r\n:1\r\n")),
		string(conn.writes[0]),
		"Expected SUBSCRIBE command to return subscription confirmation",
	)

	_, err = dispatcher.Dispatch(context.Background(), conn, command) // should not add a new channel
	require.Len(t, conn.writes, 2, "Expected one write to the connection")
	assert.Equal(t,
		string([]byte("*3\r\n$9\r\nsubscribe\r\n$8\r\nchannel1\r\n:1\r\n")),
		string(conn.writes[1]),
		"Expected SUBSCRIBE command to return subscription confirmation",
	)

	command2 := protocol.Command{
		Name: "SUBSCRIBE",
		Args: []string{"channel2"},
	}

	_, err = dispatcher.Dispatch(context.Background(), conn, command2) // should add a new channel
	require.Len(t, conn.writes, 3, "Expected one write to the connection")
	assert.Equal(t,
		string([]byte("*3\r\n$9\r\nsubscribe\r\n$8\r\nchannel2\r\n:2\r\n")),
		string(conn.writes[2]),
		"Expected SUBSCRIBE command to return subscription confirmation",
	)
}

func TestMasterDispatcherHandleUnsubscribe(t *testing.T) {
	command := protocol.Command{
		Name: "UNSUBSCRIBE",
		Args: []string{"channel1"},
	}

	dispatcher, conn := setupMasterDispatcher(t)

	// First subscribe to a channel
	subscribeCommand := protocol.Command{
		Name: "SUBSCRIBE",
		Args: []string{"channel1"},
	}
	_, err := dispatcher.Dispatch(context.Background(), conn, subscribeCommand)
	require.NoError(t, err, "Expected no error when handling SUBSCRIBE command")

	// Now unsubscribe from the channel
	_, err = dispatcher.Dispatch(context.Background(), conn, command)

	require.NoError(t, err, "Expected no error when handling UNSUBSCRIBE command")
	require.Len(t, conn.writes, 2, "Expected two writes to the connection")
	assert.Equal(t,
		string([]byte("*3\r\n$11\r\nunsubscribe\r\n$8\r\nchannel1\r\n:0\r\n")),
		string(conn.writes[1]),
		"Expected UNSUBSCRIBE command to return unsubscription confirmation",
	)
}

func TestMasterDispatcherHandleType(t *testing.T) {
	typeCommand := protocol.Command{
		Name: "TYPE",
		Args: []string{"key"},
	}

	dispatcher, conn := setupMasterDispatcher(t)

	setCommand := protocol.Command{
		Name: "SET",
		Args: []string{"key", "value"},
	}

	_, err := dispatcher.Dispatch(context.Background(), conn, setCommand)
	require.NoError(t, err, "Expected no error when setting key type")

	// Now check the type
	_, err = dispatcher.Dispatch(context.Background(), conn, typeCommand)
	require.NoError(t, err, "Expected no error when handling TYPE command")
	require.Len(t, conn.writes, 2, "Expected one write to the connection")
	assert.Equal(t, "+string\r\n", string(conn.writes[1]), "Expected TYPE command to return the correct type")
}

func TestMasterDispatcherHandleZAdd(t *testing.T) {
	command := protocol.Command{
		Name: "ZADD",
		Args: []string{"myzset", "1", "member1", "2", "member2"},
	}

	dispatcher, conn := setupMasterDispatcher(t)

	_, err := dispatcher.Dispatch(context.Background(), conn, command)

	require.NoError(t, err, "Expected no error when handling ZADD command")
	require.Len(t, conn.writes, 1, "Expected one write to the connection")
	assert.Equal(t,
		string([]byte(":2\r\n")),
		string(conn.writes[0]),
		"Expected ZADD command to return the number of elements added",
	)
}

func TestMasterDispatcherHandleZRank(t *testing.T) {
	// First, we need to add members to the sorted set
	zaddCommand := protocol.Command{
		Name: "ZADD",
		Args: []string{"myzset", "1", "member1", "2", "member2"},
	}
	dispatcher, conn := setupMasterDispatcher(t)
	_, err := dispatcher.Dispatch(context.Background(), conn, zaddCommand)
	require.NoError(t, err, "Expected no error when handling ZADD command")

	// Now test ZRANK
	zrankCommand := protocol.Command{
		Name: "ZRANK",
		Args: []string{"myzset", "member2"},
	}

	_, err = dispatcher.Dispatch(context.Background(), conn, zrankCommand)

	require.NoError(t, err, "Expected no error when handling ZRANK command")
	require.Len(t, conn.writes, 2, "Expected two writes to the connection")
	assert.Equal(t,
		string([]byte(":1\r\n")),
		string(conn.writes[1]),
		"Expected ZRANK command to return the correct rank",
	)

	// Test ZRANK for a non-existing member
	zrankNonExistCommand := protocol.Command{
		Name: "ZRANK",
		Args: []string{"myzset", "member3"},
	}

	_, err = dispatcher.Dispatch(context.Background(), conn, zrankNonExistCommand)
	require.NoError(t, err, "Expected no error when handling ZRANK command for non-existing member")
	require.Len(t, conn.writes, 3, "Expected three writes to the connection")
	assert.Equal(t,
		string([]byte("$-1\r\n")),
		string(conn.writes[2]),
		"Expected ZRANK command to return nil for non-existing member",
	)
}

func TestMasterDispatcherHandleZRankMissingKey(t *testing.T) {
	zrankCommand := protocol.Command{
		Name: "ZRANK",
		Args: []string{"nonexistset", "member1"},
	}
	dispatcher, conn := setupMasterDispatcher(t)

	_, err := dispatcher.Dispatch(context.Background(), conn, zrankCommand)

	require.NoError(t, err, "Expected no error when handling ZRANK command for missing key")
	require.Len(t, conn.writes, 1, "Expected one write to the connection")
	assert.Equal(t,
		string([]byte("$-1\r\n")),
		string(conn.writes[0]),
		"Expected ZRANK command to return nil for missing key",
	)
}

func TestMasterDispatcherHandleZRange(t *testing.T) {
	zaddCommand := protocol.Command{
		Name: "ZADD",
		Args: []string{"myzset", "1.2", "member1", "2", "member2", "1", "member3"},
	}
	dispatcher, conn := setupMasterDispatcher(t)
	_, err := dispatcher.Dispatch(context.Background(), conn, zaddCommand)
	require.NoError(t, err, "Expected no error when handling ZADD command")

	zrangeCommand := protocol.Command{
		Name: "ZRANGE",
		Args: []string{"myzset", "0", "1"},
	}
	_, err = dispatcher.Dispatch(context.Background(), conn, zrangeCommand)
	require.NoError(t, err, "Expected no error when handling ZRANGE command")
	require.Len(t, conn.writes, 2, "Expected two writes to the connection")
	assert.Equal(t,
		string([]byte("*2\r\n$7\r\nmember3\r\n$7\r\nmember1\r\n")),
		string(conn.writes[1]),
		"Expected ZRANGE command to return the correct members",
	)
}

func TestMasterDispatcherHandleZRangeNegativeIndex(t *testing.T) {
	zaddCommand := protocol.Command{
		Name: "ZADD",
		Args: []string{"myzset", "1.2", "member1", "2", "member2", "1", "member3"},
	}
	dispatcher, conn := setupMasterDispatcher(t)
	_, err := dispatcher.Dispatch(context.Background(), conn, zaddCommand)
	require.NoError(t, err, "Expected no error when handling ZADD command")

	zrangeCommand := protocol.Command{
		Name: "ZRANGE",
		Args: []string{"myzset", "0", "-2"},
	}
	_, err = dispatcher.Dispatch(context.Background(), conn, zrangeCommand)
	require.NoError(t, err, "Expected no error when handling ZRANGE command")
	require.Len(t, conn.writes, 2, "Expected two writes to the connection")
	assert.Equal(t,
		string([]byte("*2\r\n$7\r\nmember3\r\n$7\r\nmember1\r\n")),
		string(conn.writes[1]),
		"Expected ZRANGE command to return the correct members",
	)
}

func TestMasterDispatcherHandleZCard(t *testing.T) {
	zaddCommand := protocol.Command{
		Name: "ZADD",
		Args: []string{"myzset", "1.2", "member1", "2", "member2", "1", "member3"},
	}
	dispatcher, conn := setupMasterDispatcher(t)
	_, err := dispatcher.Dispatch(context.Background(), conn, zaddCommand)
	require.NoError(t, err, "Expected no error when handling ZADD command")

	zcardCommand := protocol.Command{
		Name: "ZCARD",
		Args: []string{"myzset"},
	}
	_, err = dispatcher.Dispatch(context.Background(), conn, zcardCommand)
	require.NoError(t, err, "Expected no error when handling ZCARD command")
	require.Len(t, conn.writes, 2, "Expected two writes to the connection")
	assert.Equal(t,
		string([]byte(":3\r\n")),
		string(conn.writes[1]),
		"Expected ZCARD command to return the correct cardinality",
	)
}

func TestMasterDispatcherHandleZScore(t *testing.T) {
	zaddCommand := protocol.Command{
		Name: "ZADD",
		Args: []string{"myzset", "1.2", "member1", "2", "member2", "1", "member3"},
	}
	dispatcher, conn := setupMasterDispatcher(t)
	_, err := dispatcher.Dispatch(context.Background(), conn, zaddCommand)
	require.NoError(t, err, "Expected no error when handling ZADD command")

	zscoreCommand := protocol.Command{
		Name: "ZSCORE",
		Args: []string{"myzset", "member1"},
	}
	_, err = dispatcher.Dispatch(context.Background(), conn, zscoreCommand)
	require.NoError(t, err, "Expected no error when handling ZSCORE command")
	require.Len(t, conn.writes, 2, "Expected two writes to the connection")
	assert.Equal(t,
		string([]byte("$3\r\n1.2\r\n")),
		string(conn.writes[1]),
		"Expected ZSCORE command to return the correct score",
	)
}

func TestMasterDispatcherHandlerZRem(t *testing.T) {
	zaddCommand := protocol.Command{
		Name: "ZADD",
		Args: []string{"myzset", "1.2", "member1", "2", "member2", "1", "member3"},
	}
	dispatcher, conn := setupMasterDispatcher(t)
	_, err := dispatcher.Dispatch(context.Background(), conn, zaddCommand)
	require.NoError(t, err, "Expected no error when handling ZADD command")

	zremCommand := protocol.Command{
		Name: "ZREM",
		Args: []string{"myzset", "member1", "member3"},
	}
	_, err = dispatcher.Dispatch(context.Background(), conn, zremCommand)
	require.NoError(t, err, "Expected no error when handling ZREM command")
	require.Len(t, conn.writes, 2, "Expected two writes to the connection")
	assert.Equal(t,
		string([]byte(":2\r\n")),
		string(conn.writes[1]),
		"Expected ZREM command to return the number of members removed",
	)
}

func TestMasterDispatcherHandlerXAdd(t *testing.T) {
	xaddCommand := protocol.Command{
		Name: "XADD",
		Args: []string{"mystream", "1526985054079-0", "field1", "value1", "field2", "value2"},
	}
	dispatcher, conn := setupMasterDispatcher(t)
	_, err := dispatcher.Dispatch(context.Background(), conn, xaddCommand)
	require.NoError(t, err, "Expected no error when handling XADD command")
	require.Len(t, conn.writes, 1, "Expected one write to the connection")
	assert.Regexp(t,
		"1526985054079-0",
		string(conn.writes[0]),
		"Expected XADD command to return a valid message ID",
	)
}

func TestMasterDispatcherHandlerXAddAutoSequence(t *testing.T) {
	xaddCommand := protocol.Command{
		Name: "XADD",
		Args: []string{"mystream", "1526985054079-0", "field1", "value1"},
	}
	xaddCommandAutoSequence := protocol.Command{
		Name: "XADD",
		Args: []string{"mystream", "1526985054079-*", "field1", "value1"},
	}
	dispatcher, conn := setupMasterDispatcher(t)
	_, err := dispatcher.Dispatch(context.Background(), conn, xaddCommand)
	require.NoError(t, err, "Expected no error when handling XADD command")

	_, err = dispatcher.Dispatch(context.Background(), conn, xaddCommandAutoSequence)
	require.NoError(t, err, "Expected no error when handling XADD command with auto-generated sequence")

	require.Len(t, conn.writes, 2, "Expected one write to the connection")
	assert.Regexp(t,
		"1526985054079-1",
		string(conn.writes[1]),
		"Expected XADD command to return a valid message ID",
	)
}

var _ net.Conn = (*MockConn)(nil)

type MockConn struct {
	writes [][]byte
}

func (m *MockConn) Write(b []byte) (int, error) {
	m.writes = append(m.writes, b)
	return len(b), nil
}
func (m *MockConn) Read(b []byte) (int, error) {
	return 0, nil
}
func (m *MockConn) Close() error {
	return nil
}
func (m *MockConn) LocalAddr() net.Addr {
	return nil
}
func (m *MockConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Port: 6379,
	}

}
func (m *MockConn) SetDeadline(t time.Time) error {
	return nil
}
func (m *MockConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *MockConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func setupMasterDispatcher(t *testing.T) (*DefaultCommandDispatcher, *MockConn) {
	t.Helper()

	config := &config.Config{
		Dir: "/tmp/dir",
	}
	replicasInfo, err := replication.NewReplicasInfo()
	replicasInfo.ReplID = "testreplid"
	require.NoError(t, err, "Failed to create replicas info")

	storage := storage.NewStorage()
	replicator := NewReplicator(replicasInfo)
	dispatcher := NewMasterCommandDispatcher(replicasInfo, replicator, storage, config)

	conn := &MockConn{}
	return dispatcher, conn
}

func setupReplicatedCommandDispatcher(
	t *testing.T, info *replication.MasterInfo,
) (*DefaultCommandDispatcher, *MockConn) {
	t.Helper()

	if info == nil {
		// Use default config if none provided
		info = &replication.MasterInfo{
			Conn:   &MockConn{},
			ReplID: "testreplid",
		}

	}
	storage := storage.NewStorage()
	dispatcher := NewReplicaReplicatedCommandDispatcher(info, storage)
	conn := &MockConn{}
	return dispatcher, conn
}
