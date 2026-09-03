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
	Description    string
	SeniorityLevel string
	EmploymentType string
	JobFunction    string
	Industries     string
	ScrapedAt      time.Time
}

// SearchParams define parâmetros de busca.
type SearchParams struct {
	Keywords   string
	Location   string
	Start      int
	MaxResults int
}
