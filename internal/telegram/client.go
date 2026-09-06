package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/JonasFranc1sco/vagouBot/internal/linkedin"
)

type Client struct {
	token      string
	chatID     int64
	httpClient *http.Client
}

// NewClient cria um cliente do Telegram.
func NewClient(token string, chatID int64) *Client {
	return &Client{
		token:  token,
		chatID: chatID,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// sendMessage é chamada base pra enviar mensagens.
func (c *Client) sendMessage(ctx context.Context, text string, parseMode string) error {
	params := url.Values{}
	params.Set("chat_id", fmt.Sprintf("%d", c.chatID))
	params.Set("text", text)
	if parseMode != "" {
		params.Set("parse_mode", parseMode)
	}

	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", c.token)

	// POST com form-urlencoded
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		endpoint,
		bytes.NewBufferString(params.Encode()),
	)
	if err != nil {
		return fmt.Errorf("erro ao criar request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("erro ao enviar mensagem: %w", err)
	}
	defer resp.Body.Close()

	// Lê a resposta para verificar erros
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return fmt.Errorf("telegram respondeu %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("erro ao ler resposta: %w", err)
	}
	if !result.OK {
		return fmt.Errorf("telegram: %s", result.Description)
	}

	return nil
}

func (c *Client) SendJob(ctx context.Context, job interface{ FormatJob() string }) error {
	return c.sendMessage(ctx, job.FormatJob(), ParseModeHTML)
}

func (c *Client) SendText(ctx context.Context, text string) error {
	return c.sendMessage(ctx, text, "")
}

func (c *Client) SendHTML(ctx context.Context, text string) error {
	return c.sendMessage(ctx, text, ParseModeHTML)
}

func (c *Client) SendJobs(ctx context.Context, jobs []linkedin.Job) error {
	messages, err := BuildJobsMessage(jobs, 0)
	if err != nil {
		return err
	}

	for _, msg := range messages {
		if err := c.SendHTML(ctx, msg); err != nil {
			return err
		}

		time.Sleep(1 * time.Second)
	}

	return nil
}
