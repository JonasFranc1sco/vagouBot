package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config é a estrutura principal de configuração.
// Cada campo mapeia para uma chave no YAML.
type Config struct {
	LinkedIn  LinkedInConfig  `yaml:"linkedin"`
	Proxy     ProxyConfig     `yaml:"proxy"`
	Telegram  TelegramConfig  `yaml:"telegram"`
	RateLimit RateLimitConfig `yaml:"rate_limit"`
	Filters   FiltersConfig   `yaml:"filters"`
}

type FiltersConfig struct {
	IncludeKeywords      []string `yaml:"include_keywords"`
	ExcludeKeywords      []string `yaml:"exclude_keywords"`
	IncludeCompanies     []string `yaml:"include_companies"`
	ExcludeCompanies     []string `yaml:"exclude_companies"`
	IncludeLocations     []string `yaml:"include_locations"`
	ExcludeLocations     []string `yaml:"exclude_locations"`
	MinDescriptionLength int      `yaml:"min_description_length"`
}

type LinkedInConfig struct {
	// Termos de busca (ex: "golang", "backend senior")
	Keywords []string `yaml:"keywords"`

	// Onde buscar as vagas
	Location string `yaml:"location"`

	// Limite de vagas por busca
	MaxResults int `yaml:"max_results"`
}

type ProxyConfig struct {
	// Ativa ou desativo o uso de proxy
	Enabled bool `yaml:"enabled"`

	// Lista de proxies para rotacionar
	URLs []string `yaml:"urls"`
}

type TelegramConfig struct {
	// Token do bot
	BotToken string `yaml:"bot_token"`

	// ID do chat/usuário para enviar mensagens
	ChatID int64 `yaml:"chat_id"`
}

type RateLimitConfig struct {
	// Controla  a velocidade do scraping
	RequestsPerSecond float64 `yaml:"requests_per_second"`

	// Número máximo de requests em rajada
	Burst int `yaml:"burst"`
}

// Load lê o arquivo YAML e retorna uma Config preenchida.
// Se o arquivo não existir, retorna a config padrão.
func Load(path string) (*Config, error) {
	cfg := defaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// defaultConfig retorna valores seguros para começar.
func defaultConfig() *Config {
	return &Config{
		LinkedIn: LinkedInConfig{
			Keywords:   []string{"golang"},
			Location:   "brazil",
			MaxResults: 100,
		},
		RateLimit: RateLimitConfig{
			RequestsPerSecond: 0.5,
			Burst:             1,
		},
	}
}
