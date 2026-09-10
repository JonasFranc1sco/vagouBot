package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/JonasFranc1sco/vagouBot/internal/ai"
	"github.com/JonasFranc1sco/vagouBot/internal/bot"
	"github.com/JonasFranc1sco/vagouBot/internal/config"
	"github.com/JonasFranc1sco/vagouBot/internal/filter"
	"github.com/JonasFranc1sco/vagouBot/internal/linkedin"
	"github.com/JonasFranc1sco/vagouBot/internal/telegram"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load("config.yaml")
	if err != nil {
		logger.Error("carregar config", "err", err)
		return
	}
	logger.Info("config carregada",
		"keywords", strings.Join(cfg.LinkedIn.Keywords, ", "),
		"location", cfg.LinkedIn.Location)

	// Clientes compartilhados entre o scraper (diário) e o bot (interativo).
	client := linkedin.NewClient(
		cfg.Proxy.URLs,
		cfg.Proxy.Enabled,
		cfg.RateLimit.RequestsPerSecond,
		cfg.RateLimit.Burst,
	)

	tgToken := env0r("TELEGRAM_BOT_TOKEN", cfg.Telegram.BotToken)
	tgChatID := envChatID(logger, cfg.Telegram.ChatID)
	tg := telegram.NewClient(tgToken, tgChatID)

	var aiClient *ai.Client
	if cfg.Ai.Enable {
		aiClient = ai.NewClient(
			cfg.Ai.BaseURL,
			env0r("AI_API_KEY", cfg.Ai.APIToken),
			cfg.Ai.Model,
			cfg.Ai.MaxTokens,
		)
	}

	// Modo interativo: escuta /resume enquanto o scraper dorme/coleta.
	if tgToken != "" && tgChatID != 0 {
		interactive := bot.New(tg, client, aiClient, cfg.Profile, logger)
		go interactive.Run(ctx)
		logger.Info("bot interativo ativo (comando /resume)")
	}

	for {
		if err := run(ctx, logger, cfg, client, tg, tgToken, tgChatID); err != nil {
			logger.Error("erro na execução", "err", err)
		}

		if err := waitUntil(ctx, logger, nextDailyRun(18, 0)); err != nil {
			logger.Info("encerrado durante a espera (kill switch)")
			return
		}
	}
}

func nextDailyRun(hour, minute int) time.Time {
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next
}

func waitUntil(ctx context.Context, logger *slog.Logger, t time.Time) error {
	logger.Info("aguardando próxima execução",
		"proximo", t.Format(time.RFC3339),
		"daqui_a", time.Until(t).Round(time.Minute),
	)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(time.Until(t)):
		return nil
	}
}

func run(ctx context.Context, logger *slog.Logger, cfg *config.Config, client *linkedin.Client, tg *telegram.Client, tgToken string, tgChatID int64) error {
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "."
	}
	dedup := filter.NewDedup(filepath.Join(dataDir, "seen.json"))

	filterCfg := filter.Config{
		IncludeKeywords:      cfg.Filters.IncludeKeywords,
		ExcludeKeywords:      cfg.Filters.ExcludeKeywords,
		IncludeCompanies:     cfg.Filters.IncludeCompanies,
		ExcludeCompanies:     cfg.Filters.ExcludeCompanies,
		IncludeLocations:     cfg.Filters.IncludeLocations,
		ExcludeLocations:     cfg.Filters.ExcludeLocations,
		MinDescriptionLength: cfg.Filters.MinDescriptionLength,
		MaxJobAgeDays:        cfg.Filters.MaxJobAgeDays,
	}

	var candidates []linkedin.Job

	for _, kw := range cfg.LinkedIn.Keywords {
		if ctx.Err() != nil {
			logger.Info("kill switch ativado, interrompendo busca")
			break
		}
		jobs, err := searchKeyword(ctx, client, kw, cfg.LinkedIn.Location, cfg.LinkedIn.MaxResults, fTPR(cfg.LinkedIn.PostTime))
		if err != nil {
			logger.Error("busca falhou", "keyowrd", kw, "encontradas", len(jobs))
			continue
		}
		logger.Info("busca concluída", "keyword", kw, "encontradas", len(jobs))
		candidates = append(candidates, jobs...)
	}

	candidates = dedup.FilterNew(candidates)
	if len(candidates) > cfg.LinkedIn.MaxResults {
		candidates = candidates[:cfg.LinkedIn.MaxResults]
	}
	logger.Info("após dedup", "novas", len(candidates))

	for i := range candidates {
		select {
		case <-ctx.Done():
			logger.Info("kill switch ativado, encerrando")
			return nil
		default:
		}
		if candidates[i].ID == "" {
			continue
		}
		if err := client.FetchDetail(ctx, &candidates[i]); err != nil {
			logger.Error("falha no detalhe", "id", candidates[i].ID, "err", err)
			continue
		}
	}

	filtered := filter.FilterProcess(candidates, filterCfg)
	logger.Info("vagas relevantes", "de", len(candidates), "relevantes", len(filtered))

	for i, job := range filtered {
		if i >= 3 {
			break
		}
		fmt.Println(telegram.ConsoleJob(job))
		fmt.Println("---")
	}

	// Envio das notificações para o Telegram (currículos ficam sob demanda,
	// via /resume, para economizar tokens da IA).
	if tgToken != "" && tgChatID != 0 {
		if len(filtered) == 0 {
			if err := tg.SendText(ctx, "Nenhuma vaga nova hoje."); err != nil {
				return fmt.Errorf("telegram: %w", err)
			}
		} else {
			if err := tg.SendText(ctx, fmt.Sprintf("Encontradas %d vagas novas:", len(filtered))); err != nil {
				return fmt.Errorf("telegram: %w", err)
			}
			for _, job := range filtered {
				msg := telegram.FormatJobNotification(job)
				if err := tg.SendHTML(ctx, msg); err != nil {
					return fmt.Errorf("telegram: %w", err)
				}
				dedup.MarkSeen(job.ID)
				time.Sleep(1 * time.Second)
			}
		}
	}
	return nil
}

func searchKeyword(ctx context.Context, client *linkedin.Client, keywords, location string, maxResults int, fTPR string) ([]linkedin.Job, error) {
	var allJobs []linkedin.Job

	for start := 0; start < maxResults; start += 25 {
		select {
		case <-ctx.Done():
			return allJobs, nil
		default:
		}

		searchURL := fmt.Sprintf("https://www.linkedin.com/jobs-guest/jobs/api/seeMoreJobPostings/search?keywords=%s&location=%s&start=%d", url.QueryEscape(keywords), url.QueryEscape(location), start)
		if fTPR != "" {
			searchURL += "&f_TPR=" + fTPR
		}
		resp, err := client.DoRequest(ctx, searchURL)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		jobs, err := linkedin.ParseJobs(strings.NewReader(string(body)))
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

func env0r(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envChatID(logger *slog.Logger, fallback int64) int64 {
	v := os.Getenv("TELEGRAM_CHAT_ID")
	if v == "" {
		return fallback
	}
	id, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		logger.Warn("TELEGRAM_CHAT_ID inválido", "value", v)
		return fallback
	}
	return id
}

// fTPR converte a configuração post_time no parâmetro f_TPR do LinkedIn.
// "" = qualquer data; "day" = últimas 24h; "week" = última semana; "month" = último mês.
func fTPR(postTime string) string {
	switch strings.ToLower(strings.TrimSpace(postTime)) {
	case "day":
		return "r86400"
	case "week":
		return "r604800"
	case "month":
		return "r2592000"
	default:
		return ""
	}
}
