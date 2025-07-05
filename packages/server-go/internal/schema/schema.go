package schema

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/rohanthewiz/serr"
)

// Type represents the type of a schema
type Type string

const (
	TypeString  Type = "string"
	TypeNumber  Type = "number"
	TypeBoolean Type = "boolean"
	TypeObject  Type = "object"
	TypeArray   Type = "array"
)

// Object creates an object schema with the given properties
func Object(properties map[string]Schema, required ...string) Schema {
	return &ObjectSchema{
		Properties: properties,
		Required:   required,
	}
}

// String creates a string schema
func String() *StringSchema {
	return &StringSchema{}
}

// Number creates a number schema
func Number() *NumberSchema {
	return &NumberSchema{}
}

// Boolean creates a boolean schema
func Boolean() *BooleanSchema {
	return &BooleanSchema{}
}

// Array creates an array schema
func Array(items Schema) *ArraySchema {
	return &ArraySchema{
		Items: items,
	}
}

// ObjectSchema validates object/map values
type ObjectSchema struct {
	Properties  map[string]Schema
	Required    []string
	description string
}

func (s *ObjectSchema) Validate(value any) error {
	// Check if value is a map
	mapValue, ok := value.(map[string]any)
	if !ok {
		return serr.New("expected object, got %T", value)
	}
	
	// Check required fields
	for _, field := range s.Required {
		if _, exists := mapValue[field]; !exists {
			return serr.New("missing required field: %s", field)
		}
	}
	
	// Validate each property
	for key, val := range mapValue {
		if schema, exists := s.Properties[key]; exists {
			if err := schema.Validate(val); err != nil {
				return serr.Wrap(err, "field %s validation failed", key)
			}
		}
	}
	
	return nil
}

func (s *ObjectSchema) Description() string {
	return s.description
}

func (s *ObjectSchema) Describe(desc string) *ObjectSchema {
	s.description = desc
	return s
}

func (s *ObjectSchema) ToJSON() map[string]any {
	props := make(map[string]any)
	for key, schema := range s.Properties {
		props[key] = schema.ToJSON()
	}
	
	result := map[string]any{
		"type":       "object",
		"properties": props,
	}
	
	if len(s.Required) > 0 {
		result["required"] = s.Required
	}
	
	if s.description != "" {
		result["description"] = s.description
	}
	
	return result
}

// StringSchema validates string values
type StringSchema struct {
	MinLength   *int
	MaxLength   *int
	Pattern     *regexp.Regexp
	description string
}

func (s *StringSchema) Validate(value any) error {
	str, ok := value.(string)
	if !ok {
		return serr.New("expected string, got %T", value)
	}
	
	// Check length constraints
	if s.MinLength != nil && len(str) < *s.MinLength {
		return serr.New("string length %d is less than minimum %d", len(str), *s.MinLength)
	}
	
	if s.MaxLength != nil && len(str) > *s.MaxLength {
		return serr.New("string length %d exceeds maximum %d", len(str), *s.MaxLength)
	}
	
	// Check pattern
	if s.Pattern != nil && !s.Pattern.MatchString(str) {
		return serr.New("string does not match pattern %s", s.Pattern.String())
	}
	
	return nil
}

func (s *StringSchema) Description() string {
	return s.description
}

func (s *StringSchema) Describe(desc string) *StringSchema {
	s.description = desc
	return s
}

func (s *StringSchema) Min(length int) *StringSchema {
	s.MinLength = &length
	return s
}

func (s *StringSchema) Max(length int) *StringSchema {
	s.MaxLength = &length
	return s
}

func (s *StringSchema) Regex(pattern string) *StringSchema {
	s.Pattern = regexp.MustCompile(pattern)
	return s
}

func (s *StringSchema) ToJSON() map[string]any {
	result := map[string]any{
		"type": "string",
	}
	
	if s.MinLength != nil {
		result["minLength"] = *s.MinLength
	}
	
	if s.MaxLength != nil {
		result["maxLength"] = *s.MaxLength
	}
	
	if s.Pattern != nil {
		result["pattern"] = s.Pattern.String()
	}
	
	if s.description != "" {
		result["description"] = s.description
	}
	
	return result
}

// NumberSchema validates numeric values
type NumberSchema struct {
	Min         *float64
	Max         *float64
	description string
}

func (s *NumberSchema) Validate(value any) error {
	// Convert various numeric types to float64
	var num float64
	switch v := value.(type) {
	case float64:
		num = v
	case float32:
		num = float64(v)
	case int:
		num = float64(v)
	case int64:
		num = float64(v)
	case int32:
		num = float64(v)
	default:
		return serr.New("expected number, got %T", value)
	}
	
	// Check range constraints
	if s.Min != nil && num < *s.Min {
		return serr.New("number %v is less than minimum %v", num, *s.Min)
	}
	
	if s.Max != nil && num > *s.Max {
		return serr.New("number %v exceeds maximum %v", num, *s.Max)
	}
	
	return nil
}

func (s *NumberSchema) Description() string {
	return s.description
}

func (s *NumberSchema) Describe(desc string) *NumberSchema {
	s.description = desc
	return s
}

func (s *NumberSchema) Minimum(min float64) *NumberSchema {
	s.Min = &min
	return s
}

func (s *NumberSchema) Maximum(max float64) *NumberSchema {
	s.Max = &max
	return s
}

func (s *NumberSchema) ToJSON() map[string]any {
	result := map[string]any{
		"type": "number",
	}
	
	if s.Min != nil {
		result["minimum"] = *s.Min
	}
	
	if s.Max != nil {
		result["maximum"] = *s.Max
	}
	
	if s.description != "" {
		result["description"] = s.description
	}
	
	return result
}

// BooleanSchema validates boolean values
type BooleanSchema struct {
	description string
}

func (s *BooleanSchema) Validate(value any) error {
	_, ok := value.(bool)
	if !ok {
		return serr.New("expected boolean, got %T", value)
	}
	return nil
}

func (s *BooleanSchema) Description() string {
	return s.description
}

func (s *BooleanSchema) Describe(desc string) *BooleanSchema {
	s.description = desc
	return s
}

func (s *BooleanSchema) ToJSON() map[string]any {
	result := map[string]any{
		"type": "boolean",
	}
	
	if s.description != "" {
		result["description"] = s.description
	}
	
	return result
}

// ArraySchema validates array/slice values
type ArraySchema struct {
	Items       Schema
	MinItems    *int
	MaxItems    *int
	description string
}

func (s *ArraySchema) Validate(value any) error {
	// Use reflection to handle different slice types
	v := reflect.ValueOf(value)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return serr.New("expected array, got %T", value)
	}
	
	length := v.Len()
	
	// Check length constraints
	if s.MinItems != nil && length < *s.MinItems {
		return serr.New("array length %d is less than minimum %d", length, *s.MinItems)
	}
	
	if s.MaxItems != nil && length > *s.MaxItems {
		return serr.New("array length %d exceeds maximum %d", length, *s.MaxItems)
	}
	
	// Validate each item if schema is provided
	if s.Items != nil {
		for i := 0; i < length; i++ {
			if err := s.Items.Validate(v.Index(i).Interface()); err != nil {
				return serr.Wrap(err, "item at index %d validation failed", i)
			}
		}
	}
	
	return nil
}

func (s *ArraySchema) Description() string {
	return s.description
}

func (s *ArraySchema) Describe(desc string) *ArraySchema {
	s.description = desc
	return s
}

func (s *ArraySchema) Min(items int) *ArraySchema {
	s.MinItems = &items
	return s
}

func (s *ArraySchema) Max(items int) *ArraySchema {
	s.MaxItems = &items
	return s
}

func (s *ArraySchema) ToJSON() map[string]any {
	result := map[string]any{
		"type": "array",
	}
	
	if s.Items != nil {
		result["items"] = s.Items.ToJSON()
	}
	
	if s.MinItems != nil {
		result["minItems"] = *s.MinItems
	}
	
	if s.MaxItems != nil {
		result["maxItems"] = *s.MaxItems
	}
	
	if s.description != "" {
		result["description"] = s.description
	}
	
	return result
}

// Optional wraps a schema to make it optional
type OptionalSchema struct {
	Schema Schema
}

func Optional(schema Schema) Schema {
	return &OptionalSchema{Schema: schema}
}

func (s *OptionalSchema) Validate(value any) error {
	// nil is valid for optional schemas
	if value == nil {
		return nil
	}
	return s.Schema.Validate(value)
}

func (s *OptionalSchema) Description() string {
	return s.Schema.Description()
}

func (s *OptionalSchema) ToJSON() map[string]any {
	// In JSON Schema, optional is handled by not including the field in "required"
	return s.Schema.ToJSON()
}