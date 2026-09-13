// Package nldate parses a small set of English natural-language date
// expressions into an absolute time, with a fallback to RFC 3339 for
// programmatic clients.
package nldate

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var weekdays = map[string]time.Weekday{
	"sunday":    time.Sunday,
	"monday":    time.Monday,
	"tuesday":   time.Tuesday,
	"wednesday": time.Wednesday,
	"thursday":  time.Thursday,
	"friday":    time.Friday,
	"saturday":  time.Saturday,
}

var inDaysPattern = regexp.MustCompile(`^in (\d+) days?$`)

// Parse resolves input into an absolute date, relative to ref. Recognized
// expressions are English-only and case-insensitive:
//
//   - "today", "tomorrow"
//   - a weekday name ("monday") - the next occurrence of that day, not
//     including today (1 to 7 days ahead)
//   - "next <weekday>" ("next monday") - the occurrence a full week after
//     the bare weekday name (8 to 14 days ahead)
//   - "in N days"
//
// Anything else is parsed as RFC 3339. Expressions without an explicit time
// of day resolve to midnight UTC of the resulting date.
func Parse(input string, ref time.Time) (time.Time, error) {
	text := strings.ToLower(strings.TrimSpace(input))
	dayStart := time.Date(ref.Year(), ref.Month(), ref.Day(), 0, 0, 0, 0, time.UTC)

	switch text {
	case "today":
		return dayStart, nil
	case "tomorrow":
		return dayStart.AddDate(0, 0, 1), nil
	}

	if m := inDaysPattern.FindStringSubmatch(text); m != nil {
		n, err := strconv.Atoi(m[1])
		if err != nil {
			return time.Time{}, fmt.Errorf("nldate: invalid day count in %q", input)
		}
		return dayStart.AddDate(0, 0, n), nil
	}

	if rest, ok := strings.CutPrefix(text, "next "); ok {
		if wd, ok := weekdays[rest]; ok {
			return dayStart.AddDate(0, 0, daysUntil(ref.Weekday(), wd)+7), nil
		}
	}

	if wd, ok := weekdays[text]; ok {
		return dayStart.AddDate(0, 0, daysUntil(ref.Weekday(), wd)), nil
	}

	if t, err := time.Parse(time.RFC3339, input); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("nldate: unrecognized date expression %q", input)
}

// daysUntil returns how many days ahead the next occurrence of target is,
// strictly after from: always in the range 1-7.
func daysUntil(from, target time.Weekday) int {
	diff := (int(target) - int(from) - 1 + 7) % 7
	return diff + 1
}
