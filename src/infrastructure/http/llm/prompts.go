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
	return fmt.Sprintf(`You are an expert LinkedIn content writer. Create professional LinkedIn content based on this idea:

Idea: %s

User Context: %s

Requirements:
1. Create 5 LinkedIn posts (each 100-300 words):
   - Engaging opening
   - Clear value proposition
   - Professional tone
   - Include call-to-action
   - Use emojis sparingly

2. Create 1 LinkedIn article (500-1000 words):
   - Compelling title
   - Well-structured with sections
   - Deep insights and examples
   - Professional and informative
   - Include conclusion

Return ONLY a JSON object with this exact format:
{
  "posts": ["post1", "post2", "post3", "post4", "post5"],
  "articles": ["article1"]
}`, idea, userContext)
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
