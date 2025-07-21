package main

import (
	"fmt"
	"log"
	"time"
	
	"rcode/web"
)

// Test program to verify tool execution events are broadcast correctly
func main() {
	// Simulate a session ID
	sessionID := "test-session-123"
	
	// Test tool execution start
	fmt.Println("Broadcasting tool execution start...")
	web.BroadcastToolExecutionStart(sessionID, "tool-1", "write_file")
	time.Sleep(500 * time.Millisecond)
	
	// Test tool execution progress
	fmt.Println("Broadcasting tool execution progress...")
	web.BroadcastToolExecutionProgress(sessionID, "tool-1", 25, "Writing file... 25%")
	time.Sleep(500 * time.Millisecond)
	
	web.BroadcastToolExecutionProgress(sessionID, "tool-1", 50, "Writing file... 50%")
	time.Sleep(500 * time.Millisecond)
	
	web.BroadcastToolExecutionProgress(sessionID, "tool-1", 75, "Writing file... 75%")
	time.Sleep(500 * time.Millisecond)
	
	// Test tool execution complete
	fmt.Println("Broadcasting tool execution complete...")
	metrics := map[string]interface{}{
		"bytesWritten": 1024,
		"linesWritten": 42,
	}
	web.BroadcastToolExecutionComplete(sessionID, "tool-1", "success", "✓ Wrote test.txt (1024 bytes)", 1500, metrics)
	
	fmt.Println("Test completed!")
}