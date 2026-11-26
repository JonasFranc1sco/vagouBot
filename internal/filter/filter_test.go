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

func TestDedupFilterNewBatchAndRepost(t *testing.T) {
	tmpFile := t.TempDir() + "/seen.json"
	dedup := NewDedup(tmpFile)

	batch := []linkedin.Job{
		{ID: "100", Title: "Junior Developer", Company: "Empresa A"},
		{ID: "100", Title: "Junior Developer", Company: "Empresa A"}, // Mesmo ID no mesmo lote
		{ID: "101", Title: "Junior Developer", Company: "Empresa A"}, // Repost (ID novo, mesma empresa + título)
		{ID: "102", Title: "Backend Junior", Company: "Empresa B"},
	}

	got := dedup.FilterNew(batch)
	if len(got) != 2 {
		t.Fatalf("esperava 2 vagas únicas no lote, obtive %d", len(got))
	}

	dedup.MarkAll(got)

	// Nova execução com outra vaga repostada da Empresa A
	nextRun := []linkedin.Job{
		{ID: "999", Title: "Junior Developer", Company: "Empresa A"},
		{ID: "103", Title: "Python Junior", Company: "Empresa C"},
	}
	gotNext := dedup.FilterNew(nextRun)
	if len(gotNext) != 1 || gotNext[0].ID != "103" {
		t.Fatalf("esperava apenas a vaga 103 como inédita, obtive %+v", gotNext)
	}
}

func TestPreFilterProcess(t *testing.T) {
	jobs := []linkedin.Job{
		{ID: "1", Title: "Junior Python Developer", Company: "Startup BR", Location: "Brasil (Remoto)"},
		{ID: "2", Title: "Senior Python Engineer", Company: "Startup BR", Location: "Brasil (Remoto)"},
		{ID: "3", Title: "Junior Python Developer", Company: "US Corp", Location: "United States"},
	}

	cfg := Config{
		ExcludeKeywords:  []string{"senior"},
		IncludeLocations: []string{"brasil", "brazil"},
		ExcludeLocations: []string{"united states"},
	}

	got := PreFilterProcess(jobs, cfg)
	if len(got) != 1 || got[0].ID != "1" {
		t.Fatalf("esperava apenas a vaga 1 após pré-filtro, obtive %+v", got)
	}
}
