package filter

import (
	"regexp"
	"strings"
	"time"

	"github.com/JonasFranc1sco/vagouBot/internal/linkedin"
)

// Config define os critérios de filtragem.
type Config struct {
	IncludeKeywords      []string
	ExcludeKeywords      []string
	IncludeCompanies     []string
	ExcludeCompanies     []string
	IncludeLocations     []string
	ExcludeLocations     []string
	MinDescriptionLength int
	MaxJobAgeDays        int
}

func FilterProcess(jobs []linkedin.Job, cfg Config) []linkedin.Job {
	var result []linkedin.Job

	for _, job := range jobs {
		if shouldKeep(job, cfg) {
			result = append(result, job)
		}
	}
	return result
}

// shouldKeep verifica se uma vaga passa em todos os filtros.
func shouldKeep(job linkedin.Job, cfg Config) bool {
	titleLower := strings.ToLower(job.Title)
	descLower := strings.ToLower(job.Description)
	companyLower := strings.ToLower(job.Company)
	locationLower := strings.ToLower(job.Location)

	combined := titleLower + " " + descLower

	for _, kw := range cfg.ExcludeKeywords {
		if strings.Contains(combined, strings.ToLower(kw)) {
			return false
		}
	}

	for _, comp := range cfg.ExcludeCompanies {
		if strings.Contains(companyLower, strings.ToLower(comp)) {
			return false
		}
	}

	for _, loc := range cfg.ExcludeLocations {
		if strings.Contains(locationLower, strings.ToLower(loc)) {
			return false
		}
	}

	if len(cfg.IncludeCompanies) > 0 {
		matched := false
		for _, comp := range cfg.IncludeCompanies {
			if strings.Contains(companyLower, strings.ToLower(comp)) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	if len(cfg.IncludeLocations) > 0 {
		matched := false
		for _, loc := range cfg.IncludeLocations {
			if strings.Contains(locationLower, strings.ToLower(loc)) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	if cfg.MinDescriptionLength > 0 && len(job.Description) < cfg.MinDescriptionLength {
		return false
	}

	if cfg.MaxJobAgeDays > 0 && !job.PostedAt.IsZero() &&
		int(time.Since(job.PostedAt).Hours()/24) > cfg.MaxJobAgeDays {
		return false
	}

	for _, pattern := range cfg.IncludeKeywords {
		if pattern == "" {
			continue
		}

		if isRegex(pattern) {
			re, err := regexp.Compile("(?i)" + strings.Trim(pattern, "/"))
			if err != nil {
				continue
			}
			if !re.MatchString(combined) {
				return false
			}
			continue
		}
		if !strings.Contains(combined, strings.ToLower(pattern)) {
			return false
		}
	}
	return true

}

// isRegex verifica se uma string parece ser um padrão regex.
func isRegex(s string) bool {
	return strings.HasPrefix(s, "/") && strings.HasSuffix(s, "/")
}
