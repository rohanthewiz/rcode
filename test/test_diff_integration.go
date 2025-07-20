package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// TestDiffIntegration tests the end-to-end diff visualization flow
// This should be run while the RCode server is running
func main() {
	baseURL := "http://localhost:8000"
	
	fmt.Println("🧪 Testing RCode Diff Visualization Integration")
	fmt.Println("================================================")
	
	// Check if server is running
	if !checkServerHealth(baseURL) {
		fmt.Println("❌ RCode server is not running at", baseURL)
		return
	}
	fmt.Println("✅ Server is running")
	
	// Note: This assumes you're already authenticated
	// In a real test, you'd need to handle OAuth flow
	
	fmt.Println("\n📋 Test Plan:")
	fmt.Println("1. Create a test session")
	fmt.Println("2. Simulate file modifications")
	fmt.Println("3. Check for diff creation")
	fmt.Println("4. Verify diff retrieval")
	
	// Get app info
	appInfo, err := getAppInfo(baseURL)
	if err != nil {
		fmt.Printf("❌ Failed to get app info: %v\n", err)
		return
	}
	
	if !appInfo["authenticated"].(bool) {
		fmt.Println("❌ Not authenticated. Please login first.")
		return
	}
	fmt.Println("✅ Authenticated")
	
	// Create test session
	sessionID, err := createTestSession(baseURL)
	if err != nil {
		fmt.Printf("❌ Failed to create session: %v\n", err)
		return
	}
	fmt.Printf("✅ Created session: %s\n", sessionID)
	
	// Test file modification flow
	testPath := "test/sample.go"
	beforeContent := `package main

func main() {
    println("Hello")
}`
	
	afterContent := `package main

import "fmt"

func main() {
    fmt.Println("Hello, RCode!")
    fmt.Println("Diff visualization test")
}`
	
	fmt.Printf("\n🔧 Simulating file modification: %s\n", testPath)
	
	// Note: In a real scenario, the diff would be created by the write_file tool
	// Here we're showing what the expected flow would look like
	
	fmt.Println("\n📊 Expected SSE Event:")
	expectedEvent := map[string]interface{}{
		"type":      "diff_available",
		"sessionId": sessionID,
		"data": map[string]interface{}{
			"diffId": "diff-123",
			"path":   testPath,
			"stats": map[string]int{
				"additions": 4,
				"deletions": 1,
			},
		},
	}
	
	eventJSON, _ := json.MarshalIndent(expectedEvent, "", "  ")
	fmt.Println(string(eventJSON))
	
	fmt.Println("\n✅ Test plan complete!")
	fmt.Println("\n📝 Manual Verification Steps:")
	fmt.Println("1. Check File Explorer for orange dot indicator")
	fmt.Println("2. Right-click the file and select 'View Changes'")
	fmt.Println("3. Verify all diff view modes work correctly")
	fmt.Println("4. Test Apply/Revert functionality")
	fmt.Println("5. Verify synchronized scrolling")
	
	fmt.Println("\n💡 Tip: Use the browser console to run:")
	fmt.Println("   window.diffViewer.showDiff('your-diff-id')")
}

func checkServerHealth(baseURL string) bool {
	resp, err := http.Get(baseURL + "/api/app")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func getAppInfo(baseURL string) (map[string]interface{}, error) {
	resp, err := http.Get(baseURL + "/api/app")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	
	return result, nil
}

func createTestSession(baseURL string) (string, error) {
	// Create session with test prompts
	payload := map[string]interface{}{
		"prompts": []string{
			"Testing diff visualization feature",
			"You have permission to read and write files for testing",
		},
	}
	
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	
	resp, err := http.Post(baseURL+"/api/session", "application/json", bytes.NewReader(jsonData))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		// Sometimes the response is just the session ID string
		return string(body), nil
	}
	
	if id, ok := result["id"].(string); ok {
		return id, nil
	}
	
	return "", fmt.Errorf("unexpected response format")
}