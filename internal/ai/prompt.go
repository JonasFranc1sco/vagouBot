package ai

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/JonasFranc1sco/vagouBot/internal/config"
	"github.com/JonasFranc1sco/vagouBot/internal/linkedin"
)

// maxDescriptionChars limita o tamanho da descrição de cada vaga enviada ao modelo
const maxDescriptionChars = 1500

// Messages agrupa o par de mensagens da chamada (OpenAI-compatible)
type Messages struct {
	System string
	User   string
}

// BuildMessages recebe o lote de vagas
func BuildMessages(jobs []linkedin.Job, profile config.ProfileConfig) (Messages, error) {
	profileJSON, err := json.Marshal(profile)
	if err != nil {
		return Messages{}, fmt.Errorf("serializar perfil: %w", err)
	}

	jobsJSON, err := jobsToJSON(jobs)
	if err != nil {
		return Messages{}, err
	}

	return Messages{
		System: systemPrompt(),
		User:   userPrompt(string(profileJSON), string(jobsJSON)),
	}, nil
}

// jobsToJSON projeta só os campos relevantes da vaga para o prompt.
func jobsToJSON(jobs []linkedin.Job) (string, error) {
	type jobInput struct {
		ID          string `json:"id"`
		Title       string `json:"title"`
		Seniority   string `json:"seniority_level"`
		Description string `json:"description"`
	}

	inputs := make([]jobInput, 0, len(jobs))
	for _, j := range jobs {
		inputs = append(inputs, jobInput{
			ID:          j.ID,
			Title:       j.Title,
			Seniority:   j.SeniorityLevel,
			Description: truncate(j.Description, maxDescriptionChars),
		})
	}

	b, err := json.Marshal(inputs)
	if err != nil {
		return "", fmt.Errorf("serializar vagas: %w", err)
	}
	return string(b), nil
}

// truncate corta a string no limite, sem qubrar no meio de um caractere acentuado
func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "..."
}

// systemPrompt fixa as regras de comportamento.
func systemPrompt() string {
	return `Você é um assistente especialista em currículos. Sua tarefa é adaptar o
perfil profissional do usuário para cada vaga de emprego informada.

REGRAS OBRIGATORIAS:
1. Use EXCLUSIVAMENTE os dados do perfil informado. NUNCA invente cargo,
   empresa, habilidade, idioma, formacao ou projeto.
2. Adapte o conteudo a vaga: reordene secoes, reescreva o resumo e as
   descricoes com foco nos requisitos da vaga, sem distorcer fatos.
3. Se um requisito da vaga nao existir no perfil, mencione isso no campo
   "observacoes"; nao o adicione como se fosse real.
4. Pode omitir secoes irrelevantes a vaga, mas nunca substitui-las por
   conteudo inventado.
5. Escreva em portugues do Brasil, salvo se a vaga exigir outro idioma.
6. Responda APENAS com um JSON valido, sem markdown, sem texto, sem comentarios.

FORMATO DE RESPOSTA:
Um unico objeto JSON. Cada chave e o campo "id" de uma vaga e o valor e o
curriculo correspondente, com a estrutura:
{ "resumo":      string (max ~600 chars),
  "habilidades": [string] (max 15 itens),
  "experiencia": [ { "cargo": string, "empresa": string,
                     "periodo": string, "descricao": [string] (max 3 itens) } ],
  "projetos":    [ { "nome": string, "link": string, "descricao": string } ],
  "formacao":    [ { "curso": string, "instituicao": string, "periodo": string } ],
  "idiomas":     [ { "idioma": string, "nivel": string } ],
  "observacoes": string }`
}

// userPrompt monta o conteúdo dinâmico
func userPrompt(profileJSON, jobsJSON string) string {
	var sb strings.Builder
	sb.WriteString("PERFIL DO USUARIO (unica fonte de verdade, em JSON):\n")
	sb.WriteString(profileJSON)
	sb.WriteString("\n\nVAGAS (gere um curriculo para cada uma, usando o \"id\" como chave):\n")
	sb.WriteString(jobsJSON)
	return sb.String()
}
