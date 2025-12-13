# Prompt Variable System Documentation (Fase 2)

## Overview

El sistema de variables en prompts permite generar contenido dinámico y personalizado sustituyendo placeholders en las plantillas con los datos de los topics, ideas, y configuración del usuario.

## Tipos de Variables Disponibles

### Variables para Prompts de Ideas

Estas variables se pueden usar en prompts con `type: "ideas"`:

| Variable | Fuente | Descripción | Ejemplo de Salida |
|----------|--------|-------------|-------------------|
| `{name}` | topic.name | Nombre del topic | "React Hooks" |
| `{ideas}` | topic.ideas_count | Número de ideas a generar | "5" |
| `{ideas_count}` | topic.ideas_count | Alternativa para `{ideas}` | "5" |
| `{category}` | topic.category | Categoría del topic | "Frontend" |
| `{priority}` | topic.priority | Prioridad (1-10) | "8" |
| `{[keywords]}` | topic.keywords | Array convertido a texto | "react, hooks, state, effects" |
| `{[related_topics]}` | topic.related_topics | Topics relacionados | "JavaScript, Node.js, Redux" |

### Variables para Prompts de Drafts

Estas variables se pueden usar en prompts con `type: "drafts"`:

| Variable | Fuente | Descripción | Formato |
|----------|--------|-------------|---------|
| `{content}` | idea.content | Contenido original de la idea | Texto plano |
| `{topic_name}` | idea.topic_name | Nombre del topic relacionado | String |
| `{user_context}` | user.configuration | Contexto del usuario formateado | Formato específico (ver abajo) |

### Formato de {user_context}

La variable `{user_context}` se construye desde la configuración del usuario:

```
Name: Ana García
Expertise: Full Stack Development
Tone: Profesional
```

**Campos soportados:**
- `name` → "Name: {value}"
- `expertise` → "Expertise: {value}"  
- `tone_preference` → "Tone: {value}"

## Ejemplos Prácticos

### Ejemplo 1: Prompt de Ideas Básico

**Template:**
```
Generate {ideas} ideas about {name} with keywords: {[keywords]}
```

**Topic:**
```json
{
  "name": "Microservices with Go",
  "ideas_count": 5,
  "keywords": ["gRPC", "Docker", "Kubernetes", "API Gateway"]
}
```

**Resultado:**
```
Generate 5 ideas about Microservices with Go with keywords: gRPC, Docker, Kubernetes, API Gateway
```

### Ejemplo 2: Prompt de Ideas Complejo

**Template:**
```
As an expert in {category}, generate {ideas} innovative ideas for {name}.

Focus areas: {[keywords]}
Integration with: {[related_topics]}
Priority level: {priority}/10

Each idea should be:
- Actionable and specific
- Suitable for LinkedIn posts (10-200 characters)
- Relevant to professionals interested in {name}
```

**Topic:**
```json
{
  "name": "Cloud Security Best Practices",
  "category": "Security",
  "ideas_count": 7,
  "priority": 9,
  "keywords": ["AWS", "IAM", "VPC", "Encryption", "Monitoring"],
  "related_topics": ["DevOps", "Compliance", "AWS Services"]
}
```

**Resultado:**
```
As an expert in Security, generate 7 innovative ideas for Cloud Security Best Practices.

Focus areas: AWS, IAM, VPC, Encryption, Monitoring
Integration with: DevOps, Compliance, AWS Services
Priority level: 9/10

Each idea should be:
- Actionable and specific
- Suitable for LinkedIn posts (10-200 characters)
- Relevant to professionals interested in Cloud Security Best Practices
```

### Ejemplo 3: Prompt de Drafts

**Template:**
```
Based on the following topic and idea, create LinkedIn content:

Topic: {topic_name}
Idea: {content}

User Profile:
{user_context}

Requirements:
- Write 5 posts (120-260 words each)
- Create 1 comprehensive article
- Use professional tone adapted to user
- Include actionable insights
- Format as valid JSON with "posts" and "articles" arrays
```

**Idea:**
```json
{
  "content": "Implementando Circuit Breaker con Istio para resilient microservices",
  "topic_name": "Kubernetes Production"
}
```

**User Context ( generado automáticamente ):**
```
Name: Carlos Rodríguez
Expertise: Cloud Architecture
Tone: Technical
```

**Resultado:**
```
Based on the following topic and idea, create LinkedIn content:

Topic: Kubernetes Production
Idea: Implementando Circuit Breaker con Istio para resilient microservices

User Profile:
Name: Carlos Rodríguez
Expertise: Cloud Architecture
Tone: Technical

Requirements:
- Write 5 posts (120-260 words each)
- Create 1 comprehensive article
- Use professional tone adapted to user
- Include actionable insights
- Format as valid JSON with "posts" and "articles" arrays
```

## Implementación

### Flujo de Procesamiento

1. **Template Parsing:** Identificar variables en el prompt template
2. **Variable Resolution:** Obtener valores de topic, idea, o usuario
3. **Array Processing:** Convertir arrays a texto para variables con `[]`
4. **Template Filling:** Sustituir variables con sus valores
5. **Validación:** Verificar que no queden variables sin reemplazar

### Reglas de Sustitución

#### Arrays con ` {[variable]} `
Los arrays se convierten a texto separado por comas con espacios:

```javascript
// Input: ["React", "Hooks", "Context", "Effects"]
// Output: "React, Hooks, Context, Effects"
```

#### Precedencia de Variables
Las variables se resuelven en este orden:
1. Variables específicas del contexto:
   - Ideas: Variables del topic
   - Drafts: Variables de idea + usuario
2. Variables globales (si existen)
3. Variables sin resolver → error

### Manejo de Errores

1. **Variable desconocida:** Error con lista de variables válidas
2. **Topic sin variable requerida:** Error indicando campo faltante
3. **Array vacío:** Variable se reemplaza con string vacío ""
4. **Referencia circular entre topics:** Error detectando dependencia cíclica

## Seed Directory Structure

```
seed/
├── prompt/
│   ├── base1.idea.md      # Prompt base para generar ideas
│   ├── base1.old.md       # Versión anterior (obsoleto)
│   └── pro.draft.md       # Prompt profesional para drafts
└── topic.json             # Configuración de topics con prompts
```

### Example: seed/prompt/base1.idea.md

```json
{
  "name": "base1",
  "type": "ideas",
  "prompt_template": "Generate {ideas} ideas about {name} with keywords: {[keywords]}",
  "active": true
}
```

### Example: seed/topic.json

```json
[
  {
    "name": "React Development",
    "description": "Modern React patterns and best practices",
    "prompt": "base1",
    "category": "Frontend",
    "priority": 8,
    "ideas_count": 5,
    "keywords": ["Hooks", "Context", "Performance", "Testing"],
    "related_topics": ["JavaScript", "Redux", "TypeScript", "Jest"]
  }
]
```

## Testing Considerations

### Variables Testing

1. **Content length validation:** 10-200 caracteres para ideas
2. **Array formatting:** Correcta conversión a texto
3. **Context building:** Formato correcto de {user_context}
4. **Template validation:** Variables reconocidas

### Integration Testing

1. **Prompt selection:** Correcto uso de prompts por topic
2. **Dynamic generation:** Ideas variadas con diferentes variables
3. **Draft personalization:** Contenido adaptado al usuario
4. **Performance:** Eficiente sustitución de variables

## Migration Notes

### Desde Hardcoded Prompts

**Antes:**
```go
func BuildIdeasPrompt(topic string, count int) string {
    return fmt.Sprintf("Generate %d ideas about %s", count, topic)
}
```

**Después:**
```go
func BuildIdeasPrompt(topic *Topic, prompt *Prompt) string {
    // Variable replacement in template
    return replaceVariables(prompt.Template, topic)
}
```

### Cambios en Schema

**Topics:**
- Agregar: `prompt`, `category`, `priority`, `keywords`, `related_topics`
- Renombrar: `ideas` → `ideas_count`

**Ideas:**
- Agregar: `topic_name` (normalización)

**Prompts:**
- Estandarizar: `name` en lugar de `style_name`
- Validar: `type` values ("ideas" | "drafts")

## Best Practices

### Template Design

1. **Ser explícito:** Usar nombres de variables claros
2. **Incluir contexto:** Proporcionar suficiente información al LLM
3. **Validar output:** Especificar formato y restricciones
4. **Ser consistente:** Usar patrones similares en prompts relacionados

### Variable Usage

1. **Keywords relevantes:** 3-7 términos técnicos o conceptuales
2. **Related topics:** Conexiones lógicas entre topics
3. **Priority matching:** Reflectir urgencia/importancia en contenido
4. **Category specificity:** Usar categorías significativas

### Performance Considerations

1. **Caching:** Cache de templates procesados
2. **Pre-compilation:** Validar templates al momento de carga
3. **Efficient arrays:** Optimizar conversión de array a texto
4. **Memory management:** Liberar recursos después del procesamiento

## Future Enhancements

### Variables Propuestas

1. `{date}`: Fecha actual (para contenido temporal)
2. `{trending}`: Tópicos trending relacionados
3. `{industry}`: Industria del usuario
4. `{experience}`: Años de experiencia del usuario

### Conditional Variables

1. `{if priority > 7}`: Contenido especial para high priority
2. `{if keywords contains "AI"}}: Variaciones basadas en keywords
3. `{if tone == "professional"}}: Ajustes basados en preferencias

Este sistema de variables proporciona flexibilidad para generar contenido altamente personalizado mientras mantiene consistencia y estructura en la generación de posts y artículos de LinkedIn.
