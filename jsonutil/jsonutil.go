package jsonutil

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/kaptinlin/jsonrepair"
	"github.com/tailscale/hujson"
)

// SanitizeJSON removes markdown code blocks and attempts to repair malformed JSON
func SanitizeJSON(s string) string {
	// Remove ```json and ``` markers
	s = regexp.MustCompile(`(?s)^\s*`+"```"+`json?\s*`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`(?s)\s*`+"```"+`\s*$`).ReplaceAllString(s, "")
	s = strings.TrimSpace(s)

	// If already valid JSON, don't bother with repair/standardization
	if json.Valid([]byte(s)) {
		return s
	}

	// First try jsonrepair
	repaired, err := jsonrepair.Repair(s)
	if err == nil {
		s = repaired
		// If repaired version is valid, return it early
		if json.Valid([]byte(s)) {
			return s
		}
	}

	// Then use hujson to further standardize (fixes trailing commas, comments, etc.)
	standardized, err := hujson.Standardize([]byte(s))
	if err == nil {
		return string(standardized)
	}

	return s
}

// ExtractJSON extracts the first JSON object from mixed text
// Useful when LLM responses include extra commentary before/after JSON
func ExtractJSON(text string) string {
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start == -1 || end == -1 || start >= end {
		return "{}"
	}
	return text[start : end+1]
}
