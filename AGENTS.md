# Agents Configuration — LinkGen AI

> **Nota**: Este archivo define los comportamientos y normas operativas de todos los agentes automáticos.  

---

## Flujo de Git (OBLIGATORIO)

Cada commit debe:
- Estar en inglés y seguir **Conventional Commits**
- Estar firmado
- Incluir al final exactamente estas líneas:

    CO-CREATED by Agent666 — ⟦ Product of SKRTEEEEEE ⟧  
    Co-authored-by: Agent666 <agent666@skrte.ai>

**Procedimiento exacto:**
1. Escribir el mensaje completo dentro de `commit-message.txt`  
2. Ejecutar: `git commit -F commit-message.txt`  
*(Los agents nunca deben usar `-m` ni generar commits parciales.)*

---

## Estructura

Todo el código de agents debe mantenerse dentro de `./src`, respetando estrictamente las convenciones y límites definidos en:

### ➤ [Arquitectura app](./docs/arquitectura-app.md)

*(Los agents no pueden crear nuevas capas ni alterar la estructura base.)*

---

## Test

Mantener todos los tests dentro de `./test`, replicando la misma estructura jerárquica que `./src`.

### Tipos de test
Cada función debe disponer de tests y mantener un **coverage mínimo del 80%**:
- Test unitario  
- Test de integración  
- Test de performance *(solo para funciones con riesgo de carga o concurrencia)*  

*(Los agents no deben crear tests duplicados ni testear casos de uso ya cubiertos por el core.)*

---

## Docker

Los agents deben garantizar dos modos operativos:

### **1. Modo desarrollo (watch mode)**
- La aplicación debe levantarse en Docker usando un entorno en modo *watch* (hot reload).  
- Los agentes deben mantener sincronización mediante volúmenes montados localmente sin modificar la estructura del proyecto.  
- Ningún agent debe generar imágenes pesadas; deben basarse siempre en la imagen oficial definida en el proyecto.

### **2. Modo test aislado**
- La ejecución de tests debe levantarse en un contenedor limpio y efímero.  
- Deben crearse volúmenes temporales **solo para tests** (DB, cache, temp data).  
- Tras finalizar, los agents deben borrar automáticamente dichos volúmenes, asegurando un entorno reproducible y sin residuos.  

*(Los agents nunca deben compartir datos entre contenedores de desarrollo y contenedores de test.)*

---

