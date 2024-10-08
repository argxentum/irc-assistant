package elapsed

import (
	"fmt"
	"time"
)

func ElapsedTimeDescription(t time.Time) string {
	elapsed := time.Now().Sub(t)

	year := time.Hour * 24 * 365
	month := time.Hour * 24 * 30
	week := time.Hour * 24 * 7
	day := time.Hour * 24
	hour := time.Hour
	minute := time.Minute
	second := time.Second

	if elapsed >= year {
		years := elapsed / year
		if years == 1 {
			return "last year"
		}
		return fmt.Sprintf("%d years ago", years)
	} else if elapsed >= month {
		months := elapsed / month
		if months == 1 {
			return "last month"
		}
		return fmt.Sprintf("%d months ago", months)
	} else if elapsed >= week {
		weeks := elapsed / week
		if weeks == 1 {
			return "last week"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	} else if elapsed >= day {
		days := elapsed / day
		if days == 1 {
			return "yesterday"
		}
		return fmt.Sprintf("%d days ago", days)
	} else if elapsed >= hour {
		hours := elapsed / hour
		if hours == 1 {
			return "an hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else if elapsed >= minute {
		minutes := elapsed / minute
		if minutes == 1 {
			return "a minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	} else {
		seconds := elapsed / second
		if seconds < 5 {
			return "just now"
		}
		if seconds < 30 {
			return "a few seconds ago"
		}
		return fmt.Sprintf("%d seconds ago", seconds)
	}
}
