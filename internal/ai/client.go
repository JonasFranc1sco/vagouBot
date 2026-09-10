package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/JonasFranc1sco/vagouBot/internal/config"
	"github.com/JonasFranc1sco/vagouBot/internal/linkedin"
)

// temperature baixa: voltado a seguir o schema do prompt
const temperature = 0.2

// Client encapsula a comunicação com a API OpenAI-compatible.
type Client struct {
	baseURL    string
	apiKey     string
	model      string
	maxTokens  int
	httpClient *http.Client
}

func NewClient(baseURL, apiKey, model string, maxTokens int) *Client {
	return &Client{
		baseURL:   baseURL,
		apiKey:    apiKey,
		model:     model,
		maxTokens: maxTokens,

		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// GenerateResumes faz a chamada para gerar o currículo de todas as vagas do lote
func (c *Client) GenerateResumes(ctx context.Context, jobs []linkedin.Job, profile config.ProfileConfig) (Resumes, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("Chave de API da IA não configurada (AI_API_KEY)")
	}
	if len(jobs) == 0 {
		return Resumes{}, nil
	}

	msgs, err := BuildMessages(jobs, profile)
	if err != nil {
		return nil, err
	}

	// Corpo da requisição
	reqBody := chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: msgs.System},
			{Role: "user", Content: msgs.User},
		},
		MaxTokens:   c.maxTokens,
		Temperature: temperature,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("serializar request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("criar request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("chamar IA: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("IA respondeu %d: %s", resp.StatusCode, strings.TrimSpace(string(errBody)))
	}

	var chatResp chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("ler resposta da IA: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("IA não retornou choices")
	}

	// finish_reason "lenght"
	if chatResp.Choices[0].FinishReason == "length" {
		return nil, fmt.Errorf("resposta truncada pelo max_tokens")
	}

	var resumes Resumes
	content := chatResp.Choices[0].Message.Content
	if err := json.Unmarshal([]byte(content), &resumes); err != nil {
		return nil, fmt.Errorf("IA não devolveu JSON válido: %w", err)
	}

	// Toda vaga do lote precisa ter currículo na resposta.
	for _, job := range jobs {
		if _, ok := resumes[job.ID]; !ok {
			return nil, fmt.Errorf("resposta da IA sem currículo para o job %s", job.ID)
		}
	}

	return resumes, nil
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
}

// Só os campos que nos interessam da resposta
type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}
