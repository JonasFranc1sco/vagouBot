package linkedin

import (
	"net/url"
	"strings"
	"testing"
)

func TestBuildGuestSearchURL(t *testing.T) {
	params := SearchParams{
		Keywords:         "Junior Developer",
		Location:         "Brazil",
		WorkplaceTypes:   []string{"remote"},
		ExperienceLevels: []string{"internship", "entry_level"},
		PostTime:         "week",
		SortBy:           "R",
		Start:            0,
	}

	gotURL := BuildGuestSearchURL(params)
	parsed, err := url.Parse(gotURL)
	if err != nil {
		t.Fatalf("URL inválida: %v", err)
	}

	q := parsed.Query()
	if q.Get("keywords") != "Junior Developer" {
		t.Errorf("keywords esperado 'Junior Developer', obtido %q", q.Get("keywords"))
	}
	if q.Get("location") != "Brazil" {
		t.Errorf("location esperado 'Brazil', obtido %q", q.Get("location"))
	}
	if q.Get("geoId") != BrazilGeoID {
		t.Errorf("geoId esperado %q, obtido %q", BrazilGeoID, q.Get("geoId"))
	}
	if q.Get("f_WT") != "2" {
		t.Errorf("f_WT esperado '2', obtido %q", q.Get("f_WT"))
	}
	if q.Get("f_E") != "1,2" {
		t.Errorf("f_E esperado '1,2', obtido %q", q.Get("f_E"))
	}
	if q.Get("f_TPR") != "r604800" {
		t.Errorf("f_TPR esperado 'r604800', obtido %q", q.Get("f_TPR"))
	}
	if q.Get("sortBy") != "R" {
		t.Errorf("sortBy esperado 'R', obtido %q", q.Get("sortBy"))
	}
}

func TestResolveLocationRemoteLegacy(t *testing.T) {
	params := SearchParams{
		Keywords: "python junior",
		Location: "remote",
	}
	gotURL := BuildGuestSearchURL(params)
	parsed, err := url.Parse(gotURL)
	if err != nil {
		t.Fatalf("URL inválida: %v", err)
	}
	q := parsed.Query()
	if q.Get("location") != "Brazil" {
		t.Errorf("location 'remote' deveria ser convertido para 'Brazil', obtido %q", q.Get("location"))
	}
	if q.Get("geoId") != BrazilGeoID {
		t.Errorf("geoId esperado %q, obtido %q", BrazilGeoID, q.Get("geoId"))
	}
	if q.Get("f_WT") != "2" {
		t.Errorf("f_WT esperado '2' quando location='remote', obtido %q", q.Get("f_WT"))
	}
}

func TestParseVoyagerJobs(t *testing.T) {
	sampleJSON := `{
		"included": [
			{
				"entityUrn": "urn:li:fsd_jobPostingCard:(4123456789,JOB_DETAILS)",
				"jobPostingTitle": "Junior Python Developer",
				"primaryDescription": {"text": "Acme Tech"},
				"secondaryDescription": {"text": "Brasil (Remoto)"},
				"listedAt": 1730000000000
			},
			{
				"entityUrn": "urn:li:fsd_jobPosting:4987654321",
				"title": "Desenvolvedor Backend Júnior",
				"companyName": "Beta Corp",
				"formattedLocation": "São Paulo, Brasil"
			}
		]
	}`

	jobs, err := ParseVoyagerJobs(strings.NewReader(sampleJSON))
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if len(jobs) != 2 {
		t.Fatalf("esperava 2 vagas, obtive %d", len(jobs))
	}

	if jobs[0].ID != "4123456789" || jobs[0].Title != "Junior Python Developer" || jobs[0].Company != "Acme Tech" || jobs[0].Location != "Brasil (Remoto)" {
		t.Errorf("vaga 0 incorreta: %+v", jobs[0])
	}
	if jobs[0].PostedAt.IsZero() {
		t.Errorf("esperava PostedAt preenchido na vaga 0")
	}

	if jobs[1].ID != "4987654321" || jobs[1].Title != "Desenvolvedor Backend Júnior" || jobs[1].Company != "Beta Corp" {
		t.Errorf("vaga 1 incorreta: %+v", jobs[1])
	}
}

func TestClientSetSession(t *testing.T) {
	c := NewClient(nil, false, 10, 1)
	if c.HasSession() {
		t.Fatalf("não deveria ter sessão antes de SetSession")
	}

	c.SetSession("meu_cookie_li_at", "ajax:111222333")
	if !c.HasSession() {
		t.Fatalf("deveria ter sessão após SetSession")
	}
	if got := c.currentCSRFToken(); got != "ajax:111222333" {
		t.Errorf("csrfToken esperado 'ajax:111222333', obtido %q", got)
	}
}
