package main

import (
	"context"
	"fmt"
	"log/slog"
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

	// Clientes compartilhados entre o scraper (diário) e o bot (interativo).
	client := linkedin.NewClient(
		cfg.Proxy.URLs,
		cfg.Proxy.Enabled,
		cfg.RateLimit.RequestsPerSecond,
		cfg.RateLimit.Burst,
	)

	liAt := env0r("LINKEDIN_LI_AT", cfg.LinkedIn.SessionCookie)
	csrfToken := env0r("LINKEDIN_JSESSIONID", cfg.LinkedIn.CSRFToken)
	if liAt != "" {
		client.SetSession(liAt, csrfToken)
	}

	logger.Info("config carregada",
		"keywords", strings.Join(cfg.LinkedIn.Keywords, ", "),
		"location", cfg.LinkedIn.Location,
		"geo_id", cfg.LinkedIn.GeoID,
		"workplace_type", strings.Join(cfg.LinkedIn.WorkplaceType, ","),
		"experience_level", strings.Join(cfg.LinkedIn.ExperienceLevel, ","),
		"authenticated", client.HasSession(),
		"fetch_recommended", cfg.LinkedIn.FetchRecommended,
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

	// 1. Vagas recomendadas diretamente para o perfil do usuário (se autenticado com li_at)
	if cfg.LinkedIn.FetchRecommended && client.HasSession() {
		recJobs, err := client.FetchRecommendedJobs(ctx, cfg.LinkedIn.MaxResults)
		if err != nil {
			logger.Warn("falha ao buscar vagas recomendadas do perfil", "err", err)
		} else {
			logger.Info("vagas recomendadas do perfil coletadas", "encontradas", len(recJobs))
			candidates = append(candidates, recJobs...)
		}
	}

	// 2. Busca por queries estruturadas (Voyager autenticado com fallback Guest, ou Guest direto)
	for _, kw := range cfg.LinkedIn.Keywords {
		if ctx.Err() != nil {
			logger.Info("kill switch ativado, interrompendo busca")
			break
		}
		params := linkedin.SearchParams{
			Keywords:         kw,
			Location:         cfg.LinkedIn.Location,
			GeoID:            cfg.LinkedIn.GeoID,
			WorkplaceTypes:   cfg.LinkedIn.WorkplaceType,
			ExperienceLevels: cfg.LinkedIn.ExperienceLevel,
			PostTime:         cfg.LinkedIn.PostTime,
			SortBy:           cfg.LinkedIn.SortBy,
			MaxResults:       cfg.LinkedIn.MaxResults,
		}
		jobs, err := client.SearchJobs(ctx, params)
		if err != nil {
			logger.Error("busca falhou", "keyword", kw, "encontradas", len(jobs), "err", err)
			continue
		}
		logger.Info("busca concluída", "keyword", kw, "encontradas", len(jobs))
		candidates = append(candidates, jobs...)
	}

	// Remove vagas já vistas em execuções anteriores ou duplicadas no mesmo lote
	candidates = dedup.FilterNew(candidates)
	logger.Info("após dedup", "novas", len(candidates))

	// Pré-filtro rápido pelos metadados do card (título, empresa, localização, data)
	// antes de gastar requisições buscando detalhes de vagas que já seriam descartadas.
	preFiltered := filter.PreFilterProcess(candidates, filterCfg)
	markRejectedAsSeen(dedup, candidates, preFiltered)

	if cfg.LinkedIn.MaxResults > 0 && len(preFiltered) > cfg.LinkedIn.MaxResults {
		preFiltered = preFiltered[:cfg.LinkedIn.MaxResults]
	}
	logger.Info("após pré-filtro", "candidatas", len(preFiltered))

	for i := range preFiltered {
		select {
		case <-ctx.Done():
			logger.Info("kill switch ativado, encerrando")
			return nil
		default:
		}
		if preFiltered[i].ID == "" {
			continue
		}
		if err := client.FetchDetail(ctx, &preFiltered[i]); err != nil {
			logger.Error("falha no detalhe", "id", preFiltered[i].ID, "err", err)
			continue
		}
	}

	filtered := filter.FilterProcess(preFiltered, filterCfg)
	markRejectedAsSeen(dedup, preFiltered, filtered)
	logger.Info("vagas relevantes", "de", len(preFiltered), "relevantes", len(filtered))

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
				dedup.MarkSeenJob(job)
				time.Sleep(1 * time.Second)
			}
		}
	} else {
		// Modo console: também marca as vagas processadas como vistas
		dedup.MarkAll(filtered)
	}
	return nil
}

// markRejectedAsSeen marca no seen.json as vagas que foram descartadas pelos filtros,
// evitando que o scraper refaça FetchDetail das mesmas vagas descartadas todos os dias.
func markRejectedAsSeen(dedup *filter.DedupManager, before, after []linkedin.Job) {
	kept := make(map[string]bool, len(after))
	for _, j := range after {
		kept[j.ID] = true
	}
	for _, j := range before {
		if !kept[j.ID] {
			dedup.MarkSeenJob(j)
		}
	}
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
