## Prompt
### Keys (palabras disponibles y su relación)
#### Idea
##### name
Topic.name
##### ideas
Topic.ideas
##### related_topics
Topic.related_topics
- [Array] Se muestra como lista con , -> *Ia, Backend y TypeScript*

#### Draft
##### content
Idea.content
##### user_context
- Contexto del usuario
- En teoria, esta sacando esto de la configuacion del Usario
```markdown
Análisis del userContext actual

   Viendo antes el código de buildUserContext(), lo que se está
   generando actualmente es:

   
   ```go
     if name, ok := user.Configuration["name"].(string); ok && name
     != "" {
         parts = append(parts, fmt.Sprintf("Name: %s", name))
     }

     if expertise, ok := user.Configuration["expertise"].(string);
     ok && expertise != "" {
         parts = append(parts, fmt.Sprintf("Expertise: %s",
     expertise))
     }

     if tone, ok := user.Configuration["tone_preference"].(string);
     ok && tone != "" {
         parts = append(parts, fmt.Sprintf("Tone: %s", tone))
     }
    ```
   Esto genera un texto plano como:

     Name: Juan García
     Expertise: Desarrollo Backend
     Tone: Profesional
```