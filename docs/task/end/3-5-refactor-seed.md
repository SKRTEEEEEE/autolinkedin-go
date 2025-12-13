refactor(seed): Refactor: seed prompt manager. Closes #3

## Resumen de Cambios

Se ha corregido el sistema de carga de prompts desde archivos seed para utilizar correctamente el servicio `PromptLoader` que parsea adecuadamente el frontmatter YAML de los archivos markdown, en lugar de utilizar un parser personalizado incompleto.

## Qué Cambió

### Refactorización de `SeedSyncService`

**Archivo modificado:** `src/infrastructure/services/seed_sync_service.go`

#### Eliminado
- Método `processPromptFile()` - Parser personalizado obsoleto
- Método `parsePromptFile()` - Lógica de parseo basada en heurísticas
- Función helper `contains()` - Utilidad para búsqueda de strings

#### Actualizado
- `SeedPromptsFromFiles()`: Ahora utiliza `PromptLoader.LoadPromptsFromDir()` y `PromptLoader.CreatePromptsFromFile()` para cargar y procesar los archivos de prompts correctamente
- Mejora en los logs para incluir el tipo de prompt en los mensajes de éxito

### Corrección de Archivos Seed

**Archivos modificados:**
- `seed/prompt/base2.idea.md` - Corregido el nombre en frontmatter de "base1" a "base2"

**Archivos eliminados:**
- `seed/prompt/base1.old.md` - Archivo legacy eliminado

## Por Qué

El problema reportado en el issue #3 era que al agregar un nuevo archivo de prompt (`base2.idea.md`) al directorio `seed/prompt/`, este no se cargaba correctamente en la base de datos o no era devuelto por el endpoint de API.

La causa raíz era que `SeedSyncService.processPromptFile()` utilizaba un parser personalizado muy básico que:
1. No parseaba el frontmatter YAML correctamente
2. Intentaba deducir el nombre y tipo del prompt mediante búsqueda de strings en el contenido
3. No utilizaba el servicio `PromptLoader` ya existente y correctamente implementado

## Cómo se Validó

### 1. Compilación y Formato
```bash
cd src && go build ./...
cd src && go fmt ./...
```
✅ Todo el código compila sin errores
✅ Formato aplicado correctamente

### 2. Docker Modo Desarrollo (Watch Mode)
```bash
docker-compose up -d --build
```
✅ La aplicación inicia correctamente
✅ El modo watch está activo y detecta cambios
✅ Los logs muestran la carga exitosa del nuevo prompt base2:
```
{"level":"info","msg":"Prompt created successfully","name":"base2","type":"ideas","prompt_id":"693dc0051db99e740b5eecc2"}
```

### 3. Verificación de API
```bash
GET http://localhost:8080/v1/prompts/000000000000000000000001
```
✅ El endpoint devuelve los 3 prompts correctamente:
- `base1` (type: ideas)
- `base2` (type: ideas) ← **Nuevo prompt cargado correctamente**
- `profesional` (type: drafts)

### 4. Workflow HTTP
Se verificó que el flujo completo de generación de drafts funciona correctamente según `test/http/workflow-example/draft-generation.http`:
✅ Todos los prompts son accesibles
✅ Los topics pueden referenciar cualquiera de los prompts
✅ La generación de ideas y drafts funciona correctamente

## Riesgos y Notas

### Riesgos Bajos
- **Cambio de implementación interna**: Aunque se eliminó código, el comportamiento externo se mantiene idéntico pero con mejor parsing
- **Compatibilidad**: El formato de archivos seed no ha cambiado, sigue siendo frontmatter YAML + contenido markdown

### Notas Importantes
1. **PromptLoader ya existía**: El servicio `PromptLoader` con parsing correcto de frontmatter YAML ya estaba implementado y siendo utilizado por `DevSeeder`, simplemente no era utilizado por `SeedSyncService`
2. **Reducción de código**: Se eliminaron ~100 líneas de código duplicado/obsoleto
3. **Mejor mantenibilidad**: Ahora hay un único punto de entrada para el parsing de archivos de prompts

## Próximos Pasos Sugeridos

Aunque no son parte de este issue, se sugieren las siguientes mejoras futuras:
1. Agregar tests unitarios específicos para `SeedSyncService.SeedPromptsFromFiles()`
2. Considerar agregar validación de referencias de prompts en topics durante el seeding
3. Documentar el formato esperado de archivos seed en `seed/prompt/README.md`
