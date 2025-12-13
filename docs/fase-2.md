# Flujo Detallado de la Aplicación - Fase 0-2
> *Esta pag contiene el resumen e indice del flujo de la aplicación de las Fases 0, 1 y 2*
> - Este flujo se puede probar manualmente con los endpoints descritos en [draft-generation.http](../test/http/workflow-example/draft-generation.http)

## Fase 0 — Inicialización

### 0.1 Bootstrap de Usuario (devUser)

- [x] Al iniciar la app, se verifica si existe el usuario `devUser` en MongoDB. Si no existe, se crea automáticamente, con los siguientes campos como mínimo:

- "username": "devUser",
- "id": 000000000000000000000001


### 0.2 Seeding de Topics Default
- [ ] Se verifica si `devUser` tiene topics. 
  - [ ] Si no, se utiliza el archivo [topic.json](../seed/topic.json) para generar-los, utilizando los de dicho archivo
  - [ ] Si ya tiene topics, se comprobara que estén los mismos que [topic.json](../seed/topic.json), si hay alguno de dicho archivo que no esta guardo en bdd se añade.

### 0.3 Seeding de Prompts Default
##### Funcionamiento actual?
- [x] Se verifica si `devUser` tiene prompts configurados. 
  - [x] Si no hay prompts, se crean dos prompts por defecto:
    - **Prompt para ideas**: Plantilla con variables `{name}`, `{related_topics}`, `{language}`, y `{count}` que permite generar ideas para un topic específico
    - **Prompt para drafts**: Estilo "profesional" con plantilla fija para generar 5 posts y 1 artículo a partir de una idea
  - Los prompts se almacenan en la colección [`prompts`](entity.md#prompt)
##### Funcionamiento esperado
- [ ] Se verifica si `devUser` tiene prompts configurados. 
  - [ ] Si no hay prompts, se crean prompts por defecto utilizando los archivos de [/seed/prompt/](../seed/prompt/) **que no contengan el final `.old.md`**
    - [ ] Para esto se utiliza la configuración del front-matter donde esta indica los campos extra requeridos (name & type)
  - [ ] Si los hay, se comprobara que hayan los mismos que en [/seed/prompt/](../seed/prompt/) **y que no contengan el final `.old.md`**, si hay alguno de esta carpeta que no este en la bdd se creara el/los que falten.
  

### 0.4 Generación Inicial de Ideas

Para cada topic creado, se generan X (indicado en el campo 'ideas' de cada topic) ideas usando el prompt - default: base1 - de ideas:
- Se almacenan en colección [`ideas`](entity.md#idea) con campo `used: false`

1. [ ] Al iniciar la app, se tendrá que hacer una llamada al LLM **por cada topic existente**
2. [ ] Se generan X (indicado en el campo 'ideas' de cada topic) ideas
3. [ ] Se utiliza el prompt indicado en el campo 'prompt', el cual tendrá el mismo nombre que algún 'prompt_template' del user. Sino (por default) se utiliza el que tiene el nombre de 'base1'
- [Fase 1: Generación de ideas - Funcionamiento](#12-funcionamiento)
## Fase 1 — Generación de Ideas 

### 1.1 Activación 

#### 1.1.1 Activación por Scheduler - FUTURO -> NO IMPLEMENTAR AHORA

Una goroutine se ejecuta cada **X horas** (configurable):

#### 1.1.2 [Activación por inicio app](#04-generación-inicial-de-ideas)
#### 1.1.3 Activación por modificación de topics
##### Funcionamiento esperado
- [ ] Si se crea un nuevo topic, se generaran 
  - Para cada topic creado, se generan X (indicado en el campo 'ideas' de cada topic) ideas usando el prompt default de ideas:
  - Se almacenan en colección `ideas` con campo `used: false`
- [ ] Si se modifica un topic, se generaran 
  - Se eliminan las ideas de dicho topic ya generadas, antiguamente.
  - Para el topic modificado, se generan X (indicado en el campo 'ideas' de cada topic) ideas usando el prompt default de ideas
  - Se almacenan en colección `ideas` con campo `used: false`
- [ ] Si se elimina un topic, se eliminan las ideas vinculadas a dicho topic
##### Funcionamiento actual?
- [x] Si se modifica un topic:
  - El topic se actualiza con los nuevos datos (nombre, descripción, keywords, categoría, etc.)
  - NO se eliminan las ideas existentes del topic modificado
  - NO se generan automáticamente nuevas ideas
  - **Nota**: La regeneración automática de ideas tras modificación no está implementada actualmente
  
- [x] Si se elimina un topic:
  - El topic se elimina completamente de la base de datos (hard delete)
  - Las ideas vinculadas al topic pueden permanecer como registros huérfanos
  - **Nota**: No se implementa cascading delete para ideas relacionadas con topics eliminados
  

### 1.2 Funcionamiento
#### Funcionamiento actual?
- [x] El generador de ideas funciona así:
1. Se obtiene un **topic aleatorio** del usuario (de sus topics activos)
2. Se intenta recuperar el **prompt de ideas activo** de la base de datos (pero NO se utiliza realmente)
3. Llama al LLM utilizando el prompt hardcodeado en `infrastructure/http/llm/prompts.go`:
   - Usa la función `BuildIdeasPrompt(topic, count)` que incluye instrucciones predefinidas
4. Parsea la respuesta JSON extrayendo el array de ideas
5. Crea entidades Idea con metadata y las persiste en lote

**Nota importante**: Los prompts guardados en la base de datos se recuperan pero no se utilizan actualmente. La aplicación always usa los prompts hardcodeados en el código.
#### Funcionamiento esperado
- [ ] El generador de ideas funciona así, **por cada llamada al LLM necesaria (según [activación](#11-activación)):**
1. [ ] Llama al LLM utilizando el prompt indicado en el campo 'prompt' del topic, substituyendo los campos dynamicos (name, related_topics)
2. [ ] Parsea la respuesta JSON extrayendo el array de ideas
3. [ ] Crea entidades Idea
#### 1.3 Llamada al LLM

[x] La comunicación con el LLM se realiza a través del cliente HTTP configurado en `http://100.105.212.98:8317/`

[x] Configuración del cliente LLM:
- **Timeout**: 30 segundos por defecto
- **Reintentos**: 3 intentos con exponential backoff
- **Error handling**: Se registra error pero la operación continúa
- **Prompt esperado**: Debe devolver JSON válido con formato `{"ideas": ["idea1", "idea2", ...]}`

[x] La implementación actual utiliza funciones helper en `infrastructure/http/llm/prompts.go`:
- `BuildIdeasPrompt(topic, count)`: Construye el prompt hardcodeado para generación de ideas
- `BuildDraftsPrompt(idea, userContext)`: Construye el prompt hardcodeado para generación de drafts

**Nota**: Estos prompts hardcodeados sobreescriben cualquier configuración de prompts personalizados que los usuarios tengan en la base de datos.

#### 1.4 Persistencia

[x] Las ideas se almacenan en MongoDB con la siguiente estructura:
- Colección: [`ideas`](entity.md#idea)

[x] Operaciones soportadas:
- Creación en batch: `ideasRepository.CreateBatch()`
- Listado por usuario: `ideasRepository.ListByUserID()`
- Validación de contenido: mínimo 10 caracteres, máximo 5000

#### 1.5 Endpoints de Ideas

[x] API REST para gestión de ideas:
- `GET /v1/ideas/{userId}`: Lista todas las ideas del usuario
  - Parámetros query opcionales: `topic`, `limit`
  - Devuelve IDs, contenido, estado `used` y fechas
- `DELETE /v1/ideas/{userId}/clear`: Elimina todas las ideas del usuario
  - Devuelve `204 No Content` y número de ideas eliminadas

[x] Nota: No existe endpoint directo para generar ideas manualmente
- Las ideas se generan automáticamente al crear topics o al iniciar la aplicación
- La generación manual por topic podría implementarse en el futuro
## Fase 0.5 — Gestión de Topics
### 0.5.1 Crear Topic
**Endpoint**: `POST /v1/topics`

```json
{
  "name": "Machine Learning",
  "description": "Algoritmos y aplicaciones de ML",
  "user_id": "000000000000000000000001"
}
```

**Trigger automático**: Se generan X ideas iniciales para el nuevo topic de forma asíncrona.

**Estructura de Topic**:
- `name`: nombre descriptivo, usado por la IA y usuario
- `description`: para el usuario
- `related_topics`: array de términos relacionados
- `category`: categoría/clasificación (por defecto "General")
- `priority`: importancia a la hora de crear ideas (1-10, por defecto 5)
- `ideas`: numero que indica la cantidad de ideas a partir de dicho topic
- `active`: booleano que indica si el topic está activo
- `created_at`: timestamp de creación

### 0.5.2 Modificar Topic

**Endpoint**: `PUT /v1/topics/:topicId`

```json
{
  "name": "AI & ML",
  "description": "nueva descripción",
  "related_topics": ["inteligencia artificial", "machine learning"],
  "category": "Tecnología",
  "priority": 8,
  "active": true
}
```


### 0.5.3 Eliminar Topic (Hard Delete)

**Endpoint**: `DELETE /v1/topics/:topicId`

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



## Fase 2 — Generación de Drafts

### 2.1 Trigger Manual

El usuario solicita generar drafts desde una idea:

**Endpoint**: `POST /v1/drafts/generate`

```json
{
  "user_id": "000000000000000000000001",
  "idea_id": "507f1f77bcf86cd799439011"
}
```

- [x] **No existe el campo style** en la versión actual
- [x] En realidad siempre se usa el prompt hardcodeado en el código, NO el prompt de drafts guardado en BD
- [x] Se valida que la idea exista, pertenezca al usuario y no haya sido usada previamente

### 2.2 Encolado en NATS

1. Valida que la idea existe y pertenece al usuario
2. Crea un nuevo Job en la colección `jobs` con status `pending`
3. Publica mensaje en queue NATS `drafts.generate`:
```json
{
  "job_id": "uuid",
  "user_id": "ObjectId",
  "idea_id": "ObjectId",
  "timestamp": "2025-12-07T10:30:00Z",
  "retry_count": 0
}
```
4. Responde `202 Accepted` con `job_id` generado

[x] Implementación actual usa NATS para cola de mensajes asíncrona
[x] El worker procesa los mensajes en background
[x] Cada request genera un job único para seguimiento de estado

### 2.3 Worker de Drafts

`DraftGenerationWorker` consume el mensaje y ejecuta:

1. Intenta obtener el **prompt de drafts activo** del usuario de la BD, pero NO lo utiliza
2. **Única llamada al LLM**:
   ```go
   llmService.GenerateDrafts(ctx, idea.Content, userContext)
   ```
   - Usa el prompt hardcodeado del helper `BuildDraftsPrompt(idea, userContext)`
   - Este prompt ya incluye la generación de 5 posts + 1 artículo
   - Ignora cualquier configuración de prompts personalizados del usuario

3. Valida la respuesta JSON:
   - Debe contener 5 posts y 1 artículo
   - Formato esperado: `{"posts": ["post1", ...], "articles": ["article1"]}`

4. Guarda 6 drafts en MongoDB:
```json
{
  "job_id": "uuid",
  "user_id": ObjectId,
  "idea_id": ObjectId,
  "type": "post" | "article",
  "title": "titulo del draft",
  "content": "texto generado",
  "status": "draft",
  "created_at": "2025-12-07T10:30:15Z"
}
```

5. Marca la idea como `used: true`
6. Actualiza el job: `status: "completed"` con IDs de drafts generados

[x] El worker incluye reintentos automáticos (hasta 2 intentos más)
[x] Se registra errores detallados en colección `jobErrors`
[x] Soporta métricas de procesamiento y monitoreo

### 2.4 Consulta de Estado

**Endpoint**: `GET /v1/drafts/jobs/:jobId`

Responde con el estado actual y metadatos:
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "completed",
  "idea_id": "507f1f77bcf86cd799439011",
  "draft_ids": [
    "507f...post1",
    "507f...post2",
    "507f...post3",
    "507f...post4",
    "507f...post5",
    "507f...article1"
  ],
  "created_at": "2025-12-07T10:30:00Z",
  "started_at": "2025-12-07T10:30:01Z",
  "completed_at": "2025-12-07T10:30:15Z"
}
```

[x) Estados posibles:
- `pending`: Job creado, esperando procesamiento
- `processing`: Worker ejecutando la generación
- `completed`: Drafts generados exitosamente
- `failed`: Error durante la generación (revisar campo `error`)

[x) Los draft IDs se incluyen solo cuando el estado es `completed`
