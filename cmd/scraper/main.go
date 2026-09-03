package main

import (
	"fmt"
	"io"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/JonasFranc1sco/vagouBot/internal/config"
	"github.com/JonasFranc1sco/vagouBot/internal/linkedin"
)

func main() {
	// Carrega config
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("Erro ao carregar config: %v", err)
	}

	// Cria client
	client := linkedin.NewClient(
		cfg.Proxy.URLs,
		cfg.Proxy.Enabled,
		cfg.RateLimit.RequestsPerSecond,
		cfg.RateLimit.Burst,
	)

	// Monta URL de busca
	searchURL := fmt.Sprintf("https://www.linkedin.com/jobs-guest/jobs/api/seeMoreJobPostings/search?keywords=%s&location=%s&start=0",
		url.QueryEscape(cfg.LinkedIn.Keywords[0]),
		url.QueryEscape(cfg.LinkedIn.Location),
	)

	fmt.Printf("Buscando: %s\n", searchURL)

	// Executa request
	resp, err := client.DoRequest(searchURL)
	if err != nil {
		log.Fatalf("Erro no request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Tamanho: %d bytes\n", len(body))

	jobs, err := linkedin.ParseJobs(strings.NewReader(string(body)))
	if err != nil {
		log.Fatalf("Erro ao parsear vagas: %v", err)
	}

	for i := range jobs {
		if jobs[i].ID == "" {
			continue
		}
		// Rate limit para os requests
		if i > 0 {
			time.Sleep(2 * time.Second)
		}

		fmt.Printf("Buscando detalhe [%d/%d]: %s\n", i+1, len(jobs[i].Description))

		err := client.FetchDetail(&jobs[i])
		if err != nil {
			fmt.Printf("Erro: %v\n", err)
			continue
		}

		fmt.Printf("OK = %d caracteres de descrição\n", len(jobs[i].Description))
	}

	for i, job := range jobs {
		if i >= 3 {
			break
		}
		fmt.Printf("🏢 %s\n", job.Company)
		fmt.Printf("💼 %s\n", job.Title)
		fmt.Printf("📍 %s\n", job.Location)
		fmt.Printf("📅 %s\n", job.PostedDate)
		fmt.Printf("%s | %s\n", job.SeniorityLevel, job.EmploymentType)
		fmt.Printf("🔗 %s\n", job.URL)
		fmt.Printf("🆔 %s\n", job.ID)
		if job.Description != "" {
			desc := job.Description
			if len(desc) > 300 {
				desc = desc[:300] + "..."
			}
			fmt.Printf("%s\n", desc)
		}
		fmt.Println("---")

	}
}

// min retorna o menor de dois inteiros
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
