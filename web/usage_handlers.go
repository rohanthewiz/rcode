package web

import (
	"fmt"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/rweb"
	"github.com/rohanthewiz/serr"
	"rcode/db"
)

// GetSessionUsageHandler returns usage statistics for a session
func GetSessionUsageHandler(c rweb.Context) error {
	sessionID := c.Request().Param("id")
	if sessionID == "" {
		return c.WriteError(serr.New("session ID required"), 400)
	}

	database, err := db.GetDB()
	if err != nil {
		return c.WriteError(err, 500)
	}

	// Get session usage from database
	inputTokens, outputTokens, rateLimits, err := database.GetSessionUsage(sessionID)
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to get session usage"), 500)
	}

	// Calculate estimated cost (using Opus pricing as example)
	// Opus: $15 per million input tokens, $75 per million output tokens
	inputCost := float64(inputTokens) * 0.000015
	outputCost := float64(outputTokens) * 0.000075
	totalCost := inputCost + outputCost

	response := map[string]interface{}{
		"sessionId": sessionID,
		"usage": map[string]interface{}{
			"inputTokens":  inputTokens,
			"outputTokens": outputTokens,
			"totalTokens":  inputTokens + outputTokens,
		},
		"cost": map[string]interface{}{
			"input":  inputCost,
			"output": outputCost,
			"total":  totalCost,
		},
		"rateLimits": rateLimits,
	}

	return c.WriteJSON(response)
}

// GetDailyUsageHandler returns daily usage statistics
func GetDailyUsageHandler(c rweb.Context) error {
	database, err := db.GetDB()
	if err != nil {
		return c.WriteError(err, 500)
	}

	// Get daily usage from database
	usageByModel, err := database.GetDailyUsage()
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to get daily usage"), 500)
	}

	// Calculate total and costs
	totalInput := 0
	totalOutput := 0
	totalCost := 0.0

	modelStats := make([]map[string]interface{}, 0)
	for model, usage := range usageByModel {
		totalInput += usage.Input
		totalOutput += usage.Output

		// Calculate cost based on model
		var inputRate, outputRate float64
		switch {
		case contains(model, "opus"):
			inputRate = 0.000015
			outputRate = 0.000075
		case contains(model, "sonnet"):
			inputRate = 0.000003
			outputRate = 0.000015
		case contains(model, "haiku"):
			inputRate = 0.00000025
			outputRate = 0.00000125
		default:
			// Default to Sonnet pricing
			inputRate = 0.000003
			outputRate = 0.000015
		}

		modelCost := float64(usage.Input)*inputRate + float64(usage.Output)*outputRate
		totalCost += modelCost

		modelStats = append(modelStats, map[string]interface{}{
			"model":        model,
			"inputTokens":  usage.Input,
			"outputTokens": usage.Output,
			"totalTokens":  usage.Input + usage.Output,
			"cost":         modelCost,
		})
	}

	response := map[string]interface{}{
		"daily": map[string]interface{}{
			"totalInputTokens":  totalInput,
			"totalOutputTokens": totalOutput,
			"totalTokens":       totalInput + totalOutput,
			"totalCost":         totalCost,
			"byModel":           modelStats,
		},
	}

	return c.WriteJSON(response)
}

// GetGlobalUsageHandler returns global usage statistics
func GetGlobalUsageHandler(c rweb.Context) error {
	database, err := db.GetDB()
	if err != nil {
		return c.WriteError(err, 500)
	}

	// Get global usage from database
	usageByModel, rateLimits, err := database.GetGlobalUsage()
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to get global usage"), 500)
	}

	// Calculate totals and costs
	totalInput := 0
	totalOutput := 0
	totalCost := 0.0

	modelStats := make([]map[string]interface{}, 0)
	for model, usage := range usageByModel {
		totalInput += usage.Input
		totalOutput += usage.Output

		// Calculate cost based on model
		var inputRate, outputRate float64
		switch {
		case contains(model, "opus"):
			inputRate = 0.000015
			outputRate = 0.000075
		case contains(model, "sonnet"):
			inputRate = 0.000003
			outputRate = 0.000015
		case contains(model, "haiku"):
			inputRate = 0.00000025
			outputRate = 0.00000125
		default:
			// Default to Sonnet pricing
			inputRate = 0.000003
			outputRate = 0.000015
		}

		modelCost := float64(usage.Input)*inputRate + float64(usage.Output)*outputRate
		totalCost += modelCost

		modelStats = append(modelStats, map[string]interface{}{
			"model":        model,
			"inputTokens":  usage.Input,
			"outputTokens": usage.Output,
			"totalTokens":  usage.Input + usage.Output,
			"cost":         modelCost,
		})
	}

	// Add warnings if approaching limits
	var warnings []string
	if rateLimits != nil {
		// Check if approaching request limit
		if rateLimits.RequestsLimit > 0 && rateLimits.RequestsRemaining > 0 {
			percentRemaining := float64(rateLimits.RequestsRemaining) / float64(rateLimits.RequestsLimit) * 100
			if percentRemaining < 20 {
				warnings = append(warnings, fmt.Sprintf("Low on requests: %.0f%% remaining", percentRemaining))
			}
		}

		// Check if approaching token limits
		if rateLimits.InputTokensLimit > 0 && rateLimits.InputTokensRemaining > 0 {
			percentRemaining := float64(rateLimits.InputTokensRemaining) / float64(rateLimits.InputTokensLimit) * 100
			if percentRemaining < 20 {
				warnings = append(warnings, fmt.Sprintf("Low on input tokens: %.0f%% remaining", percentRemaining))
			}
		}

		if rateLimits.OutputTokensLimit > 0 && rateLimits.OutputTokensRemaining > 0 {
			percentRemaining := float64(rateLimits.OutputTokensRemaining) / float64(rateLimits.OutputTokensLimit) * 100
			if percentRemaining < 20 {
				warnings = append(warnings, fmt.Sprintf("Low on output tokens: %.0f%% remaining", percentRemaining))
			}
		}
	}

	response := map[string]interface{}{
		"global": map[string]interface{}{
			"totalInputTokens":  totalInput,
			"totalOutputTokens": totalOutput,
			"totalTokens":       totalInput + totalOutput,
			"totalCost":         totalCost,
			"byModel":           modelStats,
		},
		"rateLimits": rateLimits,
		"warnings":   warnings,
	}

	logger.Info("Global usage retrieved", "totalTokens", totalInput+totalOutput, "warnings", len(warnings))

	return c.WriteJSON(response)
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr ||
		len(s) >= len(substr) && s[len(s)-len(substr):] == substr ||
		len(s) > len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
