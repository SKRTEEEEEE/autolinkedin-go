

# Flujo Detallado de la Aplicación - Fase +2

## 🏗️ 👁️ TO CHECK


## Fase 3 — Refinamiento de Drafts

### 3.1 Solicitud de Refinamiento

**Endpoint**: `POST /v1/drafts/:draftId/refine`

```json
{
  "instruction": "Hazlo más técnico, añade métricas"
}
```

### 3.2 Proceso

1. Obtiene draft original
2. Construye prompt contextual:
```
Contenido actual: {draft.content}
Instrucción: {instruction}

Reescribe aplicando la instrucción.
```
3. Llama al LLM **síncronamente** (timeout 45s)
4. Crea nueva versión del draft:
```json
{
  "parent_draft_id": original_id,
  "version": 2,
  "content": "texto refinado",
  "refinement_instruction": "Hazlo más técnico..."
}
```
5. Responde `200 OK` con draft refinado

Permite **encadenar refinamientos**: v1 → v2 → v3 → ...


## Fase 4 — Publicación en LinkedIn

### 4.1 Validación

**Endpoint**: `POST /v1/drafts/:draftId/publish`

Valida:
1. Usuario tiene `linkedin_access_token` configurado
2. Token no está expirado (`token_expires_at`)
3. Draft pertenece al usuario

### 4.2 Publicación según Tipo

**Si `type: "post"`** → LinkedIn UGC Posts API

```
POST https://api.linkedin.com/v2/ugcPosts
```

**Si `type: "article"`** → LinkedIn Articles API

```
POST https://api.linkedin.com/v2/articles
```

### 4.3 Manejo de Respuesta

- `201 Created` → actualizar draft: `status: "published"`, `published_at: timestamp`
- `401/403` → `status: "publish_failed"`, `error: "token_invalid"`
- `429` → `status: "publish_failed"`, `error: "rate_limit"`




---

**Versión**: 2.0  
**Última actualización**: 2025-12-07
