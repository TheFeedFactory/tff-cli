package cmd

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

var relativeTimeRe = regexp.MustCompile(`^(\d+)(d|w|mo|y)$`)

// ParseRelativeTime parses relative time expressions like "3d", "2w", "1mo", "1y"
// or absolute dates like "2026-01-15". For relative expressions, returns the time
// that many units ago from now.
func ParseRelativeTime(s string) (time.Time, error) {
	if m := relativeTimeRe.FindStringSubmatch(s); m != nil {
		n, _ := strconv.Atoi(m[1])
		now := time.Now()
		switch m[2] {
		case "d":
			return now.AddDate(0, 0, -n), nil
		case "w":
			return now.AddDate(0, 0, -n*7), nil
		case "mo":
			return now.AddDate(0, -n, 0), nil
		case "y":
			return now.AddDate(-n, 0, 0), nil
		}
	}

	t, err := time.Parse("2006-01-02", s)
	if err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("invalid time expression %q (use e.g. 3d, 2w, 1mo, 1y, or 2026-01-15)", s)
}

// ParseRelativeDate parses a relative or absolute date and returns it in yyyy-mm-dd format.
func ParseRelativeDate(s string) (string, error) {
	t, err := ParseRelativeTime(s)
	if err != nil {
		return "", err
	}
	return t.Format("2006-01-02"), nil
}

// ParseRelativeISO parses a relative or absolute date and returns it in ISO 8601 format.
func ParseRelativeISO(s string) (string, error) {
	t, err := ParseRelativeTime(s)
	if err != nil {
		return "", err
	}
	return t.Format(time.RFC3339), nil
}
