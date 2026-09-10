package filter

import (
	"testing"
	"time"

	"github.com/JonasFranc1sco/vagouBot/internal/linkedin"
)

func TestFilterProcessMaxJobAgeDays(t *testing.T) {
	now := time.Now()

	jobs := []linkedin.Job{
		{ID: "1", Title: "Recente", PostedAt: now.AddDate(0, 0, -2)},
		{ID: "2", Title: "Antiga", PostedAt: now.AddDate(0, 0, -30)},
		{ID: "3", Title: "Sem data"},
	}

	cfg := Config{MaxJobAgeDays: 7}

	got := FilterProcess(jobs, cfg)
	if len(got) != 2 {
		t.Fatalf("esperava 2 vagas, obtive %d", len(got))
	}
	for _, j := range got {
		if j.ID == "2" {
			t.Fatalf("vaga antiga %q não deveria passar no filtro", j.ID)
		}
	}
}

func TestFilterProcessMaxJobAgeDaysDisabled(t *testing.T) {
	now := time.Now()
	jobs := []linkedin.Job{
		{ID: "1", Title: "Antiga", PostedAt: now.AddDate(0, 0, -90)},
		{ID: "2", Title: "Sem data"},
	}

	got := FilterProcess(jobs, Config{})
	if len(got) != 2 {
		t.Fatalf("filtro desabilitado deveria manter tudo, obtive %d", len(got))
	}
}
