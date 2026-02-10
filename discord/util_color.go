package discord

import (
	"fmt"
	"strconv"
	"strings"
)

// ConvertToInt converts a hex color string (#rrggbb or rrggbb) to a decimal int64.
func ConvertToInt(hex string) (int64, error) {
	s := strings.TrimSpace(hex)
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 {
		return 0, fmt.Errorf("expected 6 hex characters, got %q", hex)
	}
	v, err := strconv.ParseInt(s, 16, 64)
	if err != nil {
		return 0, err
	}
	return v, nil
}
