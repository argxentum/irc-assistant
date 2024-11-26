package elapse

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var timeOffsetRegexp = regexp.MustCompile(`^(\d+(?:\.\d+)?)([a-zA-Z]+)$`)

func ParseDuration(offset string) (time.Duration, error) {
	matches := timeOffsetRegexp.FindStringSubmatch(offset)
	if len(matches) != 3 {
		return time.Duration(0), fmt.Errorf("invalid duration, %s", offset)
	}

	quantity, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return time.Duration(0), fmt.Errorf("invalid quantity, %s", matches[1])
	}

	unit := strings.ToLower(matches[2])

	switch unit {
	case "s", "sec", "secs", "second", "seconds":
		break
	case "m", "min", "mins", "minute", "minutes":
		quantity *= 60
	case "h", "hr", "hrs", "hour", "hours":
		quantity *= 60 * 60
	case "d", "day", "days":
		quantity *= 60 * 60 * 24
	case "w", "week", "weeks":
		quantity *= 60 * 60 * 24 * 7
	case "mo", "mos", "month", "months":
		quantity *= 60 * 60 * 24 * 30
	case "y", "yr", "yrs", "year", "years":
		quantity *= 60 * 60 * 24 * 365
	default:
		return time.Duration(0), fmt.Errorf("invalid unit, %s", unit)
	}

	return time.Second * time.Duration(math.Round(quantity)), nil
}

func ParseDurationDescription(offset string) string {
	matches := timeOffsetRegexp.FindStringSubmatch(offset)
	if len(matches) != 3 {
		return "invalid duration"
	}

	quantity, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return "invalid quantity"
	}

	unit := strings.ToLower(matches[2])

	switch unit {
	case "s", "sec", "secs", "second", "seconds":
		unit = "second"
	case "m", "min", "mins", "minute", "minutes":
		unit = "minute"
	case "h", "hr", "hrs", "hour", "hours":
		unit = "hour"
	case "d", "day", "days":
		unit = "day"
	case "w", "week", "weeks":
		unit = "week"
	case "mo", "mos", "month", "months":
		unit = "month"
	case "y", "yr", "yrs", "year", "years":
		unit = "year"
	default:
		return "invalid unit"
	}

	if quantity == 1 {
		return fmt.Sprintf("%g %s", quantity, unit)
	}

	return fmt.Sprintf("%g %ss", quantity, unit)
}
