package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
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
			// Longo o suficiente para o long-polling do getUpdates
			// (timeout de até 50s) e para uploads de documentos.
			Timeout: 90 * time.Second,
		},
	}
}

// ChatID retorna o chat configurado para o bot.
func (c *Client) ChatID() int64 {
	return c.chatID
}

// sendMessage é chamada base pra enviar mensagens.
func (c *Client) sendMessage(ctx context.Context, text string, parseMode string, replyTo int64) error {
	params := url.Values{}
	params.Set("chat_id", fmt.Sprintf("%d", c.chatID))
	params.Set("text", text)
	if parseMode != "" {
		params.Set("parse_mode", parseMode)
	}
	if replyTo != 0 {
		params.Set("reply_to_message_id", strconv.FormatInt(replyTo, 10))
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

func (c *Client) SendText(ctx context.Context, text string) error {
	return c.sendMessage(ctx, text, "", 0)
}

func (c *Client) SendHTML(ctx context.Context, text string) error {
	return c.sendMessage(ctx, text, ParseModeHTML, 0)
}

// SendReply responde a uma mensagem específica (reply_to_message_id).
func (c *Client) SendReply(ctx context.Context, replyToID int64, text string) error {
	return c.sendMessage(ctx, text, "", replyToID)
}

// SendChatAction avisa o Telegram que o bot está "fazendo algo",
// mantendo o indicador de "digitando/enviando documento" na conversa.
func (c *Client) SendChatAction(ctx context.Context, action string) error {
	params := url.Values{}
	params.Set("chat_id", fmt.Sprintf("%d", c.chatID))
	params.Set("action", action)

	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/sendChatAction", c.token)
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
		return fmt.Errorf("erro ao enviar chat action: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram respondeu %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (c *Client) SendJobs(ctx context.Context, jobs []linkedin.Job) error {
	for _, job := range jobs {
		msg := FormatJobNotification(job)

		if err := c.SendHTML(ctx, msg); err != nil {
			return err
		}

		time.Sleep(1 * time.Second)
	}
	return nil
}

func (c *Client) SendDocument(ctx context.Context, fileName string, content []byte, caption string) error {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// chat_id como campo do form
	if err := writer.WriteField("chat_id", fmt.Sprintf("%d", c.chatID)); err != nil {
		return fmt.Errorf("escrever chat_id: %w", err)
	}

	part, err := writer.CreateFormFile("document", fileName)
	if err != nil {
		return fmt.Errorf("criar campo document: %w", err)
	}
	if _, err := part.Write(content); err != nil {
		return fmt.Errorf("escrever arquivo: %w", err)
	}

	if caption != "" {
		if err := writer.WriteField("caption", caption); err != nil {
			return fmt.Errorf("escrever caption: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("fechar multipart: %w", err)
	}

	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/sendDocument", c.token)
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, &body)
	if err != nil {
		return fmt.Errorf("erro ao criar request: %w", err)
	}

	// Content-Type com o boundary gerado pelo multipart
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("erro ao enviar documento: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram respondeu %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("erro ao ler resposta: %w", err)
	}
	if !result.OK {
		return fmt.Errorf("telegram: %s", result.Description)
	}
	return nil
}
