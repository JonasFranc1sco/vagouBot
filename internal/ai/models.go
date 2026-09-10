package ai

// Resumes é a resposta do modelo: um currículo por vaga, keyed pelo "id".
type Resumes map[string]Resume

// Resume é a estrutura que o modelo deve devolver (schema do prompt).
// Os tags json são em PT porque é com esses nomes que a IA responde.
type Resume struct {
	Summary      string       `json:"resumo"`
	Skills       []string     `json:"habilidades"`
	Experience   []Experience `json:"experiencia,omitempty"`
	Projects     []Project    `json:"projetos,omitempty"`
	Education    []Education  `json:"formacao,omitempty"`
	Languages    []Language   `json:"idiomas,omitempty"`
	Observations string       `json:"observacoes,omitempty"`
}

type Experience struct {
	Role        string   `json:"cargo"`
	Company     string   `json:"empresa"`
	Period      string   `json:"periodo"`
	Description []string `json:"descricao"`
}

type Project struct {
	Name        string `json:"nome"`
	Link        string `json:"link"`
	Description string `json:"descricao"`
}

type Education struct {
	Course      string `json:"curso"`
	Institution string `json:"instituicao"`
	Period      string `json:"periodo"`
}

type Language struct {
	Language string `json:"idioma"`
	Level    string `json:"nivel"`
}
