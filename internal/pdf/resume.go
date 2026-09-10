package pdf

import (
	"fmt"
	"strings"

	"github.com/JonasFranc1sco/vagouBot/internal/ai"
	appconfig "github.com/JonasFranc1sco/vagouBot/internal/config"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/line"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	mconfig "github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontfamily"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/consts/pagesize"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

var (
	accentColor = props.Color{Red: 37, Green: 97, Blue: 174}   // azul "LinkedIn"
	grayColor   = props.Color{Red: 100, Green: 110, Blue: 120} // cinza de metadados
	linkColor   = props.Color{Red: 27, Green: 99, Blue: 228}   // azul de link
)

// RenderResume monta o PDF de um currículo adaptado a partir do perfil do
// candidato (header) e da resposta da IA (conteúdo). Retorna os bytes do PDF.
func RenderResume(profile appconfig.ProfileConfig, r ai.Resume) ([]byte, error) {
	cfg := mconfig.NewBuilder().
		WithPageSize(pagesize.A4).
		WithLeftMargin(10).
		WithRightMargin(10).
		WithTopMargin(12).
		WithBottomMargin(10).
		WithDefaultFont(&props.Font{
			Family: fontfamily.Arial,
			Size:   10,
			Style:  fontstyle.Normal,
			Color:  &props.BlackColor,
		}).
		Build()

	m := maroto.New(cfg)

	addProfileHeader(m, profile)
	addSummarySection(m, r.Summary)
	addSkillsSection(m, r.Skills)
	addExperienceSection(m, r.Experience)
	addProjectsSection(m, r.Projects)
	addEducationSection(m, r.Education)
	addLanguagesSection(m, r.Languages)

	doc, err := m.Generate()
	if err != nil {
		return nil, fmt.Errorf("gerar PDF: %w", err)
	}
	return doc.GetBytes(), nil
}

// addProfileHeader adiciona nome, cargo atual e contatos no topo do PDF.
func addProfileHeader(m core.Maroto, profile appconfig.ProfileConfig) {
	m.AddAutoRow(text.NewCol(12, profile.Name, props.Text{
		Style: fontstyle.Bold,
		Size:  20,
	}))

	m.AddAutoRow(text.NewCol(12, profile.JobTitle, props.Text{
		Style: fontstyle.Italic,
		Size:  12,
		Color: &grayColor,
		Top:   4,
	}))

	if contact := formatContact(profile.Contact); contact != "" {
		m.AddAutoRow(text.NewCol(12, contact, props.Text{
			Size:  9,
			Color: &grayColor,
			Top:   4,
		}))
	}

	// linha separadora do cabeçalho
	m.AddAutoRow(line.NewCol(12, props.Line{
		Color:         &accentColor,
		Thickness:     1,
		OffsetPercent: 0,
		SizePercent:   100,
	}))
}

// addSectionTitle adiciona o título destacado de uma seção.
func addSectionTitle(m core.Maroto, title string) {
	m.AddAutoRow(text.NewCol(12, title, props.Text{
		Style: fontstyle.Bold,
		Size:  12,
		Color: &accentColor,
		Top:   8,
	}))
	m.AddAutoRow(line.NewCol(12, props.Line{
		Color:         &accentColor,
		Thickness:     0.5,
		OffsetPercent: 0,
		SizePercent:   100,
	}))
}

// addSummarySection adiciona o resumo em forma de parágrafo.
func addSummarySection(m core.Maroto, summary string) {
	if summary == "" {
		return
	}
	addSectionTitle(m, "RESUMO")
	m.AddAutoRow(text.NewCol(12, summary, props.Text{
		Top:    2,
		Bottom: 4,
		Align:  align.Justify,
	}))
}

// addSkillsSection adiciona as habilidades em linhas com marcador.
func addSkillsSection(m core.Maroto, skills []string) {
	if len(skills) == 0 {
		return
	}
	addSectionTitle(m, "HABILIDADES")
	for _, skill := range skills {
		m.AddAutoRow(text.NewCol(12, "• "+skill, props.Text{Left: 6, Top: 1}))
	}
}

// addExperienceSection adiciona cada cargo com período e descrição.
func addExperienceSection(m core.Maroto, items []ai.Experience) {
	if len(items) == 0 {
		return
	}
	addSectionTitle(m, "EXPERIÊNCIA")
	for _, item := range items {
		m.AddAutoRow(text.NewCol(12, roleTitle(item.Role, item.Company), props.Text{
			Style: fontstyle.Bold,
			Top:   4,
		}))
		if item.Period != "" {
			m.AddAutoRow(text.NewCol(12, item.Period, props.Text{
				Style: fontstyle.Italic,
				Size:  9,
				Color: &grayColor,
				Left:  6,
			}))
		}
		for _, desc := range item.Description {
			m.AddAutoRow(text.NewCol(12, "• "+desc, props.Text{Left: 6, Top: 1}))
		}
	}
}

// addProjectsSection adiciona projetos com link e descrição.
func addProjectsSection(m core.Maroto, items []ai.Project) {
	if len(items) == 0 {
		return
	}
	addSectionTitle(m, "PROJETOS")
	for _, item := range items {
		m.AddAutoRow(text.NewCol(12, item.Name, props.Text{
			Style: fontstyle.Bold,
			Top:   4,
		}))
		if item.Link != "" {
			link := item.Link
			m.AddAutoRow(text.NewCol(12, link, props.Text{
				Size:      9,
				Color:     &linkColor,
				Hyperlink: &link,
			}))
		}
		if item.Description != "" {
			m.AddAutoRow(text.NewCol(12, item.Description, props.Text{Left: 6, Top: 1}))
		}
	}
}

// addEducationSection adiciona formação acadêmica.
func addEducationSection(m core.Maroto, items []ai.Education) {
	if len(items) == 0 {
		return
	}
	addSectionTitle(m, "FORMAÇÃO")
	for _, item := range items {
		m.AddAutoRow(text.NewCol(12, educationTitle(item), props.Text{
			Style: fontstyle.Bold,
			Top:   4,
		}))
		if item.Period != "" {
			m.AddAutoRow(text.NewCol(12, item.Period, props.Text{
				Size:  9,
				Style: fontstyle.Italic,
				Color: &grayColor,
				Left:  6,
			}))
		}
	}
}

// addLanguagesSection adiciona os idiomas em uma linha separada por "|".
func addLanguagesSection(m core.Maroto, items []ai.Language) {
	if len(items) == 0 {
		return
	}
	addSectionTitle(m, "IDIOMAS")

	var sb strings.Builder
	for i, item := range items {
		if i > 0 {
			sb.WriteString("  |  ")
		}
		sb.WriteString(item.Language)
		if item.Level != "" {
			sb.WriteString(" (" + item.Level + ")")
		}
	}
	m.AddAutoRow(text.NewCol(12, sb.String(), props.Text{Top: 2}))
}

// addNotesSection adiciona observações do candidato (ex: gaps, diferenciais).
func addNotesSection(m core.Maroto, notes string) {
	addSectionTitle(m, "OBSERVAÇÕES")
	m.AddAutoRow(text.NewCol(12, notes, props.Text{
		Style: fontstyle.Italic,
		Size:  9,
		Color: &grayColor,
		Top:   2,
	}))
}

// roleTitle concatena cargo e empresa em um título.
func roleTitle(role, company string) string {
	if company == "" {
		return role
	}
	return role + " - " + company
}

// educationTitle junta curso e instituição.
func educationTitle(item ai.Education) string {
	if item.Institution == "" {
		return item.Course
	}
	return item.Course + " - " + item.Institution
}

// formatContact junta email, telefone e redes em uma linha.
func formatContact(c appconfig.ContactConfig) string {
	parts := make([]string, 0, 4)
	for _, part := range []string{c.Email, c.Phone, c.LinkedIn, c.GitHub} {
		if part != "" {
			parts = append(parts, part)
		}
	}
	return strings.Join(parts, "  •  ")
}
