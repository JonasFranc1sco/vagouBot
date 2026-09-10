package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

// Update representa uma atualização recebida via getUpdates.
type Update struct {
	UpdateID int      `json:"update_id"`
	Message  *Message `json:"message"`
}

// Message representa uma mensagem do Telegram.
type Message struct {
	MessageID      int64           `json:"message_id"`
	Chat           *Chat           `json:"chat"`
	Text           string          `json:"text"`
	Entities       []MessageEntity `json:"entities,omitempty"`
	ReplyToMessage *Message        `json:"reply_to_message"`
}

// MessageEntity é uma entidade formatada dentro do texto da mensagem.
// Neste projeto interessa especialmente o tipo "text_link", cujo campo URL
// guarda o endereço do link clicável (ex.: a vaga do LinkedIn).
type MessageEntity struct {
	Type   string `json:"type"`
	Offset int    `json:"offset"`
	Length int    `json:"length"`
	URL    string `json:"url,omitempty"`
}

// Chat identifica o chat de onde a mensagem veio.
type Chat struct {
	ID int64 `json:"id"`
}

// GetUpdates faz long-polling: bloqueia até chegar uma atualização
// (ou o timeout) e retorna as atualizações a partir do offset informado.
// offset deve ser 0 na primeira chamada e depois o id da última
// atualização processada + 1.
func (c *Client) GetUpdates(ctx context.Context, offset, timeout int) ([]Update, error) {
	params := url.Values{}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}
	if timeout > 0 {
		params.Set("timeout", strconv.Itoa(timeout))
	}

	endpoint := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates", c.token)
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro no long-polling: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler resposta: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("telegram respondeu %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		OK          bool     `json:"ok"`
		Description string   `json:"description"`
		Result      []Update `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("erro ao decodificar updates: %w", err)
	}
	if !result.OK {
		return nil, fmt.Errorf("telegram: %s", result.Description)
	}

	return result.Result, nil
}
