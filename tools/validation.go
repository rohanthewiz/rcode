package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rohanthewiz/serr"
)

// ToolValidator provides validation for tool inputs
type ToolValidator struct {
	rules map[string]ValidationRules
}

// ValidationRules defines validation rules for a tool
type ValidationRules struct {
	RequiredParams []string
	ParamRules     map[string]ParamRule
	CustomRules    []CustomValidation
}

// ParamRule defines validation rules for a parameter
type ParamRule struct {
	Type         string   // "string", "integer", "boolean", "path", "regex"
	MinLength    int      // For strings
	MaxLength    int      // For strings
	MinValue     int      // For integers
	MaxValue     int      // For integers
	Pattern      string   // Regex pattern for validation
	AllowedValues []string // Enum values
	PathType     string   // "file", "directory", "any"
	MustExist    bool     // For paths
}

// CustomValidation is a function that performs custom validation
type CustomValidation func(params map[string]interface{}) error

// NewToolValidator creates a new tool validator with default rules
func NewToolValidator() *ToolValidator {
	v := &ToolValidator{
		rules: make(map[string]ValidationRules),
	}
	
	// Initialize default validation rules
	v.initializeDefaultRules()
	
	return v
}

// initializeDefaultRules sets up default validation rules for tools
func (v *ToolValidator) initializeDefaultRules() {
	// read_file validation
	v.rules["read_file"] = ValidationRules{
		RequiredParams: []string{"path"},
		ParamRules: map[string]ParamRule{
			"path": {
				Type:      "path",
				PathType:  "file",
				MustExist: true,
			},
		},
	}
	
	// write_file validation
	v.rules["write_file"] = ValidationRules{
		RequiredParams: []string{"path", "content"},
		ParamRules: map[string]ParamRule{
			"path": {
				Type:     "path",
				PathType: "file",
			},
			"content": {
				Type: "string",
			},
		},
	}
	
	// edit_file validation
	v.rules["edit_file"] = ValidationRules{
		RequiredParams: []string{"path", "start_line", "new_content"},
		ParamRules: map[string]ParamRule{
			"path": {
				Type:      "path",
				PathType:  "file",
				MustExist: true,
			},
			"start_line": {
				Type:     "integer",
				MinValue: 1,
			},
			"end_line": {
				Type:     "integer",
				MinValue: 1,
			},
			"new_content": {
				Type: "string",
			},
			"operation": {
				Type:          "string",
				AllowedValues: []string{"replace", "insert_before", "insert_after"},
			},
		},
		CustomRules: []CustomValidation{
			func(params map[string]interface{}) error {
				// Validate end_line >= start_line if both provided
				if startLine, ok := GetInt(params, "start_line"); ok {
					if endLine, ok := GetInt(params, "end_line"); ok {
						if endLine < startLine {
							return serr.New("end_line must be >= start_line")
						}
					}
				}
				return nil
			},
		},
	}
	
	// search validation
	v.rules["search"] = ValidationRules{
		RequiredParams: []string{"pattern"},
		ParamRules: map[string]ParamRule{
			"path": {
				Type:     "path",
				PathType: "any",
			},
			"pattern": {
				Type:      "regex",
				MinLength: 1,
			},
			"file_pattern": {
				Type: "string",
			},
			"case_sensitive": {
				Type: "boolean",
			},
			"max_results": {
				Type:     "integer",
				MinValue: 1,
				MaxValue: 1000,
			},
			"context_lines": {
				Type:     "integer",
				MinValue: 0,
				MaxValue: 10,
			},
		},
	}
	
	// bash validation
	v.rules["bash"] = ValidationRules{
		RequiredParams: []string{"command"},
		ParamRules: map[string]ParamRule{
			"command": {
				Type:      "string",
				MinLength: 1,
				MaxLength: 10000,
			},
			"timeout": {
				Type:     "integer",
				MinValue: 1000,
				MaxValue: 600000,
			},
		},
		CustomRules: []CustomValidation{
			func(params map[string]interface{}) error {
				// Validate dangerous commands
				if cmd, ok := GetString(params, "command"); ok {
					if isDangerousCommand(cmd) {
						return serr.New("command contains potentially dangerous operations")
					}
				}
				return nil
			},
		},
	}
	
	// Directory operations
	v.rules["list_dir"] = ValidationRules{
		ParamRules: map[string]ParamRule{
			"path": {
				Type:     "path",
				PathType: "directory",
			},
			"all": {
				Type: "boolean",
			},
			"recursive": {
				Type: "boolean",
			},
			"pattern": {
				Type: "string",
			},
		},
	}
	
	v.rules["make_dir"] = ValidationRules{
		RequiredParams: []string{"path"},
		ParamRules: map[string]ParamRule{
			"path": {
				Type:     "path",
				PathType: "directory",
			},
			"parents": {
				Type: "boolean",
			},
		},
	}
	
	v.rules["remove"] = ValidationRules{
		RequiredParams: []string{"path"},
		ParamRules: map[string]ParamRule{
			"path": {
				Type:      "path",
				PathType:  "any",
				MustExist: true,
			},
			"recursive": {
				Type: "boolean",
			},
			"force": {
				Type: "boolean",
			},
		},
		CustomRules: []CustomValidation{
			func(params map[string]interface{}) error {
				// Prevent removing critical paths
				if path, ok := GetString(params, "path"); ok {
					if isCriticalPath(path) {
						return serr.New("cannot remove critical system path")
					}
				}
				return nil
			},
		},
	}
	
	// Git operations
	v.rules["git_status"] = ValidationRules{
		ParamRules: map[string]ParamRule{
			"path": {
				Type:     "path",
				PathType: "directory",
			},
			"short": {
				Type: "boolean",
			},
		},
	}
	
	v.rules["git_diff"] = ValidationRules{
		ParamRules: map[string]ParamRule{
			"path": {
				Type:     "path",
				PathType: "directory",
			},
			"staged": {
				Type: "boolean",
			},
			"file": {
				Type: "string",
			},
			"stat": {
				Type: "boolean",
			},
			"name_only": {
				Type: "boolean",
			},
		},
	}
	
	// web_search validation
	v.rules["web_search"] = ValidationRules{
		RequiredParams: []string{"query"},
		ParamRules: map[string]ParamRule{
			"query": {
				Type:      "string",
				MinLength: 1,
				MaxLength: 500,
			},
			"max_results": {
				Type:     "integer",
				MinValue: 1,
				MaxValue: 50,
			},
		},
	}
}

// Validate validates tool parameters
func (v *ToolValidator) Validate(toolName string, params map[string]interface{}) error {
	rules, exists := v.rules[toolName]
	if !exists {
		// No validation rules defined for this tool
		return nil
	}
	
	// Check required parameters
	for _, required := range rules.RequiredParams {
		if _, exists := params[required]; !exists {
			return serr.New(fmt.Sprintf("required parameter '%s' is missing", required))
		}
	}
	
	// Validate each parameter
	for paramName, value := range params {
		if rule, exists := rules.ParamRules[paramName]; exists {
			if err := v.validateParam(paramName, value, rule); err != nil {
				return err
			}
		}
	}
	
	// Run custom validation rules
	for _, customRule := range rules.CustomRules {
		if err := customRule(params); err != nil {
			return err
		}
	}
	
	return nil
}

// validateParam validates a single parameter against its rule
func (v *ToolValidator) validateParam(name string, value interface{}, rule ParamRule) error {
	switch rule.Type {
	case "string":
		str, ok := value.(string)
		if !ok {
			return serr.New(fmt.Sprintf("parameter '%s' must be a string", name))
		}
		
		if rule.MinLength > 0 && len(str) < rule.MinLength {
			return serr.New(fmt.Sprintf("parameter '%s' must be at least %d characters", name, rule.MinLength))
		}
		if rule.MaxLength > 0 && len(str) > rule.MaxLength {
			return serr.New(fmt.Sprintf("parameter '%s' must be at most %d characters", name, rule.MaxLength))
		}
		if rule.Pattern != "" {
			if matched, _ := regexp.MatchString(rule.Pattern, str); !matched {
				return serr.New(fmt.Sprintf("parameter '%s' does not match required pattern", name))
			}
		}
		if len(rule.AllowedValues) > 0 {
			found := false
			for _, allowed := range rule.AllowedValues {
				if str == allowed {
					found = true
					break
				}
			}
			if !found {
				return serr.New(fmt.Sprintf("parameter '%s' must be one of: %v", name, rule.AllowedValues))
			}
		}
		
	case "integer":
		intVal, ok := GetInt(map[string]interface{}{name: value}, name)
		if !ok {
			return serr.New(fmt.Sprintf("parameter '%s' must be an integer", name))
		}
		
		if rule.MinValue > 0 && intVal < rule.MinValue {
			return serr.New(fmt.Sprintf("parameter '%s' must be at least %d", name, rule.MinValue))
		}
		if rule.MaxValue > 0 && intVal > rule.MaxValue {
			return serr.New(fmt.Sprintf("parameter '%s' must be at most %d", name, rule.MaxValue))
		}
		
	case "boolean":
		if _, ok := value.(bool); !ok {
			return serr.New(fmt.Sprintf("parameter '%s' must be a boolean", name))
		}
		
	case "path":
		path, ok := value.(string)
		if !ok {
			return serr.New(fmt.Sprintf("parameter '%s' must be a string path", name))
		}
		
		if rule.MustExist {
			if _, err := os.Stat(path); os.IsNotExist(err) {
				return serr.New(fmt.Sprintf("path '%s' does not exist", path))
			}
		}
		
		if rule.PathType != "" && rule.PathType != "any" {
			info, err := os.Stat(path)
			if err == nil {
				if rule.PathType == "file" && info.IsDir() {
					return serr.New(fmt.Sprintf("path '%s' must be a file, not a directory", path))
				}
				if rule.PathType == "directory" && !info.IsDir() {
					return serr.New(fmt.Sprintf("path '%s' must be a directory, not a file", path))
				}
			}
		}
		
	case "regex":
		pattern, ok := value.(string)
		if !ok {
			return serr.New(fmt.Sprintf("parameter '%s' must be a string", name))
		}
		
		if _, err := regexp.Compile(pattern); err != nil {
			return serr.New(fmt.Sprintf("parameter '%s' contains invalid regex: %v", name, err))
		}
	}
	
	return nil
}

// GetSchema returns the JSON schema for a tool
func (v *ToolValidator) GetSchema(toolName string) map[string]interface{} {
	rules, exists := v.rules[toolName]
	if !exists {
		return nil
	}
	
	properties := make(map[string]interface{})
	
	for paramName, rule := range rules.ParamRules {
		prop := make(map[string]interface{})
		
		// Map internal types to JSON schema types
		switch rule.Type {
		case "string", "path", "regex":
			prop["type"] = "string"
		case "integer":
			prop["type"] = "integer"
		case "boolean":
			prop["type"] = "boolean"
		}
		
		// Add constraints
		if rule.MinLength > 0 {
			prop["minLength"] = rule.MinLength
		}
		if rule.MaxLength > 0 {
			prop["maxLength"] = rule.MaxLength
		}
		if rule.MinValue > 0 {
			prop["minimum"] = rule.MinValue
		}
		if rule.MaxValue > 0 {
			prop["maximum"] = rule.MaxValue
		}
		if len(rule.AllowedValues) > 0 {
			prop["enum"] = rule.AllowedValues
		}
		if rule.Pattern != "" {
			prop["pattern"] = rule.Pattern
		}
		
		properties[paramName] = prop
	}
	
	return map[string]interface{}{
		"type":       "object",
		"properties": properties,
		"required":   rules.RequiredParams,
	}
}

// Helper functions

func isDangerousCommand(cmd string) bool {
	dangerous := []string{
		"rm -rf /",
		"dd if=/dev/zero",
		"mkfs",
		"shutdown",
		"reboot",
		":(){:|:&};:",  // Fork bomb
		"> /dev/sda",
	}
	
	cmdLower := strings.ToLower(cmd)
	for _, danger := range dangerous {
		if strings.Contains(cmdLower, danger) {
			return true
		}
	}
	
	return false
}

func isCriticalPath(path string) bool {
	abspath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	
	critical := []string{
		"/", "/etc", "/usr", "/bin", "/sbin",
		"/var", "/boot", "/dev", "/proc", "/sys",
		"/System", "/Library", "/Applications",
		"C:\\Windows", "C:\\Program Files",
	}
	
	for _, crit := range critical {
		if abspath == crit || abspath == crit+string(filepath.Separator) {
			return true
		}
	}
	
	return false
}

// AddCustomRule adds a custom validation rule for a tool
func (v *ToolValidator) AddCustomRule(toolName string, rule CustomValidation) {
	if rules, exists := v.rules[toolName]; exists {
		rules.CustomRules = append(rules.CustomRules, rule)
		v.rules[toolName] = rules
	}
}

// SetParamRule sets or updates a parameter rule for a tool
func (v *ToolValidator) SetParamRule(toolName, paramName string, rule ParamRule) {
	if rules, exists := v.rules[toolName]; exists {
		if rules.ParamRules == nil {
			rules.ParamRules = make(map[string]ParamRule)
		}
		rules.ParamRules[paramName] = rule
		v.rules[toolName] = rules
	}
}