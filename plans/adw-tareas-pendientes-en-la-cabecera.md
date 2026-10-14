# Feature: número de tareas pendientes en la cabecera

## Encargo y criterios de aceptación

Encargo literal (recibido por `/adw`, sin issue):

> Muestra en la cabecera de la lista cuántas tareas quedan pendientes, además del total actual. Conserva el resto del comportamiento. No añadas dependencias.

Criterios explícitos:

1. La cabecera de `/` sigue mostrando el total actual: `N tarea(s) en tu lista`.
2. Además muestra cuántas tareas están pendientes, es decir, con `completed = false`.
3. No cambia nada más: lista, orden, formularios, completar/reabrir, editar y eliminar.
4. No se añaden dependencias.

Criterios inferidos (decisiones del plan):

- I1. **Formato:** `3 tareas en tu lista · 2 pendientes`, en singular cuando toca: `1 pendiente`. Se añade dentro del mismo `<p class="list-count">`.
- I2. **Lista vacía:** se muestra solo `0 tareas en tu lista`, sin la parte de pendientes (el estado vacío ya explica que no hay tareas).
- I3. **Todas completadas:** `· 0 pendientes`.
- I4. Completar/reabrir se hace en el navegador con `public/assets/app.js`, sin recargar. **El contador de pendientes debe actualizarse también ahí**; si no, mostraría un número desfasado hasta recargar. Esto forma parte de "conservar el comportamiento": la página tiene que seguir siendo coherente después de completar una tarea.

## Comportamiento actual y propuesta

Actual (verificado):

- `templates/tasks/index.plush.html`, líneas 3-5: `<p class="list-count"><%= len(tasks) %> tarea(s) en tu lista</p>`.
- `TasksIndex` y la rama 422 de `TasksCreate` pintan esa misma plantilla con `tasks` (`models.Tasks`), ordenadas por `created_at asc, id asc`.
- `public/assets/app.js` intercepta el formulario `[data-task-toggle]`:
  - alterna la clase `.task-completed` en el `<li>`;
  - actualiza las etiquetas del botón y el `<span class="task-status">`;
  - hace `fetch(form.action)`;
  - no recarga la página.
- `templates/application.plush.html` carga `app.js?v=2` y `app.css?v=3`. El commit `d8dc18d` subió esas versiones al cambiar los assets.
- Ningún test comprueba hoy el texto de la cabecera.

Propuesta (cambio vertical mínimo):

- **Modelo:** método `func (t Tasks) PendingCount() int`, para que la plantilla lo use directamente. Así no hay que cambiar ninguna acción: las dos que pintan el índice ya pasan `tasks`.
- **Plantilla:** añadir la parte de pendientes con un `<span data-pending-count>` para el número y un `<span data-pending-label>` para "pendiente"/"pendientes".
- **JS:** después de alternar la tarea, recalcular el número con `.task:not(.task-completed)` y actualizar los dos spans, si existen.
- **Caché:** subir `app.js?v=2` → `?v=3`, siguiendo el patrón del repositorio.

## Alcance

Incluido: `models/task.go`, `models/task_test.go`, `templates/tasks/index.plush.html`, `public/assets/app.js`, `templates/application.plush.html` (solo la versión de `app.js`) y `actions/tasks_test.go`.

Excluido:
- Migraciones, rutas, acciones y CSS. `.list-count` ya da el estilo adecuado.
- Mostrar el número de completadas.
- Filtros.
- Tests de JavaScript (el repositorio no tiene infraestructura para ellos).

## Diseño por capas

### Modelo (`models/task.go`)

```go
// PendingCount devuelve cuántas tareas de la colección no están completadas.
func (t Tasks) PendingCount() int {
	count := 0
	for _, task := range t {
		if !task.Completed {
			count++
		}
	}
	return count
}
```

### Renderizado (`templates/tasks/index.plush.html`)

Sustituir el contenido de `<p class="list-count">` por:

```html
  <p class="list-count">
    <%= len(tasks) %> <%= if (len(tasks) == 1) { %>tarea en tu lista<% } else { %>tareas en tu lista<% } %><%= if (len(tasks) > 0) { %> · <span data-pending-count><%= tasks.PendingCount() %></span> <span data-pending-label><%= if (tasks.PendingCount() == 1) { %>pendiente<% } else { %>pendientes<% } %></span><% } %>
  </p>
```

Mantener exactamente `<span data-pending-count>N</span> <span data-pending-label>…</span>`, porque los tests lo buscan.

### JavaScript (`public/assets/app.js`)

Dentro del manejador del toggle, justo antes de `fetch(form.action);`:

```js
  var pendingCount = document.querySelector("[data-pending-count]");
  var pendingLabel = document.querySelector("[data-pending-label]");
  if (pendingCount && pendingLabel) {
    var pending = document.querySelectorAll(".task:not(.task-completed)").length;
    pendingCount.textContent = String(pending);
    pendingLabel.textContent = pending === 1 ? "pendiente" : "pendientes";
  }
```

En `templates/application.plush.html`: `/assets/app.js?v=2` → `/assets/app.js?v=3`.

## Archivos relevantes

| Ruta | Responsabilidad |
| --- | --- |
| `models/task.go` | `Tasks.PendingCount()`. |
| `models/task_test.go` | Test unitario de `PendingCount`. |
| `templates/tasks/index.plush.html` | Texto de la cabecera. |
| `public/assets/app.js` | Actualiza el contador al completar o reabrir sin recargar. |
| `templates/application.plush.html` | Versión de `app.js` para invalidar la caché del navegador. |
| `actions/tasks_test.go` | Tests de la cabecera. |
| `actions/tasks.go` | Sin cambios; `TasksIndex` y `TasksCreate` (422) ya pasan `tasks`. |
| `fixtures/tasks.toml` | Escenario `tres tareas`: 2 pendientes y 1 completada. Sin cambios. |

## Implementación paso a paso

1. Añadir `PendingCount` en `models/task.go` y su test.
2. Cambiar la cabecera de `index.plush.html`.
3. Añadir la actualización del contador en `app.js` y subir `?v=3` en `application.plush.html`.
4. Añadir los tests de acciones.
5. Ejecutar las validaciones.

## Tests y validación

Tests nuevos:

- `models/task_test.go` → `Test_Tasks_PendingCount`: `Tasks{}` → 0; `Tasks{{Completed: false}, {Completed: true}, {Completed: false}}` → 2. No usa la base de datos, pero va en `ModelSuite` como el resto.
- `actions/tasks_test.go`:
  - `Test_TasksIndex_ShowsPendingCount`: `LoadFixture("tres tareas")`. El cuerpo contiene `3 tareas en tu lista` y `<span data-pending-count>2</span> <span data-pending-label>pendientes</span>`.
  - `Test_TasksIndex_PendingCountSingular`: crear una tarea pendiente y otra completada con `ValidateAndCreate`. Contiene `<span data-pending-count>1</span> <span data-pending-label>pendiente</span>`.
  - `Test_TasksIndex_EmptyState_NoPendingCount`: sin tareas. No contiene `data-pending-count`.

Preparación del entorno: tests con `GO_ENV=test`. La base `todo_test` debe estar migrada (`PATH="$PATH:$HOME/go/bin" soda migrate -e test`); este cambio no añade migraciones.

```bash
gofmt -l .
go vet ./...
GO_ENV=test go test -count=1 -p 1 ./...
go build ./...
git diff --stat go.mod go.sum
```

Prueba manual: con la app levantada, completar y reabrir una tarea y comprobar que el contador cambia sin recargar y coincide con lo que se ve tras recargar.

## Riesgos y recuperación

- **Contador desfasado en el navegador** si `app.js` no se actualiza o el navegador usa la versión en caché. Lo mitigan el paso 3 y `?v=3`.
- **Si el `fetch` del toggle falla,** la vista (clase y contador) queda cambiada, pero el servidor no. Este comportamiento ya existe para `.task-completed` y no se cambia aquí.
- **Revertir:** `git revert` del commit; no hay migraciones.
