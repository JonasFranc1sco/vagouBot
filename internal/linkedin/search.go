package linkedin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	BrazilGeoID = "106057199"
)

var reNumericID = regexp.MustCompile(`(\d{6,})`)

// FTPR converte a configuração post_time no parâmetro f_TPR do LinkedIn.
// "" = qualquer data; "day" = últimas 24h; "week" = última semana; "month" = último mês.
func FTPR(postTime string) string {
	v := strings.ToLower(strings.TrimSpace(postTime))
	switch v {
	case "day", "24h", "r86400":
		return "r86400"
	case "week", "r604800":
		return "r604800"
	case "month", "r2592000":
		return "r2592000"
	default:
		if strings.HasPrefix(v, "r") && len(v) > 1 {
			return v
		}
		return ""
	}
}

// MapWorkplaceTypes converte nomes amigáveis ("remote", "hybrid", "onsite")
// nos códigos do parâmetro f_WT do LinkedIn ("1"=On-site, "2"=Remote, "3"=Hybrid).
func MapWorkplaceTypes(types []string) string {
	var codes []string
	seen := make(map[string]bool)

	for _, t := range types {
		var code string
		switch strings.ToLower(strings.TrimSpace(t)) {
		case "1", "onsite", "on-site", "presencial":
			code = "1"
		case "2", "remote", "remoto":
			code = "2"
		case "3", "hybrid", "hibrido", "híbrido":
			code = "3"
		}
		if code != "" && !seen[code] {
			seen[code] = true
			codes = append(codes, code)
		}
	}
	return strings.Join(codes, ",")
}

// MapExperienceLevels converte níveis amigáveis ("internship", "entry_level", "junior", etc.)
// nos códigos do parâmetro f_E do LinkedIn ("1" a "6").
func MapExperienceLevels(levels []string) string {
	var codes []string
	seen := make(map[string]bool)

	for _, l := range levels {
		var code string
		switch strings.ToLower(strings.TrimSpace(l)) {
		case "1", "internship", "intern", "estagio", "estágio":
			code = "1"
		case "2", "entry_level", "entry", "entry-level", "junior", "júnior", "jr", "assistente":
			code = "2"
		case "3", "associate", "associado":
			code = "3"
		case "4", "mid_senior", "mid-senior", "pleno", "senior", "sênior":
			code = "4"
		case "5", "director", "diretor":
			code = "5"
		case "6", "executive", "executivo":
			code = "6"
		}
		if code != "" && !seen[code] {
			seen[code] = true
			codes = append(codes, code)
		}
	}
	return strings.Join(codes, ",")
}

// MapSortBy normaliza o parâmetro de ordenação ("R" para relevância, "DD" para mais recentes).
func MapSortBy(sortBy string) string {
	switch strings.ToLower(strings.TrimSpace(sortBy)) {
	case "dd", "date", "recent", "recentes":
		return "DD"
	case "r", "relevance", "relevancia", "relevância":
		return "R"
	default:
		return ""
	}
}

// ResolveLocationAndGeoID corrige configurações onde location foi definido como "remote"
// e preenche automaticamente o GeoID do Brasil quando aplicável.
func ResolveLocationAndGeoID(location, geoID string, workplaceTypes []string) (string, string, []string) {
	locTrim := strings.TrimSpace(location)
	locLower := strings.ToLower(locTrim)
	geoTrim := strings.TrimSpace(geoID)

	// Se o usuário colocou location: "remote", corrige para Brasil + filtro f_WT=remote
	if locLower == "remote" || locLower == "remoto" {
		locTrim = "Brazil"
		locLower = "brazil"
		hasRemote := false
		for _, wt := range workplaceTypes {
			if strings.EqualFold(wt, "remote") || strings.EqualFold(wt, "remoto") || wt == "2" {
				hasRemote = true
				break
			}
		}
		if !hasRemote {
			workplaceTypes = append(workplaceTypes, "remote")
		}
	}

	if geoTrim == "" && (locLower == "brazil" || locLower == "brasil") {
		geoTrim = BrazilGeoID
	}

	return locTrim, geoTrim, workplaceTypes
}

// BuildGuestSearchURL monta a URL de busca da API Guest do LinkedIn com todos os filtros estruturados.
func BuildGuestSearchURL(params SearchParams) string {
	location, geoID, workplaceTypes := ResolveLocationAndGeoID(params.Location, params.GeoID, params.WorkplaceTypes)

	q := url.Values{}
	if strings.TrimSpace(params.Keywords) != "" {
		q.Set("keywords", strings.TrimSpace(params.Keywords))
	}
	if location != "" {
		q.Set("location", location)
	}
	if geoID != "" {
		q.Set("geoId", geoID)
	}
	if ftpr := FTPR(params.PostTime); ftpr != "" {
		q.Set("f_TPR", ftpr)
	}
	if fwt := MapWorkplaceTypes(workplaceTypes); fwt != "" {
		q.Set("f_WT", fwt)
	}
	if fe := MapExperienceLevels(params.ExperienceLevels); fe != "" {
		q.Set("f_E", fe)
	}
	if sortBy := MapSortBy(params.SortBy); sortBy != "" {
		q.Set("sortBy", sortBy)
	}
	q.Set("start", strconv.Itoa(params.Start))

	return "https://www.linkedin.com/jobs-guest/jobs/api/seeMoreJobPostings/search?" + q.Encode()
}

// BuildVoyagerRecommendedURL monta a URL da API Voyager para buscar vagas recomendadas ao perfil (JYMBII).
func BuildVoyagerRecommendedURL(start, count int) string {
	if count <= 0 {
		count = 25
	}
	return fmt.Sprintf(
		"https://www.linkedin.com/voyager/api/voyagerJobsDashJobCards?decorationId=com.linkedin.voyager.dash.deco.jobs.search.JobSearchCardsCollection-220&count=%d&q=jobSearch&query=(origin:JOB_HOME_JYMBII,sponsored:List())&start=%d",
		count, start,
	)
}

// BuildVoyagerSearchURL monta a URL da API Voyager para busca autenticada personalizada pelo perfil.
func BuildVoyagerSearchURL(params SearchParams, count int) string {
	if count <= 0 {
		count = 25
	}
	_, geoID, workplaceTypes := ResolveLocationAndGeoID(params.Location, params.GeoID, params.WorkplaceTypes)

	var filters []string
	if ftpr := FTPR(params.PostTime); ftpr != "" {
		filters = append(filters, fmt.Sprintf("timePostedRange:List(%s)", ftpr))
	}
	if fwt := MapWorkplaceTypes(workplaceTypes); fwt != "" {
		filters = append(filters, fmt.Sprintf("workplaceType:List(%s)", fwt))
	}
	if fe := MapExperienceLevels(params.ExperienceLevels); fe != "" {
		filters = append(filters, fmt.Sprintf("experience:List(%s)", fe))
	}
	if sortBy := MapSortBy(params.SortBy); sortBy != "" {
		filters = append(filters, fmt.Sprintf("sortBy:List(%s)", sortBy))
	}

	var queryParts []string
	queryParts = append(queryParts, "origin:JOB_SEARCH_PAGE_JOB_FILTER")
	if kw := strings.TrimSpace(params.Keywords); kw != "" {
		queryParts = append(queryParts, "keywords:"+url.QueryEscape(kw))
	}
	if geoID != "" {
		queryParts = append(queryParts, fmt.Sprintf("locationUnion:(geoId:%s)", geoID))
	}
	if len(filters) > 0 {
		queryParts = append(queryParts, fmt.Sprintf("selectedFilters:(%s)", strings.Join(filters, ",")))
	}
	queryParts = append(queryParts, "spellCorrectionEnabled:true")

	return fmt.Sprintf(
		"https://www.linkedin.com/voyager/api/voyagerJobsDashJobCards?decorationId=com.linkedin.voyager.dash.deco.jobs.search.JobSearchCardsCollection-220&count=%d&q=jobSearch&query=(%s)&start=%d",
		count, strings.Join(queryParts, ","), params.Start,
	)
}

// ParseVoyagerJobs extrai vagas do JSON retornado pela API Voyager do LinkedIn.
func ParseVoyagerJobs(r io.Reader) ([]Job, error) {
	var payload struct {
		Included []map[string]any `json:"included"`
		Elements []map[string]any `json:"elements"`
		Data     map[string]any   `json:"data"`
	}

	if err := json.NewDecoder(r).Decode(&payload); err != nil {
		return nil, fmt.Errorf("erro ao decodificar JSON Voyager: %w", err)
	}

	items := payload.Included
	if len(items) == 0 && len(payload.Elements) > 0 {
		items = payload.Elements
	}

	byID := make(map[string]*Job)
	var order []string

	for _, item := range items {
		job := extractJobFromVoyagerMap(item)
		if job.ID == "" {
			continue
		}
		existing, ok := byID[job.ID]
		if !ok {
			cp := job
			byID[job.ID] = &cp
			order = append(order, job.ID)
			continue
		}
		// Mescla campos caso venham separados entre JobPosting e JobPostingCard
		if existing.Title == "" && job.Title != "" {
			existing.Title = job.Title
		}
		if existing.Company == "" && job.Company != "" {
			existing.Company = job.Company
		}
		if existing.Location == "" && job.Location != "" {
			existing.Location = job.Location
		}
		if existing.PostedAt.IsZero() && !job.PostedAt.IsZero() {
			existing.PostedAt = job.PostedAt
			existing.PostedDate = job.PostedDate
		}
		if existing.Description == "" && job.Description != "" {
			existing.Description = job.Description
		}
	}

	var jobs []Job
	for _, id := range order {
		j := byID[id]
		if j.Title != "" {
			jobs = append(jobs, *j)
		}
	}
	return jobs, nil
}

func extractJobFromVoyagerMap(m map[string]any) Job {
	urn := stringField(m, "entityUrn")
	if urn == "" {
		urn = stringField(m, "jobPostingUrn")
	}
	if urn == "" {
		urn = stringField(m, "preDashNormalizedJobPostingUrn")
	}

	if !strings.Contains(urn, "jobPosting") && !strings.Contains(urn, "JobPosting") {
		return Job{}
	}

	match := reNumericID.FindStringSubmatch(urn)
	if len(match) < 2 {
		return Job{}
	}
	jobID := match[1]

	title := extractTextValue(m["title"])
	if title == "" {
		title = extractTextValue(m["jobPostingTitle"])
	}

	company := extractTextValue(m["primaryDescription"])
	if company == "" {
		company = extractTextValue(m["companyName"])
	}

	location := extractTextValue(m["secondaryDescription"])
	if location == "" {
		location = extractTextValue(m["formattedLocation"])
	}

	var postedAt time.Time
	var postedDate string
	if listedAt, ok := floatField(m, "listedAt"); ok && listedAt > 0 {
		postedAt = time.UnixMilli(int64(listedAt)).UTC()
		postedDate = postedAt.Format("2006-01-02")
	}

	return Job{
		ID:         jobID,
		Title:      strings.TrimSpace(title),
		Company:    strings.TrimSpace(company),
		Location:   strings.TrimSpace(location),
		URL:        "https://www.linkedin.com/jobs/view/" + jobID,
		PostedDate: postedDate,
		PostedAt:   postedAt,
		ScrapedAt:  time.Now(),
	}
}

func stringField(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func floatField(m map[string]any, key string) (float64, bool) {
	switch v := m[key].(type) {
	case float64:
		return v, true
	case int64:
		return float64(v), true
	case int:
		return float64(v), true
	}
	return 0, false
}

func extractTextValue(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case map[string]any:
		if t, ok := val["text"].(string); ok {
			return t
		}
	}
	return ""
}

// SearchJobs busca vagas usando a sessão autenticada (Voyager API + fallback Guest API)
// ou diretamente a Guest API caso o cookie li_at não esteja configurado.
func (c *Client) SearchJobs(ctx context.Context, params SearchParams) ([]Job, error) {
	maxResults := params.MaxResults
	if maxResults <= 0 {
		maxResults = 25
	}

	if c.HasSession() {
		jobs, err := c.searchVoyager(ctx, params, maxResults)
		if err == nil && len(jobs) > 0 {
			return jobs, nil
		}
	}

	return c.searchGuest(ctx, params, maxResults)
}

func (c *Client) searchVoyager(ctx context.Context, params SearchParams, maxResults int) ([]Job, error) {
	var allJobs []Job

	for start := 0; start < maxResults; start += 25 {
		if ctx.Err() != nil {
			return allJobs, nil
		}
		p := params
		p.Start = start
		reqURL := BuildVoyagerSearchURL(p, 25)

		resp, err := c.DoVoyagerRequest(ctx, reqURL)
		if err != nil {
			return allJobs, err
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return allJobs, readErr
		}
		if resp.StatusCode != 200 {
			return allJobs, fmt.Errorf("voyager status %d", resp.StatusCode)
		}

		jobs, err := ParseVoyagerJobs(strings.NewReader(string(body)))
		if err != nil {
			return allJobs, err
		}
		allJobs = append(allJobs, jobs...)
		if len(jobs) == 0 {
			break
		}
	}
	return allJobs, nil
}

func (c *Client) searchGuest(ctx context.Context, params SearchParams, maxResults int) ([]Job, error) {
	var allJobs []Job

	for start := 0; start < maxResults; start += 25 {
		if ctx.Err() != nil {
			return allJobs, nil
		}
		p := params
		p.Start = start
		reqURL := BuildGuestSearchURL(p)

		resp, err := c.DoRequest(ctx, reqURL)
		if err != nil {
			return allJobs, err
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return allJobs, readErr
		}
		if resp.StatusCode != 200 {
			return allJobs, fmt.Errorf("guest search status %d", resp.StatusCode)
		}

		jobs, err := ParseJobs(strings.NewReader(string(body)))
		if err != nil {
			return allJobs, err
		}
		allJobs = append(allJobs, jobs...)
		if len(jobs) == 0 {
			break
		}
	}
	return allJobs, nil
}

// FetchRecommendedJobs busca as vagas recomendadas especificamente para o perfil logado (JYMBII).
// Requer que o cookie li_at esteja configurado no Client.
func (c *Client) FetchRecommendedJobs(ctx context.Context, maxResults int) ([]Job, error) {
	if !c.HasSession() {
		return nil, fmt.Errorf("sessão autenticada (li_at) não configurada")
	}
	if maxResults <= 0 {
		maxResults = 25
	}

	var allJobs []Job
	for start := 0; start < maxResults; start += 25 {
		if ctx.Err() != nil {
			return allJobs, nil
		}
		reqURL := BuildVoyagerRecommendedURL(start, 25)
		resp, err := c.DoVoyagerRequest(ctx, reqURL)
		if err != nil {
			return allJobs, err
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return allJobs, readErr
		}
		if resp.StatusCode != 200 {
			return allJobs, fmt.Errorf("voyager recommended status %d", resp.StatusCode)
		}

		jobs, err := ParseVoyagerJobs(strings.NewReader(string(body)))
		if err != nil {
			return allJobs, err
		}
		allJobs = append(allJobs, jobs...)
		if len(jobs) == 0 {
			break
		}
	}
	return allJobs, nil
}
