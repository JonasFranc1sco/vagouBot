package linkedin

import "time"

// Job representa uma vaga extraída do LinkedIn
type Job struct {
	ID             string
	Title          string
	Company        string
	Location       string
	URL            string
	PostedDate     string
	PostedAt       time.Time
	Description    string
	SeniorityLevel string
	EmploymentType string
	JobFunction    string
	Industries     string
	ScrapedAt      time.Time
}

// SearchParams define parâmetros estruturados de busca no LinkedIn.
type SearchParams struct {
	Keywords         string
	Location         string
	GeoID            string
	WorkplaceTypes   []string
	ExperienceLevels []string
	PostTime         string
	SortBy           string
	Start            int
	MaxResults       int
}
