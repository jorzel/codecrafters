package commands

import (
	"testing"

	"github.com/codecrafters-io/redis-starter-go/app/protocol"
	"github.com/codecrafters-io/redis-starter-go/app/storage"
)

func TestXRangeHandler(t *testing.T) {
	storage := storage.NewStorage()
	
	// Add handler
	xaddHandler := NewXAddHandler(storage)
	xrangeHandler := NewXRangeHandler(storage)

	// Add some test entries
	_, err := xaddHandler.apply(protocol.Command{
		Name: "XADD",
		Args: []string{"stream_key", "0-1", "foo", "bar"},
	})
	if err != nil {
		t.Fatalf("Failed to add first entry: %v", err)
	}

	_, err = xaddHandler.apply(protocol.Command{
		Name: "XADD", 
		Args: []string{"stream_key", "0-2", "bar", "baz"},
	})
	if err != nil {
		t.Fatalf("Failed to add second entry: %v", err)
	}

	_, err = xaddHandler.apply(protocol.Command{
		Name: "XADD",
		Args: []string{"stream_key", "0-3", "baz", "foo"},
	})
	if err != nil {
		t.Fatalf("Failed to add third entry: %v", err)
	}

	// Test XRANGE 0-2 0-3
	result, err := xrangeHandler.apply(protocol.Command{
		Name: "XRANGE",
		Args: []string{"stream_key", "0-2", "0-3"},
	})
	if err != nil {
		t.Fatalf("XRANGE failed: %v", err)
	}

	// The result should be a RESP array containing 2 entries
	if result == nil {
		t.Fatal("XRANGE returned nil result")
	}

	t.Logf("XRANGE result: %s", string(result))

	// Test XRANGE with all entries
	result, err = xrangeHandler.apply(protocol.Command{
		Name: "XRANGE",
		Args: []string{"stream_key", "0-1", "0-3"},
	})
	if err != nil {
		t.Fatalf("XRANGE with all entries failed: %v", err)
	}

	if result == nil {
		t.Fatal("XRANGE returned nil result for all entries")
	}

	t.Logf("XRANGE all entries result: %s", string(result))

	// Test XRANGE on non-existent stream
	result, err = xrangeHandler.apply(protocol.Command{
		Name: "XRANGE",
		Args: []string{"nonexistent_key", "0-1", "0-3"},
	})
	if err != nil {
		t.Fatalf("XRANGE on non-existent stream failed: %v", err)
	}

	// Should return empty array
	if result == nil {
		t.Fatal("XRANGE on non-existent stream returned nil")
	}

	t.Logf("XRANGE non-existent result: %s", string(result))
}