# Feature: eliminar todas las tareas completadas de una vez

## Encargo y criterios de aceptación

Origen: issue #4 — https://github.com/GoyoBjS/desarrollo-agentico-todo-03/issues/4 ("Eliminar todas las tareas completadas de una vez").

> Cuando la lista acumula tareas completadas, hoy hay que borrarlas una a una. Queremos un botón para eliminar **todas las tareas completadas** de una vez.

Criterios explícitos (vienen de la issue):

1. Bajo la lista aparece un botón **"Eliminar completadas"**, solo si hay al menos una tarea completada.
2. Antes de borrar se pide confirmación con el mismo mecanismo `data-confirm` que el botón de eliminar de cada tarea: "¿Seguro que quieres eliminar las tareas completadas?".
3. Al confirmar se eliminan **solo** las tareas completadas; las pendientes no se tocan.
4. Después vuelve a la lista con el mensaje flash "Tareas completadas eliminadas.".
5. Si no hay tareas completadas, la petición no borra nada y vuelve a la lista sin error.
6. Las tareas que quedan conservan su orden actual (`created_at asc, id asc`).
7. Se conserva el resto del comportamiento: crear, validar título, completar/reabrir, editar, eliminar una tarea, prioridades y contador de pendientes.
8. La ruta usa un método que modifica datos (no GET) y el formulario envía el token CSRF.
9. No se añaden dependencias.
10. Tests de la acción: borra solo las completadas, no hace nada si no hay ninguna, y el botón aparece o no según haya completadas.

Criterios inferidos (decisiones de este plan, a revisar en el PR):

- I1. Ruta `DELETE /tasks/completed` (con `_method=DELETE` en un formulario POST, igual que eliminar una tarea), acción `TasksDestroyCompleted`.
- I2. Si no hay completadas (criterio 5), redirige a `/` con 303 y **sin** flash: no se muestra "Tareas completadas eliminadas." porque sería falso.
- I3. "Aparece solo si hay al menos una completada": cuando la lista tiene tareas, el formulario se renderiza siempre bajo la lista, pero con el atributo `hidden` si no hay completadas. Así `app.js` puede mostrarlo u ocultarlo al completar/reabrir sin recargar (hoy ya sincroniza el contador de pendientes del mismo modo). Con la lista vacía no se renderiza.
- I4. El orden (criterio 6) no requiere cambios: `TasksIndex` ya ordena por `created_at asc, id asc` y el borrado no modifica las filas restantes.

## Comportamiento actual y propuesta

Actual (verificado en el código):

- `actions/app.go` registra `GET /`, `POST /tasks`, `GET /tasks/{task_id}/edit`, `PUT /tasks/{task_id}`, `PUT /tasks/{task_id}/toggle`, `DELETE /tasks/{task_id}`. Middleware `csrf.New` y `popmw.Transaction(models.DB)` (cada petición tiene `tx`).
- `TasksDestroy` busca la tarea, la destruye, añade flash `success` "Tarea eliminada." y redirige 303 a `/`.
- `templates/tasks/index.plush.html` pinta la lista en `<ul class="task-list">` dentro de `<section>`; cada fila tiene un formulario de borrado con `data-confirm`, `authenticity_token` y `_method=DELETE`.
- `public/assets/app.js` pide confirmación para cualquier `form[data-confirm]` e intercepta `form[data-task-toggle]`: alterna `.task-completed` en el `<li>` y recalcula el contador de pendientes.
- `models.Tasks` tiene `PendingCount()`.
- No existe forma de borrar varias tareas a la vez.

Propuesta (cambio vertical mínimo):

- `models.Tasks.CompletedCount()` (simétrico a `PendingCount`).
- Acción `TasksDestroyCompleted` que borra con `tx.Where("completed = ?", true).Delete(&models.Task{})`.
- Ruta `app.DELETE("/tasks/completed", TasksDestroyCompleted)` registrada **antes** de `app.DELETE("/tasks/{task_id}", TasksDestroy)`, para que `completed` no se interprete como `task_id`.
- Formulario "Eliminar completadas" bajo la `<ul>` en `index.plush.html`.
- `app.js`: tras alternar una tarea, actualiza `hidden` del formulario según queden `.task-completed`.
- CSS mínimo para colocar el botón y respetar `[hidden]`.

## Alcance

Incluido:

- Modelo: `CompletedCount()`.
- Acción y ruta nuevas.
- Template de la lista, `app.js` y `app.css`.
- Tests de acción y de modelo.

Excluido:

- Migraciones (no cambia el esquema).
- Deshacer el borrado, borrado parcial o selección múltiple.
- Cambios en la pantalla de edición, en las prioridades o en el contador de la cabecera.
- Cambios en el comportamiento existente de `app.js` para completar/reabrir (incluida su llamada `fetch(form.action)`).
- Dependencias nuevas.

## Diseño por capas

### Modelo (`models/task.go`)

Añadir junto a `PendingCount`:

```go
// CompletedCount devuelve cuántas tareas de la colección están completadas.
func (t Tasks) CompletedCount() int {
	return len(t) - t.PendingCount()
}
```

### Acción y ruta

`actions/tasks.go`, después de `TasksDestroy`:

```go
// TasksDestroyCompleted elimina todas las tareas completadas y deja las pendientes.
func TasksDestroyCompleted(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)

	count, err := tx.Where("completed = ?", true).Count(&models.Task{})
	if err != nil {
		return err
	}
	if count == 0 {
		return c.Redirect(http.StatusSeeOther, "/")
	}

	if err := tx.Where("completed = ?", true).Delete(&models.Task{}); err != nil {
		return err
	}

	c.Flash().Add("success", "Tareas completadas eliminadas.")
	return c.Redirect(http.StatusSeeOther, "/")
}
```

Usa una query nueva en cada llamada (`tx.Where(...)`); no reutilices el objeto `*pop.Query` entre `Count` y `Delete`.

`actions/app.go`: insertar `app.DELETE("/tasks/completed", TasksDestroyCompleted)` en la línea anterior a `app.DELETE("/tasks/{task_id}", TasksDestroy)`.

### Renderizado (`templates/tasks/index.plush.html`)

Dentro de la rama `else` de la sección de la lista, justo después de `</ul>`:

```html
    <form action="/tasks/completed" method="POST" class="clear-completed" data-clear-completed data-confirm="¿Seguro que quieres eliminar las tareas completadas?"<%= if (tasks.CompletedCount() == 0) { %> hidden<% } %>>
      <input type="hidden" name="authenticity_token" value="<%= authenticity_token %>">
      <input type="hidden" name="_method" value="DELETE">
      <button type="submit" class="button">Eliminar completadas</button>
    </form>
```

Mantén este marcado exacto (los tests buscan `data-confirm="¿Seguro que quieres eliminar las tareas completadas?">` y `data-confirm="¿Seguro que quieres eliminar las tareas completadas?" hidden>`).

### JavaScript (`public/assets/app.js`)

En el manejador de `submit`, tras el bloque que actualiza el contador de pendientes y antes de `fetch(form.action);`:

```js
  var clearCompleted = document.querySelector("[data-clear-completed]");
  if (clearCompleted) {
    clearCompleted.hidden = document.querySelectorAll(".task.task-completed").length === 0;
  }
```

No cambies el resto del archivo. La confirmación ya funciona por `data-confirm` sin cambios.

### CSS (`public/assets/app.css`)

Después del bloque `.task-list`:

```css
.clear-completed {
  display: flex;
  justify-content: flex-end;
  margin-top: 0.75rem;
}

.clear-completed[hidden] {
  display: none;
}
```

## Archivos relevantes

| Ruta | Responsabilidad |
| --- | --- |
| `models/task.go` | `Tasks.CompletedCount()` |
| `models/task_test.go` | Test de `CompletedCount` |
| `actions/app.go` | Registro de `DELETE /tasks/completed` antes de `DELETE /tasks/{task_id}` |
| `actions/tasks.go` | Acción `TasksDestroyCompleted` |
| `actions/tasks_test.go` | Tests de la acción y del botón |
| `templates/tasks/index.plush.html` | Formulario "Eliminar completadas" bajo la lista |
| `public/assets/app.js` | Mostrar/ocultar el formulario al completar/reabrir sin recargar |
| `public/assets/app.css` | Colocación del botón y `[hidden]` |
| `fixtures/tasks.toml` | Escenario "tres tareas" (2 pendientes, 1 completada); solo lectura |

## Implementación paso a paso

1. Añadir `CompletedCount()` en `models/task.go` y su test en `models/task_test.go`:
   ```go
   func (ms *ModelSuite) Test_Tasks_CompletedCount() {
       ms.Equal(0, Tasks{}.CompletedCount())
       ms.Equal(1, Tasks{{Completed: false}, {Completed: true}, {Completed: false}}.CompletedCount())
   }
   ```
2. Añadir `TasksDestroyCompleted` en `actions/tasks.go`.
3. Registrar la ruta en `actions/app.go` antes de `DELETE /tasks/{task_id}`.
4. Añadir el formulario en `templates/tasks/index.plush.html`.
5. Añadir las líneas de `app.js` y el CSS.
6. Añadir los tests de acción (sección siguiente) al final de `actions/tasks_test.go`.
7. Ejecutar las validaciones.

## Tests y validación

Tests nuevos en `actions/tasks_test.go` (usa el helper existente `taskWithPriority(title, priority, completed)`):

- `Test_TasksDestroyCompleted_DeletesOnlyCompleted`: crea "Primera" (pendiente), "Segunda" (completada), "Tercera" (pendiente), "Cuarta" (completada). `as.HTML("/tasks/completed").Delete()` → 303 y `Location` `/`. En BD quedan 2 tareas, ninguna completada, títulos "Primera" y "Tercera". Después `GET /` → 200, contiene "Primera" antes que "Tercera" (`strings.Index`), no contiene "Segunda" ni "Cuarta".
- `Test_TasksDestroyCompleted_ShowsFlash`: con una completada, tras el DELETE hacer `as.HTML("/").Get()` y comprobar "Tareas completadas eliminadas." (el flash se conserva entre peticiones de la suite como en el resto de Buffalo; si no aparece, comprobar al menos la redirección y anotarlo como desviación).
- `Test_TasksDestroyCompleted_NoneCompleted`: dos pendientes; DELETE → 303 a `/`; siguen 2 tareas.
- `Test_TasksDestroyCompleted_EmptyList`: sin tareas; DELETE → 303 a `/`; 0 tareas.
- `Test_TasksIndex_ClearCompletedVisible`: `as.LoadFixture("tres tareas")`; `GET /` contiene `Eliminar completadas` y `data-confirm="¿Seguro que quieres eliminar las tareas completadas?">` (sin `hidden`).
- `Test_TasksIndex_ClearCompletedHiddenWithoutCompleted`: una pendiente; `GET /` contiene `data-confirm="¿Seguro que quieres eliminar las tareas completadas?" hidden>`.
- `Test_TasksIndex_ClearCompletedAbsentWhenEmpty`: sin tareas; `GET /` no contiene `data-clear-completed`.
- `Test_TasksDestroy` y `Test_TasksDestroy_NotFound` existentes deben seguir pasando (verifican que la ruta nueva no rompe la de una tarea).

Validación (los tests de aplicación necesitan PostgreSQL y migraciones de test: `PATH="$PATH:$HOME/go/bin" soda migrate -e test`):

```bash
gofmt -l .
go vet ./...
GO_ENV=test go test -p 1 ./...
go build ./...
```

`gofmt -l .` debe salir vacío; los demás, sin errores.

## Riesgos y recuperación

- **Colisión de rutas**: si `DELETE /tasks/completed` se registra después de `DELETE /tasks/{task_id}`, `TasksDestroy` buscaría el id `completed` y respondería 404. Lo detectan los tests de la acción; se corrige moviendo la línea.
- **Borrado masivo**: la acción borra en una sola sentencia dentro de la transacción del middleware; si falla, la transacción hace rollback. Solo filtra por `completed = true`.
- **Estado del botón tras completar sin recargar**: depende del cambio en `app.js`; sin JavaScript el estado se actualiza al recargar, igual que el resto de la página.
- **`migrations/schema.sql`**: `soda migrate` puede generarlo; no forma parte del cambio y debe borrarse.
- Recuperación: revertir el commit; no hay cambios de esquema ni de datos persistentes salvo las tareas borradas por el usuario.
