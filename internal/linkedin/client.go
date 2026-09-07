package linkedin

import (
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"

	"github.com/JonasFranc1sco/vagouBot/internal/proxy"
	ratelimit "github.com/JonasFranc1sco/vagouBot/internal/rate"
	"github.com/hashicorp/go-retryablehttp"
)

// Client encapsula o HTTP client com configuração específica
// para scraping no LinkedIn.
type Client struct {
	HTTP    *http.Client
	Limiter *ratelimit.Limiter
}

// NewClient cria um HTTP client com:
// Retry automático com backoff
// Cookie jar para manter cookies entre requests
// Headers que simulam um browser real
func NewClient(proxyURLs []string, proxyEnabled bool, rps float64, burst int) *Client {
	// Cookie jar armazena cookies que o LinkedIn envia.
	jar, _ := cookiejar.New(nil)

	// retryablehttp é um wrapper do http.Client padrão que adiciona:
	// Retry automático, exponential backoff, jitter.
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 3
	retryClient.RetryWaitMin = 2 * time.Second

	// StandardClient() retorna um *http.Client padrão que
	// usa o retry por baixo dos panos. Para passar
	// o cookie jar e outros settings normalmente
	sc := retryClient.StandardClient()
	sc.Jar = jar
	sc.Timeout = 15 * time.Second

	// Transport configura timeouts granulares do TCP/TLS.
	transport := &http.Transport{
		MaxIdleConns:          10,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
	}

	if proxyEnabled && len(proxyURLs) > 0 {
		rotator := proxy.New(proxyURLs)
		rotator.ConfigureTransport(transport)
	}

	sc.Transport = transport

	limiter := ratelimit.New(rps, burst)

	return &Client{
		HTTP:    sc,
		Limiter: limiter,
	}
}

// linkedHeaders retorna headers que um browser real enviaria.
func linkedinHeaders() map[string]string {
	return map[string]string{
		// Identifica o "browser".
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
		// Aceita o HTML que o LinkedIn vai retornar
		"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8",
		"Accept-Language": "en-US,en;q=0.9",
		"Accept-Encoding": "gzip, deflate, br",
		// LinkedIn verifica se viemos da página dele.
		"Referer":                   "https://www.linkedin.com/",
		"Sec-Fetch-Dest":            "document",
		"Sec-Fetch-Mode":            "navigate",
		"Sec-Fetch-Site":            "same-origin",
		"Sec-Fetch-User":            "?1",
		"Upgrade-Insecure-Requests": "1",
		"Cache-Control":             "max-age=0",
	}
}

// DoRequest executa um GET com os headers do LinkedIn
func (c *Client) DoRequest(ctx context.Context, url string) (*http.Response, error) {
	if err := c.Limiter.Wait(ctx); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	// Aplica todos os headers de browser
	for key, value := range linkedinHeaders() {
		req.Header.Set(key, value)
	}

	// retryablehttp retry
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}

	// Descomprime gzip se necessário
	if strings.Contains(resp.Header.Get("Content-Encoding"), "gzip") {
		rawBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		gz, err := gzip.NewReader(strings.NewReader(string(rawBody)))
		if err != nil {
			return nil, err
		}

		resp.Body = gz
	}

	return resp, nil
}
