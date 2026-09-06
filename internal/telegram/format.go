package telegram

import (
	"fmt"
	"strings"

	"github.com/JonasFranc1sco/vagouBot/internal/linkedin"
)

// FormatJobNotification formata uma vaga em HTML para o Telegram
func FormatJobNotification(job linkedin.Job) string {
	var sb strings.Builder

	// 💼 Título (negrito)
	sb.WriteString("<b>💼 ")
	sb.WriteString(escapeHTML(job.Title))
	sb.WriteString("</b>\n")

	// 🏢 Empresa
	sb.WriteString("🏢 <b>")
	sb.WriteString(escapeHTML(job.Company))
	sb.WriteString("</b>\n")

	// 📍 Localização + data
	sb.WriteString("📍 ")
	sb.WriteString(escapeHTML(job.Location))
	if job.PostedDate != "" {
		sb.WriteString(" • ")
		sb.WriteString(escapeHTML(job.PostedDate))
	}
	sb.WriteString("\n")

	// 📊 Senioridade + tipo (se disponível)
	if job.SeniorityLevel != "" || job.EmploymentType != "" {
		meta := []string{}
		if job.SeniorityLevel != "" {
			meta = append(meta, job.SeniorityLevel)
		}
		if job.EmploymentType != "" {
			meta = append(meta, job.EmploymentType)
		}
		sb.WriteString("📊 <i>")
		sb.WriteString(escapeHTML(strings.Join(meta, " | ")))
		sb.WriteString("</i>\n")
	}

	// 📝 Primeiros 400 chars da descrição
	if job.Description != "" {
		desc := job.Description
		if len(desc) > 400 {
			desc = desc[:400] + "..."
		}
		sb.WriteString("\n")
		sb.WriteString(escapeHTML(desc))
		sb.WriteString("\n")
	}

	// 🔗 Link da vaga
	sb.WriteString("\n🔗 <a href=\"")
	sb.WriteString(job.URL)
	sb.WriteString("\">Ver vaga no LinkedIn</a>\n")

	return sb.String()
}

// escapeHTML escopa caracteres especiais do HTML
func escapeHTML(s string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
	)
	return replacer.Replace(s)
}

// parseMode indica que usamos HTML no envio
const ParseModeHTML = "HTML"

// BuildJobsMessage monta uma mensagem com várias vagas.
func BuildJobsMessage(jobs []linkedin.Job, startIdx int) ([]string, error) {
	var messages []string
	var current strings.Builder

	for i, job := range jobs {
		msg := FormatJobNotification(job)

		if current.Len()+len(msg) > 4000 {
			messages = append(messages, current.String())
			current.Reset()
		}

		current.WriteString(msg)
		current.WriteString("\n----------------------------\n")

		_ = i
	}

	if current.Len() > 0 {
		messages = append(messages, current.String())
	}
	return messages, nil
}

// Formata para print no console (debug)
func ConsoleJob(job linkedin.Job) string {
	output := fmt.Sprintf("💼 %s\n🏢 %s\n📍 %s\n",
		job.Title, job.Company, job.Location)

	if job.SeniorityLevel != "" {
		output += fmt.Sprintf("📊 %s\n", job.SeniorityLevel)
	}

	if job.Description != "" {
		desc := job.Description
		if len(desc) > 300 {
			desc = desc[:300] + "..."
		}
		output += fmt.Sprintf("\n%s\n", desc)
	}

	output += fmt.Sprintf("\n🔗 %s\n", job.URL)
	return output

}
