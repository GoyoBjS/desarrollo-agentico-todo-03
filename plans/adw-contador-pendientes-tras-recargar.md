# Bug: completar una tarea desde la lista no se guarda y, al recargar, el contador vuelve a contarla como pendiente

## Encargo y resultado esperado

Encargo literal (recibido por `/adw`, sin issue):

> Bug: al completar una tarea y recargar la página, el contador de pendientes no coincide con las tareas pendientes. Pasos: 1) crear dos tareas, 2) completar una, 3) recargar. Esperado: '1 pendiente'. Actual: '2 pendientes'. Conserva el resto del comportamiento.

Resultado esperado:

1. Completar o reabrir una tarea con el botón de la lista **se guarda en el servidor**.
2. Después de recargar, la tarea sigue completada o reabierta y la cabecera muestra el número correcto. En el ejemplo del encargo: `2 tareas en tu lista · 1 pendiente`.
3. La respuesta visual inmediata, sin recarga, se mantiene: la clase, el texto y el contador cambian al instante, como hasta ahora.
4. Conserva el resto: rutas, acciones, formularios sin JS, editar, eliminar y orden.

## Evidencia y causa raíz

Hechos observados:

- `public/assets/app.js`, línea 42: el manejador del toggle hace `event.preventDefault()`, cambia el DOM y termina con `fetch(form.action);`. Sin opciones, `fetch` hace una petición **GET** sin cuerpo. La línea viene del commit `5fdd6c3 feat(ui): improve task completion feedback`.
- `actions/app.go`, línea 55: solo existe `app.PUT("/tasks/{task_id}/toggle", TasksToggle)`. No hay ruta GET.
- Reproducción contra la app en marcha (Docker, `localhost:3000`):
  - `curl -s -o /dev/null -w "%{http_code}" http://localhost:3000/tasks/095bfb69-202a-4b25-a23b-4d65c6f0b3c7/toggle` → **404**;
  - en la base, la tarea sigue con `completed = f`.
- El formulario del toggle (`templates/tasks/index.plush.html`) ya incluye `authenticity_token` y `_method=PUT`. Buffalo v1.1.4 (`method_override.go`) convierte un POST con `_method` en el método indicado. El CSRF (`gobuffalo/middleware@v1.0.0/csrf`) acepta el token en el campo `authenticity_token` del formulario (`PostFormValue`).
- El contador de la cabecera (`Tasks.PendingCount()`) se calcula en el servidor a partir de lo guardado. Es correcto: el problema es que el dato no llega a guardarse.

Causa raíz: el toggle por JavaScript sustituye el envío del formulario (`PUT` mediante `POST` + `_method`) por un `GET` que no tiene ruta. El cambio solo existe en el DOM y se pierde al recargar. El "contador incorrecto" del encargo es una consecuencia: tras recargar, la tarea vuelve a salir pendiente, y el contador (2) coincide con lo que hay guardado.

Incertidumbres:

- Sin JavaScript, el formulario se envía de forma normal y funciona; el fallo solo afecta al camino con JS.
- No hay tests de JavaScript en el repositorio. La regresión se cubre con tests Go que comprueban el contrato (ver Tests) y con una verificación manual en el navegador.

Reproducción mínima en el navegador: con la app en marcha, completar una tarea con el botón circular, recargar y ver que vuelve a estar pendiente. En la consola de red aparece `GET /tasks/<id>/toggle 404`.

## Alcance

Incluido:

- En `public/assets/app.js`, sustituir `fetch(form.action);` por un envío equivalente al del formulario: `POST` con el cuerpo del formulario codificado como `application/x-www-form-urlencoded`, que incluye `_method=PUT` y `authenticity_token`.
- Si la respuesta no es correcta o hay un error de red, recargar la página para que la vista refleje lo guardado y no quede desfasada en silencio.
- Subir `app.js?v=3` → `?v=4` en `templates/application.plush.html`, siguiendo el patrón del repositorio.
- Tests Go de regresión en `actions/tasks_test.go`.

Excluido:

- Cambios en rutas, acciones (`TasksToggle`), modelo, plantillas de la lista o CSS.
- Añadir una ruta GET para el toggle: haría que un GET modificara datos.
- Infraestructura de tests de JavaScript.
- Revertir el DOM de forma optimista; la recarga ante error es suficiente.

## Archivos relevantes

| Ruta | Responsabilidad |
| --- | --- |
| `public/assets/app.js` | Toggle en el navegador; contiene el `fetch` erróneo. |
| `templates/application.plush.html` | Versión de `app.js` (`?v=3`) para invalidar la caché. |
| `templates/tasks/index.plush.html` | Formulario del toggle con `_method=PUT` y `authenticity_token`. Sin cambios. |
| `actions/app.go` | Ruta `PUT /tasks/{task_id}/toggle`. Sin cambios. |
| `actions/tasks.go` | `TasksToggle`, que redirige con 303 a `/`. Sin cambios. |
| `actions/tasks_test.go` | Tests de regresión. |

## Implementación paso a paso

1. En `public/assets/app.js`, sustituir la línea `fetch(form.action);` por:

   ```js
   fetch(form.action, {
     method: "POST",
     body: new URLSearchParams(new FormData(form)),
   }).then(function (response) {
     if (!response.ok) {
       window.location.reload();
     }
   }).catch(function () {
     window.location.reload();
   });
   ```

   `fetch` envía las cookies de sesión en peticiones del mismo origen y sigue el 303 hacia `GET /`, que responde 200, así que `response.ok` es `true` cuando todo va bien.
2. En `templates/application.plush.html`: `/assets/app.js?v=3` → `/assets/app.js?v=4`.
3. Añadir los tests de la sección siguiente.
4. Ejecutar las validaciones.

## Tests y validación

Tests nuevos en `actions/tasks_test.go`:

- `Test_TasksToggle_MethodOverridePost`: crear una tarea pendiente; `as.HTML("/tasks/%s/toggle", task.ID).Post(url.Values{"_method": {"PUT"}})` → 303; recargada, `Completed == true`. Demuestra que el envío que hará el JS (POST + `_method=PUT`) completa la tarea. En el entorno de test el CSRF no se aplica, igual que en el resto de tests de acciones.
- `Test_TasksToggle_GetDoesNotToggle`: crear una tarea pendiente; `as.HTML("/tasks/%s/toggle", task.ID).Get()` → 404; la tarea sigue pendiente. Deja constancia de que la petición que hacía el JS no tiene ruta.
- `Test_AppJS_TogglePersistsWithPost`: `as.HTML("/assets/app.js").Get()` → 200. El cuerpo contiene `method: "POST"` y `new FormData(form)`, y **no** contiene `fetch(form.action);`. **Falla antes del arreglo**: es el test de regresión del JS.

Preparación: `GO_ENV=test` y la base `todo_test` migrada (`PATH="$PATH:$HOME/go/bin" soda migrate -e test`). No hay migraciones nuevas.

```bash
gofmt -l .
go vet ./...
GO_ENV=test go test -count=1 -p 1 ./...
go build ./...
node --check public/assets/app.js
git diff --stat go.mod go.sum
```

Verificación manual (opcional; cambia datos de desarrollo de forma reversible):

1. `docker compose up -d --build app`.
2. Completar una tarea y recargar: sigue completada y el contador es correcto.
3. Reabrirla para dejar los datos como estaban.

## Riesgos y recuperación

- **Formato del cuerpo.** Con `URLSearchParams` el cuerpo va como `application/x-www-form-urlencoded`, que `PostFormValue` lee sin necesidad de multipart. Si el CSRF rechazara la petición (403), la recarga ante error mostraría el estado real y se vería en la verificación manual.
- **Caché del navegador:** lo mitiga `?v=4`.
- **Test de cadena sobre `app.js`:** es frágil ante cambios de redacción. Si se reescribe el JS, hay que actualizarlo junto con el código.
- **Revertir:** `git revert` del commit; no hay migraciones.
