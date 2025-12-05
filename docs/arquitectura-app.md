

# 🏗️ Arquitectura Técnica — LinkGen AI (Monolito + Clean Architecture, Versión Simplificada)

Este documento describe la arquitectura técnica actual de LinkGen AI, implementada como un **monolito en Go** siguiendo principios de **Clean Architecture**.
Incluye la lógica de generación automática de ideas, creación de borradores, refinamiento iterativo y publicación en LinkedIn.

Su propósito es servir como **referencia técnica central**. El archivo `AGENTS.md` deberá **referenciar este documento** sin duplicar su contenido.

---

# 1. Visión General

LinkGen AI consiste en una única aplicación backend escrita en Go que integra:

* Un modelo LLM accesible vía HTTP
* MongoDB para persistencia
* NATS para colas de trabajos (uso mínimo, corto y simple)
* Scheduler interno para generación automática de ideas
* Endpoints HTTP REST para interacción con usuarios y clientes CLI

El monolito expone toda la API pública y ejecuta todos los procesos internos (scheduler, workers, orquestadores).

---

# 2. Clean Architecture (Estructura)

La aplicación está organizada en cuatro capas independientes:

```
/domain          ← Reglas de negocio puras
/application     ← Casos de uso y orquestación
/infrastructure  ← Implementaciones concretas (MongoDB, NATS, LLM HTTP…)
/interfaces      ← Entradas/salidas del sistema (HTTP Handlers)
```

## 2.1 domain (Enterprise Business Rules)

Contiene:

* Entidades: User, Topic, Idea, Draft
* Lógica de negocio pura, sin dependencias
* Interfaces abstractas (puertos):

  * LLMService
  * DraftRepository
  * IdeasRepository
  * TopicsRepository
  * UserRepository
  * PublisherService
  * QueueService

Nada aquí depende de infraestructura ni de frameworks.

---

## 2.2 application (Use Cases Layer)

Implementa la lógica de alto nivel de la aplicación:

* Generación periódica de ideas (scheduler)
* Generación de drafts (5 posts + 1 artículo)
* Refinamiento de contenido
* Publicación en LinkedIn
* Gestión de errores, reintentos y backoff
* Encolado de trabajos de larga duración (Draft Generation)

Aquí residen los *casos de uso*, combinando entidades y servicios abstractos del domain.

---

## 2.3 infrastructure (Adapters Layer)

Implementa todos los detalles concretos:

* Repositorios MongoDB
* Cliente HTTP hacia el LLM local
* Cliente HTTP hacia LinkedIn
* Worker de NATS (cola simple y persistencia temporal)
* Configuración, logging, utilidades
* Implementación concreta de los puertos definidos en domain

Esta capa puede cambiar sin afectar al domain ni al application.

---

## 2.4 interfaces (Frameworks & Drivers Layer)

Punto de entrada del sistema:

* Handlers HTTP
* Serialización/deserialización JSON
* Validación de datos
* Middlewares (API Key, etc.)

Los handlers **solo** llaman a casos de uso, nunca directamente a infraestructura.

---

# 3. Flujo General de la Aplicación

A continuación, se describen las fases principales de la aplicación, enfocadas en procesos internos y llamados a través del API.

---

# 3.1 Fase 1 — Generación Periódica de Ideas (Scheduler)

Representa la parte central de automatización y ha sido simplificada para ser clara y robusta.

## 3.1.1 Activación del Scheduler

Una goroutine interna se ejecuta cada **X horas** (valor configurable).

Para cada usuario activo:

1. Se leen sus temas (`userTopics`)
2. Se elige **un tema aleatorio**
3. Se genera un lote de **N ideas nuevas** (por defecto 10) usando el LLM local
4. Se almacenan en MongoDB
5. Se acumulan en el tiempo según el usuario

La generación se realiza de forma silenciosa y continua.

---

## 3.1.2 Llamada al LLM

El caso de uso genera prompts estándar basados en:

```
topic random seleccionado
configuración del usuario
historial si es necesario
```

La llamada al LLM se realiza mediante un cliente HTTP con:

* timeouts cortos
* exponential backoff
* reintentos limitados

Si todos los intentos fallan, se descarta la generación actual sin afectar al resto del sistema.

---

## 3.1.3 Persistencia de Ideas

Las ideas se **acumulan** en MongoDB, sin sobrescribir las previas.

### Colección: `ideas`

| Campo      | Tipo     | Descripción                |
| ---------- | -------- | -------------------------- |
| _id        | ObjectId | PK                         |
| user_id    | ObjectId | Usuario que recibe la idea |
| topic      | string   | Tema origen                |
| idea       | string   | Texto generado             |
| created_at | datetime | Fecha de generación        |

---

## 3.1.4 Limpieza de Ideas

La API expone un endpoint:

```
DELETE /v1/ideas/{user_id}/clear
```

Elimina las ideas acumuladas del usuario.

---

## 3.1.5 Consulta de Ideas

```
GET /v1/ideas/{user_id}
```

Parámetros opcionales:

* `topic=...`
* `limit=...`

La lectura es directa desde MongoDB.

---

# 3.2 Fase 2 — Generación de Drafts (Asíncrona mediante NATS sencilla)

Esta fase puede tardar varios segundos, por lo que se realiza de forma asíncrona usando **una cola muy simple de NATS**.

### Flujo:

1. La API recibe la petición del usuario
2. Publica un mensaje en NATS (TTL corto)
3. Devuelve `202 Accepted`
4. Un worker consume el mensaje
5. Llama al LLM para:

   * 5 drafts de posts
   * 1 artículo
6. Guarda los drafts en MongoDB
7. Marca el estado como `DRAFT_READY`

No se utiliza NATS para persistir ideas ni metadatos, solo como **cola temporal**.

---

# 3.3 Fase 3 — Refinamiento Interactivo (Sincrónico)

El usuario puede mejorar un borrador con un prompt personalizado.

```
POST /v1/drafts/{draft_id}/refine
```

Flujo:

1. Se obtiene el borrador desde MongoDB
2. Se construye el mensaje contextual
3. Se llama al LLM local
4. Se actualiza el contenido y el historial
5. Se guarda la versión refinada en MongoDB

Es completamente sincrónico.

---

# 3.4 Fase 4 — Publicación en LinkedIn

```
POST /v1/drafts/{draft_id}/publish
```

El flujo:

1. Se validan credenciales del usuario
2. Se elige la estrategia de publicación:

   * Post simple
   * Artículo académico
3. Se llama al LinkedIn API a través de infrastructure
4. Se actualiza el estado del borrador (`PUBLISHED` o `PUBLISH_FAILED`)

---

# 4. Modelo de Datos (MongoDB)

### Colección: `users`

Token de LinkedIn, API keys y config LLM.

### Colección: `userTopics`

Temas preferidos del usuario.

### Colección: `ideas`

Ideas acumuladas generadas periódicamente (ver Fase 3.1).

### Colección: `drafts`

Posts y artículos generados, refinados y eventualmente publicados.

---

# 5. Resumen de Responsabilidades

| Capa           | Responsabilidad                                     |
| -------------- | --------------------------------------------------- |
| domain         | Entidades, reglas de negocio, interfaces abstractas |
| application    | Casos de uso, orquestación de procesos, scheduler   |
| infrastructure | DB, NATS, LLM HTTP, LinkedIn API                    |
| interfaces     | API HTTP y validación                               |

---

# 6. Relación con AGENTS.md

`AGENTS.md` debe:

* Referenciar este documento cuando necesite detalles técnicos
* Definir únicamente flujos operativos para agentes
* No duplicar datos de arquitectura
* Respetar los límites y responsabilidades definidos aquí
