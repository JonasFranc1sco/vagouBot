package telegram

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/JonasFranc1sco/vagouBot/internal/linkedin"
)

// jobURLRe casa a URL de uma vaga do LinkedIn dentro do texto da mensagem.
var jobURLRe = regexp.MustCompile(`https?://[^\s"<>]+linkedin\.com/jobs/view/[^\s"<>/?#]*\d+`)

// JobIDFromMessage extrai o ID da vaga de uma mensagem de notificação.
// Prioriza a entidade "text_link" (onde o Telegram guarda a URL quando a
// mensagem foi enviada com parse_mode=HTML); se não achar, tenta a entidade
// "url" recortando o texto e, por último, cai no fallback por regex no texto.
func JobIDFromMessage(msg *Message) string {
	if msg == nil {
		return ""
	}

	// 1) Entidade text_link: a URL vem completa e limpa no campo URL.
	for _, e := range msg.Entities {
		if e.Type == "text_link" && e.URL != "" {
			if id := JobIDFromURL(e.URL); id != "" {
				return id
			}
		}
	}

	// 2) Entidade url: a URL aparece solta no texto — recorta pelo offset.
	for _, e := range msg.Entities {
		if e.Type == "url" && e.Offset >= 0 && e.Length > 0 && e.Offset+e.Length <= len(msg.Text) {
			if id := JobIDFromURL(msg.Text[e.Offset : e.Offset+e.Length]); id != "" {
				return id
			}
		}
	}

	// 3) Fallback: procura a URL diretamente no texto.
	return ExtractJobID(msg.Text)
}

// JobIDFromURL extrai o ID numérico da vaga a partir de uma URL do LinkedIn.
// Aceita URLs numéricas (.../jobs/view/4371654421) e com slug
// (.../jobs/view/python-developer-at-iris-software-inc-4464493957).
func JobIDFromURL(rawURL string) string {
	if rawURL == "" || !jobURLRe.MatchString(rawURL) {
		return ""
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	// Último segmento do path (o ID fica sempre no final).
	path := strings.TrimRight(u.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return ""
	}
	last := parts[len(parts)-1]

	// Em URLs com slug, o ID é o último token separado por "-".
	segments := strings.Split(last, "-")
	id := segments[len(segments)-1]

	// Garante que o que sobrou é só o ID numérico.
	if id == "" {
		return ""
	}
	for _, r := range id {
		if r < '0' || r > '9' {
			return ""
		}
	}
	return id
}

// ExtractJobID extrai o ID numérico da vaga a partir do texto de uma
// notificação do Telegram (a mensagem que o usuário respondeu com /resume).
func ExtractJobID(text string) string {
	m := jobURLRe.FindString(text)
	if m == "" {
		return ""
	}
	return JobIDFromURL(m)
}

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
