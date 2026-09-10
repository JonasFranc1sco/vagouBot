package bot

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/JonasFranc1sco/vagouBot/internal/ai"
	"github.com/JonasFranc1sco/vagouBot/internal/config"
	"github.com/JonasFranc1sco/vagouBot/internal/linkedin"
	"github.com/JonasFranc1sco/vagouBot/internal/pdf"
	"github.com/JonasFranc1sco/vagouBot/internal/telegram"
)

const (
	// pollTimeout é o timeout do long-polling aceito pelo Telegram (máx. 50s).
	pollTimeout = 50
	// pollInterval é o intervalo entre polls em caso de erro.
	pollInterval = 5 * time.Second
)

// Bot gerencia o modo interativo: escuta novos updates do Telegram e
// responde ao comando /resume gerando o currículo da vaga respondida.
type Bot struct {
	tg       *telegram.Client
	linkedin *linkedin.Client
	ai       *ai.Client
	profile  config.ProfileConfig
	logger   *slog.Logger
	// offset controla quais updates já foram processados.
	offset int
}

// New cria um Bot. aiClient pode ser nil (IA desligada) — nesse caso o
// comando /resume responde avisando que não está configurado.
func New(tg *telegram.Client, client *linkedin.Client, aiClient *ai.Client, profile config.ProfileConfig, logger *slog.Logger) *Bot {
	return &Bot{
		tg:       tg,
		linkedin: client,
		ai:       aiClient,
		profile:  profile,
		logger:   logger,
	}
}

// Run mantém o long-polling ativo até o contexto ser cancelado.
func (b *Bot) Run(ctx context.Context) {
	for {
		updates, err := b.tg.GetUpdates(ctx, b.offset, pollTimeout)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			b.logger.Error("erro no long-polling do telegram", "err", err)
			time.Sleep(pollInterval)
			continue
		}

		for _, u := range updates {
			// Pula updates antigos (getUpdates pode repetir os últimos).
			if u.UpdateID < b.offset {
				continue
			}
			b.offset = u.UpdateID + 1
			b.handleUpdate(ctx, u)
		}

		if ctx.Err() != nil {
			return
		}
	}
}

func (b *Bot) handleUpdate(ctx context.Context, u telegram.Update) {
	if u.Message == nil || u.Message.Chat == nil {
		return
	}

	// Só responde no chat configurado (não vira bot público).
	if u.Message.Chat.ID != b.tg.ChatID() {
		return
	}

	if !isResumeCommand(u.Message.Text) {
		return
	}

	b.handleResume(ctx, u.Message)
}

func (b *Bot) handleResume(ctx context.Context, msg *telegram.Message) {
	if msg.ReplyToMessage == nil {
		b.reply(ctx, msg, "Use /resume como resposta a uma vaga para gerar o currículo dela.")
		return
	}

	jobID := telegram.JobIDFromMessage(msg.ReplyToMessage)
	if jobID == "" {
		b.logger.Warn("reply sem vaga detectada",
			"reply_text", msg.ReplyToMessage.Text,
			"reply_entities", msg.ReplyToMessage.Entities,
		)
		b.reply(ctx, msg, "Não encontrei a vaga na mensagem respondida. Responda a uma notificação de vaga.")
		return
	}

	if b.ai == nil {
		b.reply(ctx, msg, "A IA não está configurada. Ative `ai.enable` no config.yaml e defina a AI_API_KEY.")
		return
	}

	if err := b.tg.SendChatAction(ctx, "upload_document"); err != nil {
		b.logger.Warn("falha ao enviar chat action", "err", err)
	}
	b.reply(ctx, msg, fmt.Sprintf("Gerando currículo para a vaga... (%s)", jobID))

	job := linkedin.Job{ID: jobID}
	if err := b.linkedin.FetchDetail(ctx, &job); err != nil {
		b.logger.Error("falha ao buscar detalhes da vaga", "id", jobID, "err", err)
		b.reply(ctx, msg, "Não consegui buscar os detalhes da vaga. Tente novamente em alguns segundos.")
		return
	}

	resumes, err := b.ai.GenerateResumes(ctx, []linkedin.Job{job}, b.profile)
	if err != nil {
		b.logger.Error("falha ao gerar currículo", "id", jobID, "err", err)
		b.reply(ctx, msg, "Erro ao gerar o currículo. Tente novamente em alguns instantes.")
		return
	}
	resume, ok := resumes[job.ID]
	if !ok {
		b.logger.Error("IA não retornou currículo", "id", jobID)
		b.reply(ctx, msg, "A IA não retornou um currículo para essa vaga.")
		return
	}

	content, err := pdf.RenderResume(b.profile, resume)
	if err != nil {
		b.logger.Error("falha ao gerar PDF", "id", jobID, "err", err)
		b.reply(ctx, msg, "Erro ao gerar o PDF do currículo.")
		return
	}

	fileName := fmt.Sprintf("curriculo_%s.pdf", jobID)
	caption := "Curriculo adaptado para: " + job.Title
	if err := b.tg.SendDocument(ctx, fileName, content, caption); err != nil {
		b.logger.Error("falha ao enviar PDF", "id", jobID, "err", err)
		b.reply(ctx, msg, "Erro ao enviar o PDF do currículo.")
	}
}

// reply responde a mensagem do usuário no mesmo thread, logando o erro.
func (b *Bot) reply(ctx context.Context, msg *telegram.Message, text string) {
	if err := b.tg.SendReply(ctx, msg.MessageID, text); err != nil {
		b.logger.Error("falha ao responder", "err", err)
	}
}

// isResumeCommand detecta /resume no início da mensagem (aceita título/args).
func isResumeCommand(text string) bool {
	cmd := strings.ToLower(strings.TrimSpace(text))
	return cmd == "/resume" || strings.HasPrefix(cmd, "/resume ")
}
