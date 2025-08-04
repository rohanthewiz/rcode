package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSmartEditReplaceMode tests pattern-based replacement mode
func TestSmartEditReplaceMode(t *testing.T) {
	// Create temporary test file
	tmpDir, err := os.MkdirTemp("", "smart_edit_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.go")
	originalContent := `func oldFunction() {
	fmt.Println("old function")
}

func anotherOldFunction() {
	oldVariable := 42
	fmt.Println(oldVariable)
}`

	if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
		t.Fatal(err)
	}

	tool := &SmartEditTool{}

	// Test 1: Replace all occurrences with minimal response
	input := map[string]interface{}{
		"path":          testFile,
		"mode":          "replace",
		"pattern":       "old",
		"replacement":   "new",
		"replace_all":   true,
		"response_mode": "minimal",
	}

	result, err := tool.Execute(input)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Check minimal response format
	if !strings.Contains(result, "4 replacements") {
		t.Errorf("Expected '4 replacements' in minimal response, got: %s", result)
	}

	// Verify file was modified
	content, _ := os.ReadFile(testFile)
	if strings.Contains(string(content), "old") {
		t.Error("File still contains 'old' after replacement")
	}
	if !strings.Contains(string(content), "newFunction") {
		t.Error("File doesn't contain 'newFunction' after replacement")
	}

	// Test 2: Single replacement with summary response
	os.WriteFile(testFile, []byte(originalContent), 0644) // Reset

	input2 := map[string]interface{}{
		"path":          testFile,
		"mode":          "replace",
		"pattern":       "oldFunction",
		"replacement":   "newFunction",
		"replace_all":   false,
		"response_mode": "summary",
	}

	result2, err := tool.Execute(input2)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(result2, "Replacements: 1") {
		t.Errorf("Expected 'Replacements: 1' in summary, got: %s", result2)
	}

	// Test 3: Regex pattern with named groups (Go uses ${1} not $1)
	os.WriteFile(testFile, []byte(originalContent), 0644) // Reset

	input3 := map[string]interface{}{
		"path":          testFile,
		"mode":          "replace",
		"pattern":       `(\w+)Function`,
		"replacement":   "${1}Func",
		"response_mode": "minimal",
	}

	_, err = tool.Execute(input3)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	content3, _ := os.ReadFile(testFile)
	if !strings.Contains(string(content3), "oldFunc") || !strings.Contains(string(content3), "anotherOldFunc") {
		t.Errorf("Regex replacement with backreference failed. Content: %s", string(content3))
	}
}

// TestSmartEditLineMode tests line-based editing mode
func TestSmartEditLineMode(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "smart_edit_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	originalContent := `line 1
line 2
line 3
line 4
line 5`

	if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
		t.Fatal(err)
	}

	tool := &SmartEditTool{}

	// Test line replacement
	input := map[string]interface{}{
		"path":          testFile,
		"mode":          "line",
		"start_line":    2,
		"end_line":      3,
		"new_content":   "new line 2\nnew line 3",
		"operation":     "replace",
		"response_mode": "minimal",
	}

	result, err := tool.Execute(input)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(result, "2 lines modified") {
		t.Errorf("Expected '2 lines modified', got: %s", result)
	}

	// Test insert before
	input2 := map[string]interface{}{
		"path":          testFile,
		"mode":          "line",
		"start_line":    1,
		"new_content":   "inserted line",
		"operation":     "insert_before",
		"response_mode": "summary",
	}

	result2, err := tool.Execute(input2)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(result2, "Lines added: 1") {
		t.Errorf("Expected 'Lines added: 1' in summary, got: %s", result2)
	}

	content, _ := os.ReadFile(testFile)
	lines := strings.Split(string(content), "\n")
	if lines[0] != "inserted line" {
		t.Error("Insert before operation failed")
	}
}

// TestSmartEditSedMode tests sed command mode
func TestSmartEditSedMode(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "smart_edit_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	originalContent := `var x = 10
var y = 20
const z = 30
var w = 40`

	if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
		t.Fatal(err)
	}

	tool := &SmartEditTool{}

	// Test sed replacement
	input := map[string]interface{}{
		"path": testFile,
		"mode": "sed",
		"commands": []interface{}{
			"s/var/let/g",
			"/const/d",
		},
		"response_mode": "summary",
	}

	_, err = tool.Execute(input)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	content, _ := os.ReadFile(testFile)
	contentStr := string(content)

	// Check replacements
	if strings.Contains(contentStr, "var") {
		t.Error("Sed replacement failed: 'var' still present")
	}
	if !strings.Contains(contentStr, "let") {
		t.Error("Sed replacement failed: 'let' not found")
	}
	// Check deletion
	if strings.Contains(contentStr, "const") {
		t.Error("Sed deletion failed: 'const' still present")
	}
}

// TestSmartEditDryRun tests dry run functionality
func TestSmartEditDryRun(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "smart_edit_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	originalContent := `original content`

	if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
		t.Fatal(err)
	}

	tool := &SmartEditTool{}

	// Test dry run - should not modify file
	input := map[string]interface{}{
		"path":          testFile,
		"mode":          "replace",
		"pattern":       "original",
		"replacement":   "modified",
		"dry_run":       true,
		"response_mode": "minimal",
	}

	result, err := tool.Execute(input)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Check dry run prefix
	if !strings.HasPrefix(result, "[DRY RUN]") {
		t.Errorf("Expected dry run prefix, got: %s", result)
	}

	// Verify file was NOT modified
	content, _ := os.ReadFile(testFile)
	if !strings.Contains(string(content), "original") {
		t.Error("File was modified during dry run")
	}
	if strings.Contains(string(content), "modified") {
		t.Error("File was modified during dry run")
	}
}

// TestSmartEditResponseModes tests different response modes
func TestSmartEditResponseModes(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "smart_edit_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	originalContent := `line 1
line 2
line 3`

	if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
		t.Fatal(err)
	}

	tool := &SmartEditTool{}

	testCases := []struct {
		name         string
		responseMode string
		maxLength    int // Expected max response length
	}{
		{"minimal", "minimal", 50},
		{"summary", "summary", 200},
		{"diff", "diff", 500},
		{"full", "full", 1000},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset file
			os.WriteFile(testFile, []byte(originalContent), 0644)

			input := map[string]interface{}{
				"path":          testFile,
				"mode":          "replace",
				"pattern":       "line",
				"replacement":   "row",
				"response_mode": tc.responseMode,
			}

			result, err := tool.Execute(input)
			if err != nil {
				t.Fatalf("Execute failed: %v", err)
			}

			// Check response is appropriately sized
			if tc.responseMode == "minimal" && len(result) > tc.maxLength {
				t.Errorf("Minimal response too long: %d chars (expected < %d)", 
					len(result), tc.maxLength)
			}

			// Check response contains expected elements
			switch tc.responseMode {
			case "minimal":
				if !strings.Contains(result, "replacement") {
					t.Error("Minimal response missing replacement count")
				}
			case "summary":
				if !strings.Contains(result, "Edited:") {
					t.Error("Summary response missing file path")
				}
			case "diff":
				if !strings.Contains(result, "@@") || !strings.Contains(result, "+") {
					t.Error("Diff response missing diff markers")
				}
			case "full":
				if !strings.Contains(result, "Before:") || !strings.Contains(result, "After:") {
					t.Error("Full response missing before/after sections")
				}
			}
		})
	}
}

// TestSmartEditBackup tests backup functionality
func TestSmartEditBackup(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "smart_edit_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	originalContent := `original content`

	if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
		t.Fatal(err)
	}

	tool := &SmartEditTool{}

	// Test with backup enabled
	input := map[string]interface{}{
		"path":        testFile,
		"mode":        "replace",
		"pattern":     "original",
		"replacement": "modified",
		"backup":      true,
	}

	_, err = tool.Execute(input)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Check backup file exists
	backupFile := testFile + ".bak"
	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		t.Error("Backup file was not created")
	}

	// Verify backup contains original content
	backupContent, _ := os.ReadFile(backupFile)
	if string(backupContent) != originalContent {
		t.Error("Backup file doesn't contain original content")
	}

	// Verify main file was modified
	newContent, _ := os.ReadFile(testFile)
	if !strings.Contains(string(newContent), "modified") {
		t.Error("Main file was not modified")
	}
}