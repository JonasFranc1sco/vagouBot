package telegram

import "testing"

func TestJobIDFromMessage(t *testing.T) {
	const jobURL = "https://www.linkedin.com/jobs/view/3871234567/?alternateChannel=search&position=1&pageNum=0"

	tests := []struct {
		name string
		msg  *Message
		want string
	}{
		{
			name: "entidade text_link com URL + query",
			msg: &Message{
				Text: "💼 <b>Vaga</b>\n🔗 Ver vaga no LinkedIn",
				Entities: []MessageEntity{
					{Type: "bold", Offset: 0, Length: 1},
					{Type: "text_link", Offset: 5, Length: 19, URL: jobURL},
				},
			},
			want: "3871234567",
		},
		{
			name: "entidade text_link com URL simples",
			msg: &Message{
				Text: "🔗 Ver vaga no LinkedIn",
				Entities: []MessageEntity{
					{Type: "text_link", Offset: 0, Length: 19, URL: "https://www.linkedin.com/jobs/view/123456"},
				},
			},
			want: "123456",
		},
		{
			name: "entidade text_link com URL slug (formato real da busca)",
			msg: &Message{
				Text: "🔗 Ver vaga no LinkedIn",
				Entities: []MessageEntity{
					{Type: "text_link", Offset: 0, Length: 19, URL: "https://cf.linkedin.com/jobs/view/junior-calculator-at-terberg-totaal-installaties-4371654421?position=1&pageNum=0&refId=SD1szjv8spYHEDXDfJ0qwA%3D%3D&trackingId=kjfYLZ%2FxXCQgcflZt91zTQ%3D%3D"},
				},
			},
			want: "4371654421",
		},
		{
			name: "entidade text_link com URL slug sem query",
			msg: &Message{
				Text: "🔗 Ver vaga no LinkedIn",
				Entities: []MessageEntity{
					{Type: "text_link", Offset: 0, Length: 19, URL: "https://ca.linkedin.com/jobs/view/python-developer-at-iris-software-inc-4464493957"},
				},
			},
			want: "4464493957",
		},
		{
			name: "entidade url recorta do texto",
			msg: &Message{
				Text: "Vaga: https://br.linkedin.com/jobs/view/987654 e tal",
				Entities: []MessageEntity{
					{Type: "url", Offset: 6, Length: 40},
				},
			},
			want: "987654",
		},
		{
			name: "fallback por regex no texto",
			msg: &Message{
				Text: "Acesse https://www.linkedin.com/jobs/view/555777/",
			},
			want: "555777",
		},
		{
			name: "sem URL nenhuma",
			msg: &Message{
				Text: "Encontradas 3 vagas novas:",
			},
			want: "",
		},
		{
			name: "mensagem nula",
			msg:  nil,
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := JobIDFromMessage(tt.msg); got != tt.want {
				t.Errorf("JobIDFromMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestJobIDFromURL(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://www.linkedin.com/jobs/view/3871234567/?alternateChannel=search&position=1&pageNum=0", "3871234567"},
		{"https://www.linkedin.com/jobs/view/3871234567/?alternateChannel=search&amp;position=1", "3871234567"},
		{"https://br.linkedin.com/jobs/view/987654", "987654"},
		{"https://www.linkedin.com/jobs/view/123456/", "123456"},
		{"https://cf.linkedin.com/jobs/view/junior-calculator-at-terberg-totaal-installaties-4371654421?position=1&pageNum=0&refId=SD1szjv8spYHEDXDfJ0qwA%3D%3D&trackingId=kjfYLZ%2FxXCQgcflZt91zTQ%3D%3D", "4371654421"},
		{"https://ca.linkedin.com/jobs/view/python-developer-at-iris-software-inc-4464493957", "4464493957"},
		{"", ""},
		{"https://exemplo.com/nao-e-vaga", ""},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			if got := JobIDFromURL(tt.url); got != tt.want {
				t.Errorf("JobIDFromURL(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestExtractJobID(t *testing.T) {
	text := "Olha isso > https://www.linkedin.com/jobs/view/555777/?capColoOverride=true&position=1 ah e &amp; tal"
	if got := ExtractJobID(text); got != "555777" {
		t.Errorf("ExtractJobID() = %q, want %q", got, "555777")
	}
}
