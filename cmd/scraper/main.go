package main

import (
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/url"
	"strings"

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
	client := linkedin.NewClient()

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

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Erro ao ler body: %v", err)
	}

	var body []byte
	if strings.Contains(resp.Header.Get("Content-Encoding"), "gzip") {
		gz, err := gzip.NewReader(strings.NewReader(string(rawBody)))
		if err != nil {
			log.Fatalf("Erro ao criar gzip reader: %v", err)
		}
		defer gz.Close()
		body, err = io.ReadAll(gz)
		if err != nil {
			log.Fatalf("Erro ao descomprimir: %v", err)
		}
	} else {
		body = rawBody
	}

	jobs, err := linkedin.ParseJobs(strings.NewReader(string(body)))
	if err != nil {
		log.Fatalf("Erro ao parsear vagas: %v", err)
	}

	for i, job := range jobs {
		if i >= 5 {
			break
		}
		fmt.Printf("🏢 %s\n", job.Company)
		fmt.Printf("💼 %s\n", job.Title)
		fmt.Printf("📍 %s\n", job.Location)
		fmt.Printf("📅 %s\n", job.PostedDate)
		fmt.Printf("%s\n", job.Description)
		fmt.Printf("🔗 %s\n", job.URL)
		fmt.Printf("🆔 %s\n", job.ID)
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
