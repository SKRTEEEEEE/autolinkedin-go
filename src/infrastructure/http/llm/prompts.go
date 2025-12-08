package llm

import (
	"fmt"
	"strings"
)

// BuildIdeasPrompt generates a prompt for idea generation
func BuildIdeasPrompt(topic string, count int) string {
	return fmt.Sprintf(`You are an expert LinkedIn content strategist. Generate %d unique and engaging content ideas about the following topic:

Topic: %s

Requirements:
- Each idea should be specific and actionable
- Ideas should be diverse and cover different angles
- Focus on professional value and insights
- Keep ideas concise (1-2 sentences each)
- Make them suitable for LinkedIn audience

Return ONLY a JSON object with this exact format:
{"ideas": ["idea1", "idea2", "idea3", ...]}`, count, topic)
}

// BuildDraftsPrompt generates a prompt for draft generation
func BuildDraftsPrompt(idea string, userContext string) string {
	trimmedIdea := strings.TrimSpace(idea)
	trimmedContext := strings.TrimSpace(userContext)

	return fmt.Sprintf(`[PARTE_FIJA][?][Eres experto en LinkedIn.]
Genera contenido profesional basado en: %s

Contexto adicional del usuario:
%s

Instrucciones clave:
- Escribe SIEMPRE en español neutro profesional.
- Cada post debe tener 120-260 palabras, abrir con un gancho potente y cerrar con una CTA o pregunta.
- El artículo debe tener título atractivo, introducción, desarrollo con viñetas o subtítulos y conclusión clara.
- No inventes datos sensibles, pero puedes añadir insights inspirados en mejores prácticas.
- No utilices comillas triples, bloques de código ni texto fuera del JSON.

Formato de salida OBLIGATORIO (sin comentarios, sin Markdown, sin texto extra):
{
  "posts": [
    "Post 1 completo en una sola cadena con \n para saltos de línea",
    "Post 2 completo",
    "Post 3 completo",
    "Post 4 completo",
    "Post 5 completo"
  ],
  "articles": [
    "Título del artículo\n\nCuerpo del artículo con secciones y conclusión"
  ]
}

Asegúrate de que el JSON sea válido, escapando caracteres especiales según el estándar.`, trimmedIdea, trimmedContext)
}

// BuildRefinementPrompt generates a prompt for draft refinement
func BuildRefinementPrompt(draft string, userPrompt string, history []string) string {
	var sb strings.Builder

	sb.WriteString("You are an expert LinkedIn content editor. Refine the following draft based on user feedback.\n\n")
	sb.WriteString(fmt.Sprintf("Current Draft:\n%s\n\n", draft))

	if len(history) > 0 {
		sb.WriteString("Previous Refinement History:\n")
		for i, h := range history {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, h))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("User Feedback: %s\n\n", userPrompt))
	sb.WriteString("Requirements:\n")
	sb.WriteString("- Apply the user's feedback accurately\n")
	sb.WriteString("- Maintain professional tone\n")
	sb.WriteString("- Keep the core message intact\n")
	sb.WriteString("- Improve clarity and engagement\n\n")
	sb.WriteString("Return ONLY a JSON object with this exact format:\n")
	sb.WriteString(`{"refined": "refined draft content here"}`)

	return sb.String()
}
