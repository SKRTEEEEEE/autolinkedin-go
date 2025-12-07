# Error Report: Tests Faltantes para Fase 2

**Fecha**: 2025-12-07  
**Tarea**: 2-7-integrar-fase-2.md  
**Fase**: PRE-BUCLE  

## Resumen

La implementación de la Fase 2 está completa en el código, pero **todos los tests son placeholders de "TDD Red phase"** que fallan inmediatamente con `t.Fatal()`. Esto no cumple con los requerimientos de AGENTS.md que exigen:

- Tests unitarios funcionales
- Tests de integración funcionales  
- Tests de performance (solo para funciones con riesgo de carga)
- **Cobertura mínima del 80%**

## Tests Faltantes o Incorrectos

### 1. `test/application/usecases/generate_drafts_usecase_test.go`
**Estado**: ❌ Todos los tests son placeholders  
**Tests requeridos**:
- ✅ TestGenerateDraftsUseCase_Success - Existe pero es placeholder
- ✅ TestGenerateDraftsUseCase_ValidationErrors - Existe pero es placeholder
- ✅ TestGenerateDraftsUseCase_UserNotFound - Existe pero es placeholder
- ✅ TestGenerateDraftsUseCase_IdeaNotFound - Existe pero es placeholder
- ✅ TestGenerateDraftsUseCase_IdeaOwnership - Existe pero es placeholder
- ✅ TestGenerateDraftsUseCase_IdeaAlreadyUsed - Existe pero es placeholder
- ✅ TestGenerateDraftsUseCase_IdeaExpired - Existe pero es placeholder
- ✅ TestGenerateDraftsUseCase_LLMIntegration - Existe pero es placeholder
- ✅ TestGenerateDraftsUseCase_LLMErrors - Existe pero es placeholder

**Acción**: Convertir todos los placeholders en tests funcionales con mocks

### 2. `test/application/usecases/refine_draft_usecase_test.go`
**Estado**: ⚠️ No revisado (probablemente también placeholder)  
**Acción**: Revisar y crear tests funcionales

### 3. `test/application/workers/draft_generation_worker_test.go`
**Estado**: ❌ Todos los tests son placeholders  
**Tests requeridos**:
- WorkerCreation
- WorkerStart/Stop
- ProcessMessage
- UseCaseExecution
- RetryLogic
- ErrorHandling
- ContextCancellation
- Concurrency

**Acción**: Convertir todos los placeholders en tests funcionales

### 4. `test/interfaces/handlers/drafts_handler_test.go`
**Estado**: ❌ Todos los tests son placeholders  
**Tests requeridos**:
- GenerateDrafts (Success, Validation, Queue Integration)
- ListDrafts (Success, Validation, Filters)
- RefineDraft (Success, Validation, LLM Timeout)
- Method validation
- Content-Type validation
- Error response format

**Acción**: Convertir todos los placeholders en tests funcionales con httptest

### 5. `test/integration/draft_generation_flow_test.go`
**Estado**: ❌ Todos los tests son placeholders  
**Tests requeridos**:
- TestDraftGenerationAsyncFlow - End-to-end async flow
- TestDraftRefinementFlow - Refinement workflow
- TestPublishToLinkedInFlow - Publishing workflow (FASE 4, no ahora)

**Acción**: Crear tests de integración con Docker (MongoDB + NATS)

### 6. `test/infrastructure/database/repositories/draft_repository_test.go`
**Estado**: ⚠️ No revisado  
**Acción**: Verificar que existan tests funcionales para CRUD operations

### 7. `test/domain/entities/draft_test.go`
**Estado**: ⚠️ No revisado  
**Acción**: Verificar tests de validación de entidades

## Plan de Acción

1. **Crear tests unitarios funcionales** para cada componente de Fase 2
2. **Crear tests de integración** que validen el flujo completo
3. **Ejecutar tests** con `make docker-test` para validar cobertura
4. **Asegurar cobertura mínima del 80%**
5. Continuar al BUCLE una vez completados los tests

## Notas

- La implementación del código está completa
- Solo faltan los tests funcionales
- Los placeholders proporcionan una buena guía de qué testear
