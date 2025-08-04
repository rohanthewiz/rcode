package tools

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestRipgrepToolFilesOnly tests the files_only output mode
func TestRipgrepToolFilesOnly(t *testing.T) {
	// Skip if ripgrep is not installed
	if _, err := exec.LookPath("rg"); err != nil {
		t.Skip("ripgrep not installed, skipping test")
	}

	// Create a temporary directory with test files
	tmpDir, err := os.MkdirTemp("", "ripgrep_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files with known content
	testFiles := map[string]string{
		"file1.go":   "func TestFunction() {}\nvar testVar = 42",
		"file2.go":   "func AnotherFunction() {}\nconst testConst = 100",
		"file3.txt":  "This is a test file\nWith test content",
		"README.md":  "# Test Project\nThis is a test readme",
	}

	for name, content := range testFiles {
		filePath := filepath.Join(tmpDir, name)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Test files_only mode
	tool := &RipgrepTool{}
	input := map[string]interface{}{
		"pattern":     "test",
		"path":        tmpDir,
		"output_mode": "files_only",
		"case_sensitive": false,
	}

	result, err := tool.Execute(input)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Check that we found the expected files
	if !strings.Contains(result, "file1.go") {
		t.Errorf("Expected file1.go in results, got: %s", result)
	}
	if !strings.Contains(result, "file2.go") {
		t.Errorf("Expected file2.go in results, got: %s", result)
	}
	if !strings.Contains(result, "file3.txt") {
		t.Errorf("Expected file3.txt in results, got: %s", result)
	}
	if !strings.Contains(result, "README.md") {
		t.Errorf("Expected README.md in results, got: %s", result)
	}
}

// TestRipgrepToolCount tests the count output mode
func TestRipgrepToolCount(t *testing.T) {
	// Skip if ripgrep is not installed
	if _, err := exec.LookPath("rg"); err != nil {
		t.Skip("ripgrep not installed, skipping test")
	}

	// Create a temporary directory with test files
	tmpDir, err := os.MkdirTemp("", "ripgrep_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file with multiple matches
	testContent := `func TestOne() {}
func TestTwo() {}
func TestThree() {}
var nontest = 42`

	filePath := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(filePath, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Test count mode
	tool := &RipgrepTool{}
	input := map[string]interface{}{
		"pattern":     "Test",
		"path":        tmpDir,
		"output_mode": "count",
	}

	result, err := tool.Execute(input)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Check that we got the correct count
	if !strings.Contains(result, "test.go: 3 matches") {
		t.Errorf("Expected 3 matches in test.go, got: %s", result)
	}
	if !strings.Contains(result, "Total: 3 matches") {
		t.Errorf("Expected total of 3 matches, got: %s", result)
	}
}

// TestRipgrepToolContent tests the content output mode with context
func TestRipgrepToolContent(t *testing.T) {
	// Skip if ripgrep is not installed
	if _, err := exec.LookPath("rg"); err != nil {
		t.Skip("ripgrep not installed, skipping test")
	}

	// Create a temporary directory with test files
	tmpDir, err := os.MkdirTemp("", "ripgrep_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file with context
	testContent := `line 1
line 2
target line here
line 4
line 5`

	filePath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(filePath, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Test content mode with context
	tool := &RipgrepTool{}
	input := map[string]interface{}{
		"pattern":       "target",
		"path":          tmpDir,
		"output_mode":   "content",
		"context_lines": 1,
	}

	result, err := tool.Execute(input)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Check that we got context lines
	if !strings.Contains(result, "line 2") {
		t.Errorf("Expected context line 'line 2', got: %s", result)
	}
	if !strings.Contains(result, "target line here") {
		t.Errorf("Expected match line 'target line here', got: %s", result)
	}
	if !strings.Contains(result, "line 4") {
		t.Errorf("Expected context line 'line 4', got: %s", result)
	}
}

// TestRipgrepToolFileType tests file type filtering
func TestRipgrepToolFileType(t *testing.T) {
	// Skip if ripgrep is not installed
	if _, err := exec.LookPath("rg"); err != nil {
		t.Skip("ripgrep not installed, skipping test")
	}

	// Create a temporary directory with mixed file types
	tmpDir, err := os.MkdirTemp("", "ripgrep_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files of different types
	testFiles := map[string]string{
		"test.go":   "func SearchMe() {}",
		"test.js":   "function searchMe() {}",
		"test.py":   "def search_me():",
		"test.txt":  "search me please",
	}

	for name, content := range testFiles {
		filePath := filepath.Join(tmpDir, name)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Test with Go file type filter
	tool := &RipgrepTool{}
	input := map[string]interface{}{
		"pattern":     "search",
		"path":        tmpDir,
		"output_mode": "files_only",
		"file_type":   "go",
		"case_sensitive": false,
	}

	result, err := tool.Execute(input)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Check that we only found Go files
	if !strings.Contains(result, "test.go") {
		t.Errorf("Expected test.go in results, got: %s", result)
	}
	if strings.Contains(result, "test.js") || strings.Contains(result, "test.py") || strings.Contains(result, "test.txt") {
		t.Errorf("Found non-Go files in results when filtering for Go files: %s", result)
	}
}

// TestRipgrepToolGlob tests glob pattern filtering
func TestRipgrepToolGlob(t *testing.T) {
	// Skip if ripgrep is not installed
	if _, err := exec.LookPath("rg"); err != nil {
		t.Skip("ripgrep not installed, skipping test")
	}

	// Create a temporary directory with nested structure
	tmpDir, err := os.MkdirTemp("", "ripgrep_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create nested directories and files
	testFiles := map[string]string{
		"main.go":           "func main() {}",
		"main_test.go":      "func TestMain() {}",
		"pkg/util.go":       "func Util() {}",
		"pkg/util_test.go":  "func TestUtil() {}",
	}

	for name, content := range testFiles {
		filePath := filepath.Join(tmpDir, name)
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Test with glob pattern for test files
	tool := &RipgrepTool{}
	input := map[string]interface{}{
		"pattern":     "Test",
		"path":        tmpDir,
		"output_mode": "files_only",
		"glob":        "*_test.go",
	}

	result, err := tool.Execute(input)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Check that we only found test files
	if !strings.Contains(result, "main_test.go") {
		t.Errorf("Expected main_test.go in results, got: %s", result)
	}
	if !strings.Contains(result, "util_test.go") {
		t.Errorf("Expected util_test.go in results, got: %s", result)
	}
	if strings.Contains(result, "main.go") || strings.Contains(result, "util.go") {
		t.Errorf("Found non-test files when filtering with *_test.go glob: %s", result)
	}
}

// TestRipgrepToolNoMatches tests behavior when no matches are found
func TestRipgrepToolNoMatches(t *testing.T) {
	// Skip if ripgrep is not installed
	if _, err := exec.LookPath("rg"); err != nil {
		t.Skip("ripgrep not installed, skipping test")
	}

	// Create a temporary directory with a test file
	tmpDir, err := os.MkdirTemp("", "ripgrep_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	filePath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(filePath, []byte("nothing to see here"), 0644); err != nil {
		t.Fatal(err)
	}

	// Test with pattern that won't match
	tool := &RipgrepTool{}
	input := map[string]interface{}{
		"pattern":     "impossible_pattern_xyz123",
		"path":        tmpDir,
		"output_mode": "files_only",
	}

	result, err := tool.Execute(input)
	if err != nil {
		t.Fatalf("Execute should not fail for no matches: %v", err)
	}

	// Check that we got a no matches message
	if !strings.Contains(result, "No files found") {
		t.Errorf("Expected 'No files found' message, got: %s", result)
	}
}

// TestRipgrepToolMultiline tests multiline pattern matching
func TestRipgrepToolMultiline(t *testing.T) {
	// Skip if ripgrep is not installed
	if _, err := exec.LookPath("rg"); err != nil {
		t.Skip("ripgrep not installed, skipping test")
	}

	// Create a temporary directory with test file
	tmpDir, err := os.MkdirTemp("", "ripgrep_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file with multiline content
	testContent := `func Example() {
	// This is a
	// multiline comment
	fmt.Println("test")
}`

	filePath := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(filePath, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Test multiline pattern
	tool := &RipgrepTool{}
	input := map[string]interface{}{
		"pattern":     `// This is a.*// multiline`,
		"path":        tmpDir,
		"output_mode": "files_only",
		"multiline":   true,
	}

	result, err := tool.Execute(input)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Check that we found the file with multiline match
	if !strings.Contains(result, "test.go") {
		t.Errorf("Expected test.go in multiline search results, got: %s", result)
	}
}