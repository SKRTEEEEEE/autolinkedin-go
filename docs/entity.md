# Entity definition
> *Definición de la __estructura__ de las entidades utilizadas en la aplicación*

## All use
### Auto?
- `id`
- `created_at`: timestamp de creación
- `updated_at`: ?idk?
## Topic
### Obligatorios
- `name`: [unique] nombre descriptivo, usado por la IA y usuario
- `description`: para el usuario
### Default
- `category`: categoría/clasificación (por defecto "General")
- `priority`: importancia a la hora de crear ideas (1-10, por defecto 5)
- `ideas`: numero que indica la cantidad de ideas a partir de dicho topic (por defecto 2)
- `active`: booleano que indica si el topic está activo
- `prompt`: (NEW) prompt.'name' que utiliza dicho topic (por defecto 'base1')
### Optional
- `related_topics`: array de términos relacionados('name' de otros topic)
### [Auto](#auto)
## Prompt
### Obligatorios
- `name`: [unique] (actual, 'style_name') nombre identificativo del prompt
- `type`: para que se utiliza el prompt (ideas | draft)
- `prompt_template`: texto en formate plano(/n, etc..), con uso de {} y campos reservados (🏗️✏️por definir) para utilizar los campos de 'topic' y la app.
### Default
- `active`: booleano que indica si el prompt está activo
### [Auto](#auto)
- `user_id`: id del usuario que utiliza el prompt (para seed, devUserId)
## Idea
- `content`: Texto de la idea (OLD->10-5000 caracteres, NEW->10-200 caracteres)
### Default
- `quality_score`: Puntuación opcional (0.0-1.0, default 0.0)
- `used`: Booleano que indica si ya se utilizó para generar drafts (default false)
### [Auto](#auto)
- `expires_at`: Fecha de expiración (30 días por defecto)
- `user_id`: ID del usuario propietario
- `topic_id`: ID del topic relacionado
- `topic_name`: (NEW) unique name del topic relacionado