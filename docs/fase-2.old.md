# Flujo Detallado de la Aplicación - Fase 0-2
> *Esta pag contiene el resumen e indice del flujo de la aplicación de las Fases 0, 1 y 2*
> - Este flujo se puede probar manualmente con los endpoints descritos en [draft-generation.http](../test/http/workflow-example/draft-generation.http)
## Fase 0 — Inicialización

### 0.1 Bootstrap de Usuario (devUser)

Al iniciar la app, se verifica si existe el usuario `devUser` en MongoDB. Si no existe, se crea automáticamente:

```go
{
  "username": "devUser",
  "language": "es",
  "active": true,
  "created_at": timestamp
}
```

### 0.2 Seeding de Topics Default

Se verifica si `devUser` tiene topics. Si no, se crean 3 topics por defecto:

```go
topics := []Topic{
  {Name: "Inteligencia Artificial", Description: "..."},
  {Name: "Backend Development", Description: "..."},
  {Name: "TypeScript", Description: "..."},
}
```

### 0.3 Seeding de Prompts Default

Se crean 2 prompts por defecto:

1. **Prompt para generar Ideas**:
```
Eres un experto en {[related_topics]}.
Genera {ideas} ideas únicas sobre {name} para posibles posts o artículos.
Idioma: {language}
```

2. **Prompt para generar Drafts (estilo: profesional)**:
```
[PARTE_FIJA][?][Eres experto en LinkedIn.]
Genera contenido profesional basado en: {idea}
```

### 0.4 Generación Inicial de Ideas

Para cada topic creado, se generan X ideas usando el prompt default de ideas:
- Total inicial: **6 ideas** (2 por cada topic)
- Se almacenan en colección `ideas` con campo `used: false`


## Fase 1 — Generación de Ideas 

### 1.1 Activación 

#### 1.1.1 Activación por Scheduler - FUTURO -> NO IMPLEMENTAR AHORA

Una goroutine se ejecuta cada **X horas** (configurable):

```go
ideasScheduler := scheduler.NewIdeasScheduler(generateIdeasUC, userRepo, interval)
go ideasScheduler.Start(ctx)
```

#### 1.1.2 Activación por inicio app
#### 1.1.3 Activación por modificación de topics

### 1.2 Funcionamiento

Para cada usuario activo:
1. Lee todos sus topics **habilitados** (`disabled: false`)
2. Genera **N ideas** para cada topic (configurable por topic o global)
3. Usa el **prompt de ideas** configurado por el usuario (o default)

### 1.3 Llamada al LLM

Construcción del prompt:
```
{prompt_template}
con variables: {topic.name}, {topic.description}, {user.language}
```

Configuración:
- Timeout: 30s
- Reintentos: 3 con exponential backoff
- Si falla → log error y continuar

### 1.4 Persistencia
### 1.5 Endpoints de Ideas
## Fase 0.5 — Gestión de Topics
### 0.5.1 Crear Topic
**Endpoint**: `POST /v1/topics`

```json
{
  "name": "Machine Learning",
  "description": "Algoritmos y aplicaciones de ML"
}
```

**Trigger automático**: Se generan 10 ideas iniciales para el nuevo topic.

### 0.5.2 Modificar Topic

**Endpoint**: `PATCH /v1/topics/:topicId`

```json
{
  "name": "AI & ML",
  "description": "nueva descripción"
}
```

Las ideas previas **no se regeneran**.

### 0.5.3 Eliminar Topic (Soft Delete)

**Endpoint**: `DELETE /v1/topics/:topicId`

Marca el topic como `disabled: true`. El scheduler lo ignora pero las ideas persisten.

**Reactivación**: `PATCH /v1/topics/:topicId {"disabled": false}`


## Fase 0.6 — Gestión de Prompts (Por Revisar)

### 0.6.1 Listar Prompts/Estilos

```
GET /v1/prompts
```

Devuelve todos los prompts (ideas y drafts) del usuario.

### 0.6.2 Crear/Modificar Prompt

```
POST /v1/prompts
PATCH /v1/prompts/:id
```

Permite definir:
- Prompts para generar **drafts** vinculados a un **estilo**

Colección `prompts`:
```json
{
  "user_id": ObjectId,
  "type": "ideas" | "drafts",
  "style_name": "professional" | "technical" | ...,
  "prompt_template": "texto del prompt (parte fija)",
  "active": true
}
```



## Fase 2 — Generación de Drafts (Por Revisar)

### 2.1 Trigger Manual

El usuario solicita generar drafts desde una idea:

**Endpoint**: `POST /v1/drafts/generate`

```json
{
  "idea_id": "ObjectId",
  "style": "professional"  // opcional, usa prompt vinculado al estilo
}
```

- [ ] **Es opcional el campo style**, si el usuario no se introduce se usa por default, professional

### 2.2 Encolado en NATS

1. Valida que la idea existe y pertenece al usuario
2. Publica mensaje en queue `drafts.generate`:
```json
{
  "job_id": "uuid",
  "user_id": "ObjectId",
  "idea_id": "ObjectId",
  "idea_text": "texto",
  "style": "professional"
}
```
3. Responde `202 Accepted` con `job_id`

### 2.3 Worker de Drafts

`DraftGenerationWorker` consume el mensaje y ejecuta:

1. Obtiene el **prompt de drafts** vinculado al estilo solicitado
2. Realiza **2 llamadas al LLM en paralelo**:

   **Llamada 1 — 5 Posts**:
   ```
   {prompt_draft_template}
   Genera 5 posts con: {idea_text}
   Estilo: {style}
   ```

   **Llamada 2 — 1 Artículo**:
   ```
   {prompt_draft_template}
   Genera 1 artículo con: {idea_text}
   Estilo: {style}
   ```

3. Guarda 6 drafts en MongoDB:
```json
{
  "job_id": "uuid",
  "user_id": ObjectId,
  "idea_id": ObjectId,
  "type": "post" | "article",
  "content": "texto generado",
  "style": "professional",
  "version": 1,
  "status": "draft"
}
```

4. Marca la idea como `used: true`
5. Actualiza el job: `status: "completed"`

### 2.4 Consulta de Estado

```
GET /v1/drafts/jobs/:jobId
```

Responde con `status: pending|processing|completed|failed`
