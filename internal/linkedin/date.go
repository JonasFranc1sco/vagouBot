package linkedin

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	reDaysAgo   = regexp.MustCompile(`(?i)(\d+)\s*\+?\s*days?\s*ago`)
	reWeeksAgo  = regexp.MustCompile(`(?i)(\d+)\s*\+?\s*weeks?\s*ago`)
	reMonthsAgo = regexp.MustCompile(`(?i)(\d+)\s*\+?\s*months?\s*ago`)
	reYearsAgo  = regexp.MustCompile(`(?i)(\d+)\s*\+?\s*years?\s*ago`)
)

// parseAbsoluteDate converte o atributo datetime (YYYY-MM-DD) do card do
// LinkedIn para time.Time. Retorna zero time se não for conversível.
func parseAbsoluteDate(value string) time.Time {
	value = strings.TrimSpace(value)
	for _, layout := range []string{"2006-01-02", "2006-01-02T15:04:05Z", time.RFC3339} {
		if t, err := time.Parse(layout, value); err == nil {
			return t
		}
	}
	return time.Time{}
}

// parsePostedDate converte o texto relativo de postagem ("3 days ago",
// "1 week ago", "hoje") em uma data absoluta usando `now` como referência.
// Retorna zero time se não conseguir interpretar.
func parsePostedDate(text string, now time.Time) (time.Time, bool) {
	lower := strings.ToLower(strings.TrimSpace(text))
	if lower == "" {
		return time.Time{}, false
	}

	switch {
	case strings.Contains(lower, "hoje") || strings.Contains(lower, "today") || lower == "agora":
		return now, true
	case strings.Contains(lower, "ontem") || strings.Contains(lower, "yesterday"):
		return now.AddDate(0, 0, -1), true
	}

	if m := reDaysAgo.FindStringSubmatch(lower); m != nil {
		return now.AddDate(0, 0, -parseInt(m[1])), true
	}
	if m := reWeeksAgo.FindStringSubmatch(lower); m != nil {
		return now.AddDate(0, 0, -parseInt(m[1])*7), true
	}
	if m := reMonthsAgo.FindStringSubmatch(lower); m != nil {
		return now.AddDate(0, -parseInt(m[1]), 0), true
	}
	if m := reYearsAgo.FindStringSubmatch(lower); m != nil {
		return now.AddDate(-parseInt(m[1]), 0, 0), true
	}

	if t := parseAbsoluteDate(lower); !t.IsZero() {
		return t, true
	}

	return time.Time{}, false
}

func parseInt(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}
