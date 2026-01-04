package ansi

import (
	"crypto/sha256"
	"fmt"
)

// generateChromaThemeName generates a unique theme name based on the Chroma configuration.
// This ensures that different color schemes get different theme names, allowing proper
// theme switching without conflicts.
func generateChromaThemeName(config *Chroma) string {
	// Build a fingerprint from key colors that define the theme
	// We use a subset of colors that are most distinctive
	fingerprint := fmt.Sprintf("%v|%v|%v|%v|%v|%v|%v|%v",
		ptrStr(config.Keyword.Color),
		ptrStr(config.LiteralString.Color),
		ptrStr(config.Comment.Color),
		ptrStr(config.LiteralNumber.Color),
		ptrStr(config.NameFunction.Color),
		ptrStr(config.NameBuiltin.Color),
		ptrStr(config.Operator.Color),
		ptrStr(config.Background.BackgroundColor),
	)
	
	// Generate a short hash for the theme name
	hash := sha256.Sum256([]byte(fingerprint))
	// Use first 8 bytes (16 hex chars) for a reasonably short but unique name
	return fmt.Sprintf("charm-%x", hash[:8])
}

// ptrStr safely extracts string value from pointer, returns empty string if nil
func ptrStr(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}
