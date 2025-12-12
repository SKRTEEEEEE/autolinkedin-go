pre-fix: Refactorización Fase 1: Generación de Ideas. Closes #3

- refactor: GenerateIdeasUseCase usa el prompt del topic con fallback, respeta `topic.ideas`, limita la longitud de contenido y persiste `topic_name` en las ideas.
- refactor: saneado de ideas con recorte a 200 caracteres y padding mínimo para cumplir las validaciones del dominio.
- chore: DevSeeder reutiliza el caso de uso y carga prompts desde seed con un adaptador de logger zap.
- fix: TopicRepository actualiza `ideas`, `prompt` y `related_topics` al persistir cambios.

Validaciones:
- go test ./... (src) ✅
- go test ./application/usecases (test) ⚠️ no ejecutado: el módulo `test` exige paquetes ausentes en `src` (mocks/dto) según `go mod tidy`, se requiere resolver dependencias antes de correr la suite.
