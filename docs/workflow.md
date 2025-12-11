# App workflow
> *Esta pag contiene el resumen e indice del flujo de la aplicación*
## [Fase 0-2](./fase-2.md)

### App(config)
- user (hardcoded)
- idioma: *por implementar, (hardcoded -> es) idioma que se utiliza para generar las ideas*

### Prompt
- [ ] al iniciar la app se crea un prompt default para generar ideas y otro para generar draft(estilo: profesional)
- [ ] El usuario puede modificar el prompt que genera las ideas 
- [ ] El usuario puede vincular a un ESTILO cada prompt que se usara para generar los 'draft' a partir de las 'ideas'

### Topics
- [ ] al iniciar la app se cargan 3 topics por defecto
- [ ] el usuario puede introducir nuevos topics, modificar y eliminar topics

*AI, backend, nextjs, typescript, etc..*

- name: nombre descriptivo, usado por la IA y usuario
- description: para el usuario
- categoría: *por implementar*
- priority: *por implementar, importancia a la hora de crear ideas*
- related_topics: *por implementar, [ARRAY de topics] usados para generar la idea*
- ideas: *por implementar, [num] cantidad de ideas que se generara en cada schedule sobre dicho topic*

### Ideas
- [ ] se generan tantas ideas como este indicado para cada topic
  - [ ] al iniciar la app se crean ideas para los topics por defecto
  - [ ] cada vez que el usuario introduce un nuevo topic, elimina o modifica un topic
  - [ ] cada X tiempo 

*AI -> Importancia de la IA en el dia a dia del estudiante..., La burbuja de la IA, ¿verdad o mentira?, etc...*
*Backend -> El 80% de los programadores backend cobran mas, descubre..., etc...*

### Draft
- ideaId: id de la idea para crear los draft
- estilo: *por implementar, estilo vinculado a un prompt para crear los draft*

- [ ] El usuario ejecuta esta función por una 'idea' y se genera:
  - [ ] 5 'draft' o sugerencias de publicaciones listas para LinkedIn
  - [ ] 1 articulo, listo para publicar 
## [Fase +2](./fase-oth.md)