package linkedin

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/JonasFranc1sco/vagouBot/internal/proxy"
	ratelimit "github.com/JonasFranc1sco/vagouBot/internal/rate"
	"github.com/hashicorp/go-retryablehttp"
)

const defaultCSRFToken = "ajax:5829410394810293841"

// Client encapsula o HTTP client com configuração específica
// para scraping no LinkedIn (Guest API e Voyager API autenticada).
type Client struct {
	HTTP      *http.Client
	Limiter   *ratelimit.Limiter
	hasAuth   bool
	csrfToken string
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

// SetSession configura o cookie de sessão li_at e o JSESSIONID (csrf-token)
// no CookieJar do cliente para habilitar buscas autenticadas e vagas recomendadas.
func (c *Client) SetSession(liAt, csrfToken string) {
	liAt = strings.Trim(strings.TrimSpace(liAt), "\"")
	if liAt == "" {
		return
	}

	csrfToken = strings.Trim(strings.TrimSpace(csrfToken), "\"")
	if csrfToken == "" {
		csrfToken = defaultCSRFToken
	}

	c.hasAuth = true
	c.csrfToken = csrfToken

	if c.HTTP.Jar == nil {
		jar, _ := cookiejar.New(nil)
		c.HTTP.Jar = jar
	}

	u, err := url.Parse("https://www.linkedin.com")
	if err != nil {
		return
	}

	c.HTTP.Jar.SetCookies(u, []*http.Cookie{
		{
			Name:   "li_at",
			Value:  liAt,
			Domain: ".linkedin.com",
			Path:   "/",
			Secure: true,
		},
		{
			Name:   "JSESSIONID",
			Value:  fmt.Sprintf("%q", csrfToken),
			Domain: ".linkedin.com",
			Path:   "/",
			Secure: true,
		},
	})
}

// HasSession informa se o cliente possui o cookie li_at configurado.
func (c *Client) HasSession() bool {
	return c.hasAuth
}

// currentCSRFToken retorna o JSESSIONID atual do CookieJar (sem aspas) ou o token padrão.
func (c *Client) currentCSRFToken() string {
	if c.HTTP != nil && c.HTTP.Jar != nil {
		if u, err := url.Parse("https://www.linkedin.com"); err == nil {
			for _, ck := range c.HTTP.Jar.Cookies(u) {
				if ck.Name == "JSESSIONID" && ck.Value != "" {
					return strings.Trim(ck.Value, "\"")
				}
			}
		}
	}
	if c.csrfToken != "" {
		return c.csrfToken
	}
	return defaultCSRFToken
}

// linkedinHeaders retorna headers que um browser real enviaria.
func linkedinHeaders() map[string]string {
	return map[string]string{
		// Identifica o "browser".
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
		// Aceita o HTML que o LinkedIn vai retornar
		"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8",
		"Accept-Language": "pt-BR,pt;q=0.9,en-US;q=0.8,en;q=0.7",
		"Accept-Encoding": "gzip, deflate",
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
func (c *Client) DoRequest(ctx context.Context, reqURL string) (*http.Response, error) {
	return c.doRequestWithHeaders(ctx, reqURL, linkedinHeaders())
}

// DoVoyagerRequest executa um GET autenticado na API interna Voyager do LinkedIn.
func (c *Client) DoVoyagerRequest(ctx context.Context, reqURL string) (*http.Response, error) {
	headers := linkedinHeaders()
	headers["Accept"] = "application/vnd.linkedin.normalized+json+2.1"
	headers["csrf-token"] = c.currentCSRFToken()
	headers["x-restli-protocol-version"] = "2.0.0"
	headers["x-li-lang"] = "pt_BR"
	headers["Referer"] = "https://www.linkedin.com/jobs/"
	headers["Sec-Fetch-Dest"] = "empty"
	headers["Sec-Fetch-Mode"] = "cors"

	return c.doRequestWithHeaders(ctx, reqURL, headers)
}

func (c *Client) doRequestWithHeaders(ctx context.Context, reqURL string, headers map[string]string) (*http.Response, error) {
	if err := c.Limiter.Wait(ctx); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

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
