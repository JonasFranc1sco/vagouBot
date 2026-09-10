package linkedin

import (
	"testing"
	"time"
)

func TestParseAbsoluteDate(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"data padrão", "2026-09-01", "2026-09-01"},
		{"timestamp ISO", "2026-09-01T12:00:00Z", "2026-09-01T12:00:00Z"},
		{"vazio", "", ""},
		{"texto", "3 days ago", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseAbsoluteDate(tt.input)
			if tt.want == "" {
				if !got.IsZero() {
					t.Fatalf("esperava zero, obtive %v", got)
				}
				return
			}
			want := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
			switch tt.input {
			case "2026-09-01T12:00:00Z":
				want = time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
			}
			if !got.Equal(want) {
				t.Fatalf("esperava %v, obtive %v", want, got)
			}
		})
	}
}

func TestParsePostedDate(t *testing.T) {
	now := time.Date(2026, 9, 9, 15, 0, 0, 0, time.UTC)

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"hoje", "Hoje", "2026-09-09"},
		{"today", "today", "2026-09-09"},
		{"ontem", "ontem", "2026-09-08"},
		{"yesterday", "yesterday", "2026-09-08"},
		{"1 day ago", "1 day ago", "2026-09-08"},
		{"3 days ago", "3 days ago", "2026-09-06"},
		{"30+ days ago", "30+ days ago", "2026-08-10"},
		{"1 week ago", "1 week ago", "2026-09-02"},
		{"2 weeks ago", "2 weeks ago", "2026-08-26"},
		{"1 month ago", "1 month ago", "2026-08-09"},
		{"1 year ago", "1 year ago", "2025-09-09"},
		{"data absoluta", "2026-09-01", "2026-09-01"},
		{"vazio", "   ", ""},
		{"desconhecido", "tempo integral", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parsePostedDate(tt.input, now)
			if tt.want == "" {
				if ok {
					t.Fatalf("esperava falha, obtive %v", got)
				}
				return
			}
			want, _ := time.Parse("2006-01-02", tt.want)
			if !ok || got.Format("2006-01-02") != want.Format("2006-01-02") {
				t.Fatalf("esperava %v (ok=%v), obtive %v (ok=%v)",
					want.Format("2006-01-02"), true, got.Format("2006-01-02"), ok)
			}
		})
	}
}
