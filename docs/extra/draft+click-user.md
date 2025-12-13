
## 1️⃣ Configura tu app de LinkedIn

1. Crea tu app en [LinkedIn Developers](https://developer.linkedin.com).
2. En **Products**, añade:

   * **Share on LinkedIn**
3. Solicita permisos:

   * `w_member_social` (para publicar posts)
   * `r_liteprofile` (opcional, para mostrar info del usuario)
   * `r_emailaddress` (opcional, si quieres vincular con email)
4. En la solicitud de aprobación, explica:

   > “Nuestra app genera borradores de posts a partir de contenido RSS. El usuario revisa y aprueba cada post antes de publicarlo. La publicación solo ocurre cuando el usuario pulsa ‘Publicar’.”

🔹 Esto cumple las políticas de LinkedIn.

---

## 2️⃣ Flujo de publicación seguro

El flujo será algo así:

1. **RSS → Generación de Draft**

   * Tu app recoge los items del RSS
   * Genera un **PostDraft** con título, descripción, enlace o imagen

2. **Interfaz de revisión**

   * Muestra los drafts al usuario
   * Permite editar texto, elegir imagen, añadir hashtags

3. **Acción humana**

   * El usuario pulsa **“Publicar en LinkedIn”**
   * Solo entonces se hace la llamada al endpoint

4. **Publicación via API**

   * Endpoint: `POST https://api.linkedin.com/v2/ugcPosts`
   * Headers:

     ```http
     Authorization: Bearer <ACCESS_TOKEN>
     X-Restli-Protocol-Version: 2.0.0
     ```
   * Body mínimo:

     ```json
     {
       "author": "urn:li:person:{personId}",
       "lifecycleState": "PUBLISHED",
       "specificContent": {
         "com.linkedin.ugc.ShareContent": {
           "shareCommentary": { "text": "Texto del post" },
           "shareMediaCategory": "NONE"
         }
       },
       "visibility": { "com.linkedin.ugc.MemberNetworkVisibility": "PUBLIC" }
     }
     ```

---

## 3️⃣ Cómo manejar tokens

* **OAuth 2.0**

  * Usuario inicia sesión y concede permisos
  * Intercambias `authorization code` por `access token`
  * Guardas token en **local storage / base local** (si es uso local)
* **Refresh token**

  * LinkedIn expira tokens (normalmente 60 días)
  * Para uso local, se puede regenerar fácilmente pidiendo al usuario que vuelva a autenticar

---

## 4️⃣ Arquitectura recomendada (Clean / Hexagonal)

* **Core**

  * Lógica de drafts: `RSS → Draft`
  * Validación: `Draft → ReadyToPublish`
  * Comando: `PublishPost(draft, userToken)`
* **Adapter LinkedIn**

  * Hace la llamada al endpoint oficial con el token
* **Frontend**

  * Muestra lista de drafts
  * Botón “Publicar”
* **Infra (opcional)**

  * Almacenamiento local (JSON o SQLite)
  * Historial de posts publicados

---

## ✅ Beneficios de esta opción

* Cumples ToS de LinkedIn → app aprobable
* Código abierto y uso local sin riesgos de bans
* Flexible: puedes añadir otras redes (Twitter, Mastodon, etc.)
* Fácil de escalar si luego quieres añadir automatización semi-manual (notificaciones, recordatorios, etc.)

