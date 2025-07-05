package anthropic

import (
	"github.com/sst/opencode/server-go/internal/schema"
	"github.com/sst/opencode/server-go/internal/tool"
)

// ConvertSchemaToJSON converts internal tool schema to JSON Schema format for Anthropic
func ConvertSchemaToJSON(s tool.Schema) map[string]interface{} {
	return convertSchema(s)
}

func convertSchema(s tool.Schema) map[string]interface{} {
	switch typed := s.(type) {
	case *schema.ObjectSchema:
		return convertObjectSchema(typed)
	case *schema.StringSchema:
		return convertStringSchema(typed)
	case *schema.NumberSchema:
		return convertNumberSchema(typed)
	case *schema.BooleanSchema:
		return convertBooleanSchema(typed)
	case *schema.ArraySchema:
		return convertArraySchema(typed)
	case *schema.OptionalSchema:
		// For optional fields, return the inner schema
		// The "required" array in the parent object handles optionality
		return convertSchema(typed.Inner)
	default:
		// Fallback for unknown types
		return map[string]interface{}{
			"type": "string",
		}
	}
}

func convertObjectSchema(s *schema.ObjectSchema) map[string]interface{} {
	properties := make(map[string]interface{})
	required := []string{}

	for name, fieldSchema := range s.Properties {
		properties[name] = convertSchema(fieldSchema)
		
		// Check if field is required (not wrapped in OptionalSchema)
		if _, isOptional := fieldSchema.(*schema.OptionalSchema); !isOptional {
			required = append(required, name)
		}
	}

	result := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}

	if len(required) > 0 {
		result["required"] = required
	}

	if s.Description != "" {
		result["description"] = s.Description
	}

	return result
}

func convertStringSchema(s *schema.StringSchema) map[string]interface{} {
	result := map[string]interface{}{
		"type": "string",
	}

	if s.Description != "" {
		result["description"] = s.Description
	}

	if s.MinLength > 0 {
		result["minLength"] = s.MinLength
	}

	if s.MaxLength > 0 {
		result["maxLength"] = s.MaxLength
	}

	if s.Pattern != "" {
		result["pattern"] = s.Pattern
	}

	if len(s.Enum) > 0 {
		result["enum"] = s.Enum
	}

	return result
}

func convertNumberSchema(s *schema.NumberSchema) map[string]interface{} {
	result := map[string]interface{}{
		"type": "number",
	}

	if s.Description != "" {
		result["description"] = s.Description
	}

	if s.Minimum != nil {
		result["minimum"] = *s.Minimum
	}

	if s.Maximum != nil {
		result["maximum"] = *s.Maximum
	}

	return result
}

func convertBooleanSchema(s *schema.BooleanSchema) map[string]interface{} {
	result := map[string]interface{}{
		"type": "boolean",
	}

	if s.Description != "" {
		result["description"] = s.Description
	}

	return result
}

func convertArraySchema(s *schema.ArraySchema) map[string]interface{} {
	result := map[string]interface{}{
		"type":  "array",
		"items": convertSchema(s.Items),
	}

	if s.Description != "" {
		result["description"] = s.Description
	}

	if s.MinItems > 0 {
		result["minItems"] = s.MinItems
	}

	if s.MaxItems > 0 {
		result["maxItems"] = s.MaxItems
	}

	return result
}

// ConvertToolToAnthropicFormat converts a tool to Anthropic's expected format
func ConvertToolToAnthropicFormat(t tool.Tool) AnthropicTool {
	// Replace dots with underscores in tool ID as per OpenCode convention
	name := t.ID()
	// Note: In the TypeScript implementation, dots are replaced with underscores
	// but for now we'll keep the original ID
	
	return AnthropicTool{
		Name:        name,
		Description: t.Description(),
		InputSchema: ConvertSchemaToJSON(t.Parameters()),
	}
}