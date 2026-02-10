package discord

import (
	"fmt"
	"strconv"
	"strings"
)

func normalizeUint64String(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	v, err := strconv.ParseUint(s, 0, 64)
	if err != nil {
		return "", err
	}
	return strconv.FormatUint(v, 10), nil
}

func intToUint64String(v int) string {
	if v <= 0 {
		return "0"
	}
	return strconv.FormatUint(uint64(v), 10)
}

func uint64StringToPermissionBit(s string) (uint64, error) {
	ns, err := normalizeUint64String(s)
	if err != nil {
		return 0, err
	}
	if ns == "" {
		return 0, nil
	}
	v, err := strconv.ParseUint(ns, 10, 64)
	if err != nil {
		return 0, err
	}
	return v, nil
}

func maxPlatformInt() uint64 {
	// int max for current platform.
	return uint64(^uint(0) >> 1)
}

func uint64ToIntIfFits(v uint64) (int, error) {
	if v > maxPlatformInt() {
		return 0, fmt.Errorf("value %d does not fit in int on this platform", v)
	}
	return int(v), nil
}
