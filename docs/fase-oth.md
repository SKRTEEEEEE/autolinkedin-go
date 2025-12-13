

# Flujo Detallado de la Aplicación - Fase +2




## Fase 3 — Refinamiento de Drafts

### 3.1 Solicitud de Refinamiento

**Endpoint**: `POST /v1/drafts/:draftId/refine`

**Request Body**:
```json
{
  "prompt": "Hazlo más técnico, añade métricas"
}
```

**Validaciones**:
- `prompt`: Requerido, mínimo 10 caracteres, máximo 500 caracteres
- `draftId`: Debe ser un MongoDB ObjectID válido
- Solo se pueden refinar drafts con status `DRAFT` o `REFINED`
- Límite máximo: 10 refinamientos por draft

### 3.2 Proceso Detallado

1. **Validación del Draft**:
   - Verifica existencia del draft
   - Confirma que pertenece al usuario
   - Valida estado refinable (`DRAFT` o `REFINED`)
   - Verifica límite de refinamientos (máximo 10)

2. **Construcción del Prompt Contextual**:
   ```
   Contenido actual: {draft.content}
   Historial de refinamientos: {refinement_entries}
   Instrucción del usuario: {prompt}

   Refina aplicando la instrucción manteniendo el contexto de versiones anteriores.
   ```

3. **Llamada al LLM**:
   - Timeout: 45 segundos
   - Síncrono (usuario espera respuesta)
   - Incluye historial como contexto para mantener coherencia

4. **Actualización del Draft**:
   - Guarda contenido refinado
   - Agrega entrada a `refinement_history`:
   ```json
   {
     "timestamp": "2025-12-13T10:30:00Z",
     "prompt": "Hazlo más técnico, añade métricas",
     "content": "texto refinado con métricas",
     "version": 2
   }
   ```
   - Actualiza status a `REFINED`
   - Actualiza `updated_at`

5. **Respuesta**:
   - `200 OK` con draft completo incluyendo historial
   - Incluye versión actual y todas las anteriores

### 3.3 Ejemplos de Uso

**Añadir Emojis y Engagement**:
```json
{
  "prompt": "Hazlo más engaging y añade emojis apropiados para LinkedIn. Incluye un gancho potente al inicio y una llamada a la acción al final."
}
```

**Estilo Técnico con Métricas**:
```json
{
  "prompt": "Transforma a estilo técnico: añade métricas específicas, datos concretos y estadísticas verificables. Cita benchmarks o estudios cuando sea posible."
}
```

**Tono Corporativo**:
```json
{
  "prompt": "Adopta un tono corporativo ejecutivo: lenguaje profesional, sin jerga informal, enfocado en ROI y beneficios de negocio. Elimina excesos de emojis."
}
```

**Optimización para LinkedIn**:
```json
{
  "prompt": "Optimiza para el algoritmo de LinkedIn: incluye 3-5 hashtags relevantes, estructura con espacios en blanco, pregunta enganchadora y keywords de industria."
}
```

**Storytelling Personal**:
```json
{
  "prompt": "Convierte en narrativa personal: incluye una breve historia o caso real, conecta emocionalmente con la audiencia, muestra vulnerabilidad y aprendizaje."
}
```

### 3.4 Encadenamiento de Refinamientos

Permite refinamientos secuenciales manteniendo todo el historial:

```
v1: Draft original (DRAFT)
  ↓ Refinar: "Añadir emojis"
v2: Draft con emojis (REFINED)
  ↓ Refinar: "Hacer más técnico"
v3: Draft técnico con emojis (REFINED)
  ↓ Refinar: "Optimizar para SEO"
v4: Draft técnico, optimizado, con emojis (REFINED)
```

**Historial Completo**:
```json
{
  "draft": {
    "id": "507f1f77bcf86cd799439011",
    "content": "Contenido final refinado",
    "status": "REFINED",
    "refinement_history": [
      {
        "timestamp": "2025-12-13T10:30:00Z",
        "prompt": "Añadir emojis",
        "content": "Contenido v1 con emojis 🚀",
        "version": 1
      },
      {
        "timestamp": "2025-12-13T10:32:00Z",
        "prompt": "Hacer más técnico",
        "content": "Contenido v2 técnico 📊 85% mejora",
        "version": 2
      },
      {
        "timestamp": "2025-12-13T10:35:00Z",
        "prompt": "Optimizar para SEO",
        "content": "Contenido final refinado #TechTips",
        "version": 3
      }
    ]
  }
}
```

### 3.5 Manejo de Errores

**400 Bad Request**:
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "prompt must be at least 10 characters"
  }
}
```

**404 Not Found**:
```json
{
  "error": {
    "code": "NOT_FOUND", 
    "message": "Draft not found"
  }
}
```

**409 Conflict (Límite Excedido)**:
```json
{
  "error": {
    "code": "LIMIT_EXCEEDED",
    "message": "refinement limit exceeded (maximum 10)"
  }
}
```

**408 Request Timeout**:
```json
{
  "error": {
    "code": "TIMEOUT",
    "message": "LLM request timeout after 45 seconds"
  }
}
```

### 3.6 Restricciones y Límites

- **Máximo 10 refinamientos** por draft
- **Solo drafts no publicados** (`DRAFT`, `REFINED`)
- **Timeout de 45s** para LLM
- **Prompt length**: 10-500 caracteres
- **Refinamiento síncrono** (usuario espera)
- **Historial inmutable** (no se puede eliminar)

### 3.7 Casos de Uso Recomendados

1. **Iteración Creativa**: Generar múltiples versiones hasta encontrar el tono perfecto
2. **A/B Testing**: Crear variaciones para probar qué contenido funciona mejor
3. **Adaptación de Audiencia**: Refinar el mismo contenido para diferentes segmentos
4. **Mejora Continua**: Partir de una versión base y refinarla progresivamente
5. **Corrección Post-Generación**: Ajustar drafts generados automáticamente

## 🏗️ 👁️ TO CHECK
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
