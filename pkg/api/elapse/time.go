package elapse

import (
	"fmt"
	"math"
	"time"
)

func TimeDescription(t time.Time) string {
	if t.Before(time.Now()) {
		return PastTimeDescription(t)
	} else {
		return FutureTimeDescription(t)
	}
}

func PastTimeDescription(t time.Time) string {
	elapsed := time.Now().Sub(t)
	years, months, weeks, days, hours, minutes, seconds := units(elapsed)

	if years > 0 {
		if years == 1 {
			return "last year"
		}
		return fmt.Sprintf("%d years ago", years)
	} else if months > 0 {
		if months == 1 {
			return "last month"
		}
		return fmt.Sprintf("%d months ago", months)
	} else if weeks > 0 {
		if weeks == 1 {
			return "last week"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	} else if days > 0 {
		if days == 1 {
			return "yesterday"
		}
		return fmt.Sprintf("%d days ago", days)
	} else if hours > 0 {
		if hours == 1 {
			return "an hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else if minutes > 0 {
		if minutes == 1 {
			return "a minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	} else {
		if seconds < 5 {
			return "just now"
		}
		if seconds < 30 {
			return "a few seconds ago"
		}
		return fmt.Sprintf("%d seconds ago", seconds)
	}
}

func PastTimeDescriptionConcise(t time.Time) string {
	elapsed := time.Now().Sub(t)
	years, months, weeks, days, hours, minutes, seconds := units(elapsed)

	if years > 0 {
		if years == 1 {
			return "a year"
		}
		return fmt.Sprintf("%d years", years)
	} else if months > 0 {
		if months == 1 {
			return "a month"
		}
		return fmt.Sprintf("%d months", months)
	} else if weeks > 0 {
		if weeks == 1 {
			return "a week"
		}
		return fmt.Sprintf("%d weeks", weeks)
	} else if days > 0 {
		if days == 1 {
			return "a day"
		}
		return fmt.Sprintf("%d days", days)
	} else if hours > 0 {
		if hours == 1 {
			return "an hour"
		}
		return fmt.Sprintf("%d hours", hours)
	} else if minutes > 0 {
		if minutes == 1 {
			return "a minute"
		}
		return fmt.Sprintf("%d minutes", minutes)
	} else {
		if seconds == 1 {
			return "a second"
		}
		return fmt.Sprintf("%d seconds", seconds)
	}
}

func FutureTimeDescription(t time.Time) string {
	elapsed := t.Sub(time.Now())
	years, months, weeks, days, hours, minutes, seconds := units(elapsed)

	if years > 0 {
		if years == 1 {
			return "next year"
		}
		return fmt.Sprintf("%d years from now", years)
	} else if months > 0 {
		if months == 1 {
			return "next month"
		}
		return fmt.Sprintf("%d months from now", months)
	} else if weeks > 0 {
		if weeks == 1 {
			return "next week"
		}
		return fmt.Sprintf("%d weeks from now", weeks)
	} else if days > 0 {
		if days == 1 {
			return "tomorrow"
		}
		return fmt.Sprintf("%d days from now", days)
	} else if hours > 0 {
		if hours == 1 {
			return "an hour from now"
		}
		return fmt.Sprintf("%d hours from now", hours)
	} else if minutes > 0 {
		if minutes == 1 {
			return "a minute from now"
		}
		return fmt.Sprintf("%d minutes from now", minutes)
	} else {
		if seconds == 1 {
			return "a second from now"
		}
		return fmt.Sprintf("%d seconds from now", seconds)
	}
}

func FutureTimeDescriptionConcise(t time.Time) string {
	elapsed := t.Sub(time.Now())
	years, months, weeks, days, hours, minutes, seconds := units(elapsed)

	if years > 0 {
		if years == 1 {
			return "a year"
		}
		return fmt.Sprintf("%d years", years)
	} else if months > 0 {
		if months == 1 {
			return "a month"
		}
		return fmt.Sprintf("%d months", months)
	} else if weeks > 0 {
		if weeks == 1 {
			return "a week"
		}
		return fmt.Sprintf("%d weeks", weeks)
	} else if days > 0 {
		if days == 1 {
			return "a day"
		}
		return fmt.Sprintf("%d days", days)
	} else if hours > 0 {
		if hours == 1 {
			return "an hour"
		}
		return fmt.Sprintf("%d hours", hours)
	} else if minutes > 0 {
		if minutes == 1 {
			return "a minute"
		}
		return fmt.Sprintf("%d minutes", minutes)
	} else {
		if seconds == 1 {
			return "a second"
		}
		return fmt.Sprintf("%d seconds", seconds)
	}
}

func units(d time.Duration) (years, months, weeks, days, hours, minutes, seconds int) {
	year := time.Hour * 24 * 365
	month := time.Hour * 24 * 30
	week := time.Hour * 24 * 7
	day := time.Hour * 24
	hour := time.Hour
	minute := time.Minute
	second := time.Second

	if d >= year {
		years = measure(d, year)
		d -= time.Duration(years) * year
	}
	if d >= month {
		months = measure(d, month)
		d -= time.Duration(months) * month
	}
	if d >= week {
		weeks = measure(d, week)
		d -= time.Duration(weeks) * week
	}
	if d >= day {
		days = measure(d, day)
		d -= time.Duration(days) * day
	}
	if d >= hour {
		hours = measure(d, hour)
		d -= time.Duration(hours) * hour
	}
	if d >= minute {
		minutes = measure(d, minute)
		d -= time.Duration(minutes) * minute
	}
	if d >= second {
		seconds = measure(d, second)
	}

	return
}

func measure(a, b time.Duration) int {
	return int(math.Round(float64(a) / float64(b)))
}
