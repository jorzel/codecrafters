package commands

import (
	"context"
	"testing"

	"github.com/codecrafters-io/redis-starter-go/app/protocol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func dispatchAll(t *testing.T, dispatcher *DefaultCommandDispatcher, conn *MockConn, commands ...protocol.Command) {
	t.Helper()
	for _, command := range commands {
		_, err := dispatcher.Dispatch(context.Background(), conn, command)
		require.NoError(t, err, "Expected no error when handling %s", command.Name)
	}
}

func xadd(args ...string) protocol.Command {
	return protocol.Command{Name: "XADD", Args: args}
}

func xrange(args ...string) protocol.Command {
	return protocol.Command{Name: "XRANGE", Args: args}
}

func TestMasterDispatcherHandleXRangeReturnsInclusiveRangeWithFieldsInInsertionOrder(t *testing.T) {
	dispatcher, conn := setupMasterDispatcher(t)
	dispatchAll(t, dispatcher, conn,
		xadd("s", "0-1", "a", "1"),
		xadd("s", "0-2", "temperature", "36", "humidity", "95"),
		xadd("s", "0-3", "c", "3"),
		xrange("s", "0-2", "0-3"),
	)

	require.Len(t, conn.writes, 4)
	assert.Equal(t,
		"*2\r\n"+
			"*2\r\n$3\r\n0-2\r\n*4\r\n$11\r\ntemperature\r\n$2\r\n36\r\n$8\r\nhumidity\r\n$2\r\n95\r\n"+
			"*2\r\n$3\r\n0-3\r\n*2\r\n$1\r\nc\r\n$1\r\n3\r\n",
		string(conn.writes[3]),
	)
}

func TestMasterDispatcherHandleXRangeWithoutSequenceCoversWholeMillisecond(t *testing.T) {
	dispatcher, conn := setupMasterDispatcher(t)
	dispatchAll(t, dispatcher, conn,
		xadd("s", "1-0", "a", "1"),
		xadd("s", "1-1", "b", "2"),
		xadd("s", "2-0", "c", "3"),
		xrange("s", "1", "1"),
	)

	require.Len(t, conn.writes, 4)
	assert.Equal(t,
		"*2\r\n"+
			"*2\r\n$3\r\n1-0\r\n*2\r\n$1\r\na\r\n$1\r\n1\r\n"+
			"*2\r\n$3\r\n1-1\r\n*2\r\n$1\r\nb\r\n$1\r\n2\r\n",
		string(conn.writes[3]),
	)
}

func TestMasterDispatcherHandleXRangeMissingKeyReturnsEmptyArray(t *testing.T) {
	dispatcher, conn := setupMasterDispatcher(t)
	dispatchAll(t, dispatcher, conn, xrange("missing", "0-1", "0-3"))

	require.Len(t, conn.writes, 1)
	assert.Equal(t, "*0\r\n", string(conn.writes[0]))
}
