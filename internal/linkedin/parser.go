package linkedin

import (
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

	postedDate := strings.TrimSpace(s.Find("time.job-search-card__listdate").Text())

	description := strings.TrimSpace(s.Find(".artdeco-entity-snippet__subtitle-link, .search-entity-description").Text())

	jobID := extractJobID(link)

	return Job{
		ID:          jobID,
		Title:       title,
		Company:     company,
		Location:    location,
		URL:         link,
		PostedDate:  postedDate,
		Description: description,
		ScrapedAt:   time.Now(),
	}
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

	parts := strings.Split(strings.TrimRight(url, "/"), "/")
	if len(parts) == 0 {
		return ""
	}

	return parts[len(parts)-1]
}
