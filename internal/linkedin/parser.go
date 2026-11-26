package linkedin

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// ParseJobs recebe o HTML da busca e retorna uma lista de Jobs.
func ParseJobs(reader io.Reader) ([]Job, error) {
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return nil, fmt.Errorf("erro ao parsear HTML: %w", err)
	}

	var jobs []Job

	// Separa vagas <li> da classe de resultados do HTML
	doc.Find("[data-entity-urn]").Each(func(i int, s *goquery.Selection) {
		job := parseJobCard(s)
		if job.ID != "" {
			jobs = append(jobs, job)
		}
	})

	if len(jobs) == 0 {
		doc.Find("a[href*='/jobs/view/']").Each(func(i int, s *goquery.Selection) {
			href, _ := s.Attr("href")
			title := strings.TrimSpace(s.Find(".base-search-card__title, h3, h4").Text())
			if title == "" {
				title = strings.TrimSpace(s.Text())
			}
			jobID := extractJobID(href)
			if jobID != "" && title != "" {
				jobs = append(jobs, Job{
					ID:        jobID,
					Title:     title,
					URL:       href,
					ScrapedAt: time.Now(),
				})
			}
		})
	}
	return jobs, nil
}

// parseJobCard extrai os dados de um único card de vaga.
func parseJobCard(s *goquery.Selection) Job {
	title := strings.TrimSpace(s.Find(".base-search-card__title").Text())
	if title == "" {
		title = strings.TrimSpace(s.Find("h3").Text())
	}
	company := strings.TrimSpace(s.Find(".base-search-card__subtitle").Text())
	if company == "" {
		company = strings.TrimSpace(s.Find("h4").Text())
	}
	location := strings.TrimSpace(s.Find(".job-search-card__location").Text())

	link, _ := s.Find("a.base-card__full-link, a[href*='/jobs/view/']").Attr("href")

	postedSel := s.Find("time[class*='job-search-card__listdate']")
	postedDate := strings.TrimSpace(postedSel.Text())
	postedAt := time.Time{}
	if datetime, ok := postedSel.Attr("datetime"); ok {
		postedAt = parseAbsoluteDate(datetime)
	}
	if postedAt.IsZero() {
		postedAt, _ = parsePostedDate(postedDate, time.Now())
	}

	description := strings.TrimSpace(s.Find(".artdeco-entity-snippet__subtitle-link, .search-entity-description").Text())

	jobID := extractURNJobID(s)
	if jobID == "" {
		jobID = extractJobID(link)
	}

	cleanURL := cleanJobURL(link, jobID)

	return Job{
		ID:          jobID,
		Title:       title,
		Company:     company,
		Location:    location,
		URL:         cleanURL,
		PostedDate:  postedDate,
		PostedAt:    postedAt,
		Description: description,
		ScrapedAt:   time.Now(),
	}
}

// extractURNJobID extrai o ID numérico diretamente do atributo data-entity-urn="urn:li:jobPosting:12345"
func extractURNJobID(s *goquery.Selection) string {
	urn, ok := s.Attr("data-entity-urn")
	if !ok || urn == "" {
		urn, _ = s.Find("[data-entity-urn]").First().Attr("data-entity-urn")
	}
	if urn == "" {
		return ""
	}
	parts := strings.Split(urn, ":")
	id := strings.TrimSpace(parts[len(parts)-1])
	for _, r := range id {
		if r < '0' || r > '9' {
			return ""
		}
	}
	return id
}

func cleanJobURL(rawURL, jobID string) string {
	rawURL = strings.TrimSpace(rawURL)
	if idx := strings.Index(rawURL, "?"); idx != -1 {
		rawURL = rawURL[:idx]
	}
	if rawURL == "" && jobID != "" {
		return "https://www.linkedin.com/jobs/view/" + jobID
	}
	return rawURL
}

// extractJobID pega o ID numérico da vaga a partir da URL

func extractJobID(url string) string {
	if url == "" {
		return ""
	}

	// Remove query params
	if idx := strings.Index(url, "?"); idx != -1 {
		url = url[:idx]
	}

	url = strings.TrimRight(url, "/")
	parts := strings.Split(url, "/")
	if len(parts) == 0 {
		return ""
	}

	lastSegment := parts[len(parts)-1]

	slugParts := strings.Split(lastSegment, "-")
	if len(slugParts) == 0 {
		return ""
	}

	return slugParts[len(slugParts)-1]
}

// FetchDetail busca os detalhes completos de uma vaga
func (c *Client) FetchDetail(ctx context.Context, job *Job) error {
	if job.ID == "" {
		return fmt.Errorf("ID vazio, impossível buscar detalhe")
	}

	// ID pode conter query params, precisa ser limpado
	detailID := job.ID
	if idx := strings.Index(detailID, "?"); idx != -1 {
		detailID = detailID[:idx]
	}

	detailURL := fmt.Sprintf("https://www.linkedin.com/jobs-guest/jobs/api/jobPosting/%s", detailID)

	resp, err := c.DoRequest(ctx, detailURL)
	if err != nil {
		return fmt.Errorf("Erro ao buscar detalhe: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("status %d no detalhe", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("erro ao ler detalhe: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("erro ao parsear detalhe: %w", err)
	}

	if job.Title == "" {
		job.Title = strings.TrimSpace(doc.Find(".top-card-layout__title, h2.top-card-layout__title").First().Text())
	}
	if job.Company == "" {
		job.Company = strings.TrimSpace(doc.Find(".topcard__org-name-link, .topcard__flavor").First().Text())
	}
	if job.Location == "" {
		job.Location = strings.TrimSpace(doc.Find(".topcard__flavor--bullet").First().Text())
	}

	// Descrição completa
	job.Description = strings.TrimSpace(doc.Find(".description__text--rich, .show-more-less-html__markup").First().Text())

	// Critérios (suporta labels em inglês e português)
	doc.Find(".description__job-criteria-item").Each(func(i int, s *goquery.Selection) {
		criteria := strings.TrimSpace(s.Find(".description__job-criteria-subheader").Text())
		value := strings.TrimSpace(s.Find(".description__job-criteria-text").Text())

		switch strings.ToLower(criteria) {
		case "seniority level", "nível de experiência", "nivel de experiencia":
			job.SeniorityLevel = value
		case "employment type", "tipo de emprego":
			job.EmploymentType = value
		case "job function", "função", "funcao":
			job.JobFunction = value
		case "industries", "setores":
			job.Industries = value
		}
	})

	return nil
}
