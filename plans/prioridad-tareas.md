# Feature: prioridad alta, media o baja en las tareas

## Encargo y criterios de aceptación

Encargo literal:

> Añade prioridad alta, media o baja a las tareas. Se elige al crear la tarea, se ve en la lista y se puede cambiar después. Media por defecto y para las tareas que ya existen. Rechaza valores desconocidos. Conserva el resto del comportamiento (crear, validar título, completar/reabrir, editar, eliminar) y el orden actual del listado. No añadas dependencias.

Criterios explícitos (vienen del encargo):

1. Cada tarea tiene una prioridad con uno de estos valores: `alta`, `media` o `baja`.
2. El formulario de crear permite elegir la prioridad.
3. La lista muestra la prioridad de cada tarea.
4. La prioridad de una tarea existente se puede cambiar en la pantalla de edición.
5. Una tarea nueva sin prioridad queda en `media`.
6. Las tareas que ya existen en la base de datos quedan en `media` después de migrar.
7. Un valor desconocido (p. ej. `urgente`, `Alta`, `1`) se rechaza con un error de validación: no se crea ni se modifica la tarea.
8. Crear, validar título, completar/reabrir, editar el título y eliminar siguen funcionando igual.
9. El listado mantiene el orden actual: `created_at asc, id asc`. La prioridad no cambia el orden.
10. No se añaden dependencias a `go.mod`.

Criterios inferidos (decisiones de este plan, a revisar):

- I1. Los valores guardados y enviados por formulario son las cadenas en minúscula `alta`, `media`, `baja`. Las etiquetas visibles son `Alta`, `Media`, `Baja`.
- I2. La comparación es exacta, sin pasar a minúsculas: `Alta` o `ALTA` son desconocidos. Solo se recortan espacios; una cadena vacía (o solo espacios) significa "sin prioridad".
- I3. "Sin prioridad" al **crear** → `media` (criterio 5).
- I4. "Sin prioridad" al **actualizar** (campo ausente o vacío) → se conserva la prioridad actual de la tarea. Así un cliente que solo envía el título se comporta igual que hoy.
- I5. En el formulario de crear, `Media` aparece seleccionada por defecto.
- I6. La lista muestra la prioridad de **todas** las tareas, incluidas las de prioridad media.
- I7. Completar/reabrir (`TasksToggle`) no toca la prioridad.
- I8. El mensaje de error es `La prioridad no es válida.` en la clave `priority`.

## Comportamiento actual y propuesta

Flujo actual (verificado en el código):

- Rutas en `actions/app.go`: `GET /` → `TasksIndex`, `POST /tasks` → `TasksCreate`, `GET /tasks/{task_id}/edit` → `TasksEdit`, `PUT /tasks/{task_id}` → `TasksUpdate`, `PUT /tasks/{task_id}/toggle` → `TasksToggle`, `DELETE /tasks/{task_id}` → `TasksDestroy`.
- `TasksIndex` ordena con `tx.Order("created_at asc, id asc")` y pasa `&models.Task{}` como `task` al formulario de crear.
- `TasksCreate` hace `c.Bind(task)` sobre `models.Task` (campos con etiqueta `form:"-"` no se enlazan), recorta el título y llama a `tx.ValidateAndCreate`. Con errores, vuelve a pintar `tasks/index.plush.html` con 422.
- `TasksUpdate` enlaza a una struct anónima que **solo** tiene `Title`; el resto de campos de la tarea no cambia. `Test_TasksUpdate_PersistsTitleOnly` lo comprueba. Usa `tx.ValidateAndUpdate`; con errores, pinta `tasks/edit.plush.html` con 422.
- `TasksToggle` usa `tx.Update` (sin validar).
- `models.Task` tiene `ID`, `Title`, `Completed`, `CreatedAt`, `UpdatedAt`. `Validate` solo comprueba el título.
- Pop v6.4.1 (`go.mod`) implementa el callback `BeforeValidate(*pop.Connection) error`, y tanto `ValidateAndCreate` como `ValidateAndUpdate` lo llaman antes de `Validate` (`pop/v6@v6.4.1/executors.go`, líneas 116 y 357).
- Migraciones en `migrations/` (embebidas con `//go:embed *.fizz *.sql`): `20260901000000_create_tasks` (fizz) y `20260901000001_seed_tasks` (SQL, inserta 3 tareas). `cmd/app/main.go` ejecuta `models.Migrate()` al arrancar.
- Templates: `templates/tasks/index.plush.html` (lista + formulario de crear) y `templates/tasks/edit.plush.html`. Los dos muestran solo `errors.Get("title")`. El texto de ayuda de la edición dice "Los demás datos de la tarea no cambian."
- `public/assets/app.js` intercepta el formulario de completar: alterna `.task-completed`, lee `.task-title` para el `aria-label` e inserta/quita `<span class="task-status">` antes de `.task-actions`.
- CSS: `.task` es un grid de 3 columnas (`44px minmax(0, 1fr) auto`; en móvil `36px …`). `.task-status` está oculto visualmente (posición absoluta, como `.sr-only`), así que no ocupa celda del grid.

Propuesta (cambio vertical mínimo, sin arquitectura nueva):

- Columna `priority` (string, NOT NULL, default `'media'`) en `tasks` mediante una nueva migración fizz.
- Campo `Priority` en `models.Task`, constantes de los tres valores, `BeforeValidate` que pone `media` si está vacía y regla en `Validate` que rechaza valores desconocidos.
- `TasksIndex` inicializa el formulario de crear con prioridad media. `TasksCreate` no cambia (el `Bind` ya recoge `priority`). `TasksUpdate` añade `Priority` a su struct de entrada y solo la aplica si llega no vacía.
- Select de prioridad en el formulario de crear y en el de editar; etiqueta de prioridad en cada fila de la lista; mensajes de error de prioridad en ambos formularios.
- CSS para la etiqueta y el select. `app.js` **no cambia**.

## Alcance

Incluido:

- Migración `add_column` con default y su `down`.
- Modelo: campo, constantes, valor por defecto, validación y etiqueta visible.
- Acciones `TasksIndex` y `TasksUpdate`.
- Templates `index` y `edit`.
- Estilos en `public/assets/app.css`.
- Tests de modelo y de acciones para el nuevo comportamiento, y un ajuste a un test existente.

Excluido:

- Ordenar o filtrar por prioridad.
- Cambiar la prioridad desde la lista sin pasar por la edición (inline o por JS).
- Cambios en `public/assets/app.js`, en rutas, en `TasksToggle` o en `TasksDestroy`.
- Restricción `CHECK` en la base de datos (la validación vive en el modelo, como la del título).
- Cambiar la migración de seed o el fixture `tres tareas` (el default de la columna ya les da `media`).
- Nuevas dependencias en `go.mod`/`go.sum`.

## Diseño por capas

### Persistencia y migración

Nueva migración (el timestamp debe ser posterior a `20260901000001`):

- `migrations/20261014000000_add_priority_to_tasks.up.fizz`:
  ```
  add_column("tasks", "priority", "string", {"size": 10, "default": "media"})
  ```
  Fizz crea las columnas `NOT NULL` salvo que se indique `"null": true`. Con el default, Postgres rellena con `media` todas las filas existentes, incluidas las 3 del seed.
- `migrations/20261014000000_add_priority_to_tasks.down.fizz`:
  ```
  drop_column("tasks", "priority")
  ```

No hace falta tocar `migrations/embed.go`: el patrón `*.fizz` ya incluye los ficheros nuevos.

### Modelo (`models/task.go`)

```go
// Valores admitidos para la prioridad de una tarea.
const (
	PriorityHigh   = "alta"
	PriorityMedium = "media"
	PriorityLow    = "baja"
)

var priorityLabels = map[string]string{
	PriorityHigh:   "Alta",
	PriorityMedium: "Media",
	PriorityLow:    "Baja",
}
```

- Campo nuevo en `Task`, después de `Completed`:
  `Priority string `json:"priority" db:"priority" form:"priority"``
- `BeforeValidate(tx *pop.Connection) error`: `t.Priority = strings.TrimSpace(t.Priority)`; si queda vacía, `t.Priority = PriorityMedium`. Devuelve `nil`. Así cualquier `ValidateAndCreate` sin prioridad (acción o tests existentes como `models.Task{Title: "..."}`) guarda `media`.
- En `Validate`, después de la regla del título: si `priorityLabels[t.Priority]` no existe, `verrs.Add("priority", "La prioridad no es válida.")`.
- `func (t Task) PriorityLabel() string`: devuelve `priorityLabels[t.Priority]` (cadena vacía si es desconocida). Receptor por valor para que Plush lo pueda llamar sobre los elementos de `tasks` (que son `Task`, no `*Task`) y sobre `task` (`*Task`).

Nota: `BeforeValidate` solo se ejecuta en `ValidateAnd*`. `TasksToggle` usa `tx.Update` sobre una tarea cargada de la BD, que ya tiene prioridad válida; no hay que cambiarlo.

### Acciones (`actions/tasks.go`)

- `TasksIndex`: `c.Set("task", &models.Task{Priority: models.PriorityMedium})` en lugar de `&models.Task{}`. Así el select marca `Media` y no la primera opción.
- `TasksCreate`: sin cambios de código. `c.Bind(task)` enlaza `priority` por la etiqueta `form:"priority"`; `BeforeValidate` pone `media` si falta y `Validate` rechaza valores desconocidos. Con error, ya vuelve a pintar el índice con 422 y no crea nada.
- `TasksUpdate`:
  - Añadir `Priority string `form:"priority" json:"priority"`` a la struct de entrada.
  - Después de asignar el título: `if p := strings.TrimSpace(input.Priority); p != "" { task.Priority = p }`. Vacía o ausente → se conserva la actual (I4).
  - Actualizar el comentario de la función: "actualiza el título y, si llega, la prioridad, conservando los demás datos de la tarea."
  - Con un valor desconocido, `ValidateAndUpdate` devuelve el error y la acción ya responde 422 sin guardar (la transacción de `popmw` no llega a hacer el `UPDATE`).

Rutas: sin cambios.

### Renderizado (Plush)

`templates/tasks/index.plush.html`

- En cada `<li class="task">`, justo después de `<span class="task-title">…</span>` y antes del bloque de `task-status`:
  ```html
  <span class="task-priority task-priority-<%= t.Priority %>"><span class="sr-only">Prioridad: </span><%= t.PriorityLabel() %></span>
  ```
  Queda fuera de `.task-title`, así que el `aria-label` que construye `app.js` a partir de `.task-title` no cambia. `app.js` sigue insertando `.task-status` antes de `.task-actions`; como `.task-status` no ocupa celda del grid, el orden visual no se rompe.
- Formulario de crear, dentro de `.new-task-row`, entre el `<input>` del título y el botón `Añadir`:
  ```html
  <label for="task-priority" class="sr-only">Prioridad</label>
  <select id="task-priority" name="priority"<%= if (len(errors.Get("priority")) > 0) { %> aria-invalid="true" aria-describedby="priority-error"<% } %>>
    <option value="alta"<%= if (task.Priority == "alta") { %> selected<% } %>>Alta</option>
    <option value="media"<%= if (task.Priority == "media") { %> selected<% } %>>Media</option>
    <option value="baja"<%= if (task.Priority == "baja") { %> selected<% } %>>Baja</option>
  </select>
  ```
  Mantener exactamente el formato `value="X" selected` porque los tests lo buscan.
- Errores del formulario de crear: el input del título conserva `autofocus aria-invalid="true" aria-describedby="title-error"`, pero solo cuando `len(errors.Get("title")) > 0` (hoy es `errors.HasAny()`). Debajo de la fila:
  - `<p id="title-error" class="field-error" role="alert">…</p>` solo si hay errores de título (mismo contenido que hoy).
  - `<p id="priority-error" class="field-error" role="alert"><%= for (msg) in errors.Get("priority") { %><%= msg %><% } %></p>` solo si hay errores de prioridad.

`templates/tasks/edit.plush.html`

- Mismo cambio de condición en el input del título: `aria-invalid`/`aria-describedby="title-error"` solo si hay errores de título; si no, `aria-describedby="title-help"`. El `<p id="title-error">` solo si hay errores de título. (Mantiene `Test_TasksUpdate_EmptyTitle`, que busca `aria-describedby="title-error"`.)
- Cambiar el texto de ayuda a: `Hasta 200 caracteres.` (la frase "Los demás datos de la tarea no cambian" deja de ser cierta).
- Después del bloque del título y antes de `.edit-task-actions`, añadir:
  ```html
  <label for="task-priority">Prioridad</label>
  <select id="task-priority" name="priority"<%= if (len(errors.Get("priority")) > 0) { %> aria-invalid="true" aria-describedby="priority-error"<% } %>>
    …mismas tres opciones con `selected` según task.Priority…
  </select>
  <%= if (len(errors.Get("priority")) > 0) { %>
    <p id="priority-error" class="field-error" role="alert"><%= for (msg) in errors.Get("priority") { %><%= msg %><% } %></p>
  <% } %>
  ```

### CSS (`public/assets/app.css`)

- `.task`: `grid-template-columns: 44px minmax(0, 1fr) auto auto;` y en `@media (max-width: 760px)`: `36px minmax(0, 1fr) auto auto;`. La nueva columna es la etiqueta de prioridad.
- `.task-priority`: `align-self: center; padding: 0.1rem 0.5rem; border-radius: 999px; font-size: 0.75rem; font-weight: 600; white-space: nowrap; color: var(--color-muted); background: #f2f2ee;`.
- `.task-priority-alta`: `color: #912a20; background: #fff0eb;` (mismos tonos que ya usa `.field-error`). `.task-priority-baja`: `background: transparent; border: 1px solid var(--color-border);`. `media` usa el estilo base.
- `.task-completed .task-priority { opacity: 0.6; }` para acompañar al título atenuado.
- `select`: mismo aspecto que `input[type="text"]` (`min-height: 44px; padding: 0.6rem 0.75rem; font: inherit; color: var(--color-text); background: #ffffff; border: 1px solid #92928a; border-radius: 6px;`), y `select[aria-invalid="true"]` igual que el input inválido. Además, `.new-task-row select { flex-shrink: 0; width: auto; }`, y en la edición `.edit-task label[for="task-priority"] { margin-top: 1.25rem; }` para separar la prioridad del bloque del título.
- En `@media (max-width: 380px)` la fila ya hace `flex-wrap: wrap`; comprobar a mano que el select y el botón quedan en la segunda línea sin desbordar.

### JavaScript

Sin cambios. Solo hay que comprobar que la etiqueta no rompe el flujo de completar (ver "Tests y validación").

## Archivos relevantes

| Ruta | Responsabilidad |
| --- | --- |
| `migrations/20261014000000_add_priority_to_tasks.up.fizz` (nuevo) | Añade `priority` NOT NULL default `media`. |
| `migrations/20261014000000_add_priority_to_tasks.down.fizz` (nuevo) | Quita la columna. |
| `models/task.go` | Campo `Priority`, constantes, `BeforeValidate`, validación, `PriorityLabel`. |
| `models/task_test.go` | Tests de valor por defecto, valores válidos y desconocidos. |
| `actions/tasks.go` | `TasksIndex` (prioridad inicial) y `TasksUpdate` (aplicar prioridad si llega). |
| `actions/tasks_test.go` | Tests de crear, listar, editar y actualizar con prioridad; ajuste de `Test_TasksUpdate_PersistsTitleOnly`. |
| `templates/tasks/index.plush.html` | Etiqueta en la lista, select y errores en el formulario de crear. |
| `templates/tasks/edit.plush.html` | Select, error y texto de ayuda en la edición. |
| `public/assets/app.css` | Columna del grid, etiqueta y select. |
| `public/assets/app.js` | Sin cambios; referencia para no romper el toggle. |
| `actions/app.go`, `fixtures/tasks.toml`, `migrations/20260901000001_seed_tasks.up.sql` | Sin cambios; referencia. |
| `database.yml` | Sin cambios; `test` apunta a `todo_test` en `DB_HOST` o `127.0.0.1`. |

## Implementación paso a paso

1. Crear las dos migraciones fizz del apartado de persistencia.
2. Modelo: añadir constantes, `priorityLabels`, el campo `Priority`, `BeforeValidate`, la regla en `Validate` y `PriorityLabel`.
3. Preparar la base de test (ver "Tests y validación → Entorno") y ejecutar `GO_ENV=test go test -p 1 ./models/...`. Los tests existentes deben seguir en verde (gracias a `BeforeValidate`).
4. Añadir los tests de modelo nuevos y comprobarlos.
5. Acciones: cambiar `TasksIndex` y `TasksUpdate` como se describe.
6. Templates: `index.plush.html` y `edit.plush.html`.
7. CSS en `app.css`.
8. Añadir los tests de acciones nuevos y ajustar `Test_TasksUpdate_PersistsTitleOnly`.
9. Ejecutar las validaciones completas.
10. Reconstruir la app de Docker y hacer la prueba manual.

## Tests y validación

### Entorno (importante en este Mac)

- En `127.0.0.1:5432` responde el **PostgreSQL 17 de Homebrew** (usuario `postgres`/`postgres`, bases `todo_development` y `todo_test`), no el contenedor de Docker. Los tests y `soda` lanzados desde el host sin `DB_HOST` usan `database.yml` → `127.0.0.1` → Homebrew. Hoy `todo_test` en Homebrew tiene aplicadas `20260901000000` y `20260901000001`.
- El contenedor `db` de `compose.yaml` publica también `*:5432`, pero la app de Docker se conecta por `DB_HOST=db` y aplica sus migraciones sola al arrancar (`models.Migrate()` en `cmd/app/main.go`).
- `soda` no está instalado (en `~/go/bin` solo está `buffalo`). Instalar la misma versión que usa CI (es una herramienta en `~/go/bin`, no una dependencia del proyecto; no cambia `go.mod`):
  ```bash
  go install github.com/gobuffalo/pop/v6/soda@v6.4.1
  ```
- Si el shell no tiene `~/go/bin` en el `PATH`: `export PATH="$PATH:$HOME/go/bin"` (y para Docker `export PATH="$PATH:/Applications/Docker.app/Contents/Resources/bin"`).
- Aplicar la migración nueva a la base de test (desde la raíz del repo):
  ```bash
  soda migrate -e test
  ```
  Comprobar con `PGPASSWORD=postgres psql -h 127.0.0.1 -U postgres -d todo_test -c '\d tasks'` que existe `priority` con default `'media'::character varying` y `not null`.
- Los tests deben ejecutarse con `GO_ENV=test` (como en CI); sin esa variable `models.DB` se conecta a `todo_development`.

### Tests nuevos de modelo (`models/task_test.go`)

- `Test_Task_Priority_DefaultsToMedia`: `ValidateAndCreate(&Task{Title: "…"})` sin errores; `task.Priority == PriorityMedium`; tras `Find`, sigue siendo `media`.
- `Test_Task_Priority_KnownValues_Valid`: para `alta`, `media`, `baja`, crear sin errores y recargar con el mismo valor.
- `Test_Task_Priority_Unknown_Invalid`: `Priority: "urgente"` → `verrs.Get("priority")` contiene `La prioridad no es válida.`; `Count == 0`. Repetir con `"Alta"` para fijar la comparación exacta (I2).
- `Test_Task_PriorityLabel`: `Task{Priority: "baja"}.PriorityLabel() == "Baja"`.

### Tests nuevos o ajustados de acciones (`actions/tasks_test.go`)

- Ajuste: en `Test_TasksUpdate_PersistsTitleOnly`, añadir `as.Equal(models.PriorityMedium, task.Priority)` tras recargar (una actualización sin prioridad la conserva).
- `Test_TasksIndex_NewTaskFormDefaultsToMedia`: `GET /` contiene `value="media" selected` y no contiene `value="alta" selected`.
- `Test_TasksIndex_ShowsPriorityKeepingOrder`: crear con `ValidateAndCreate` tres tareas en este orden: "Primera" `baja`, "Segunda" `alta`, "Tercera" sin prioridad. El cuerpo contiene `task-priority-baja`, `task-priority-alta`, `task-priority-media` y los textos `Baja`, `Alta`, `Media`; y `strings.Index(body, "Primera") < strings.Index(body, "Segunda") < strings.Index(body, "Tercera")` (el orden no depende de la prioridad).
- `Test_TasksCreate_WithPriority`: `POST /tasks` con `url.Values{"title": {"Con prioridad"}, "priority": {"alta"}}` → 303; la tarea guardada tiene `alta`.
- En `Test_TasksCreate_Valid` (envía `models.Task{Title: …}` sin prioridad), añadir `as.Equal(models.PriorityMedium, task.Priority)`.
- `Test_TasksCreate_UnknownPriority_Invalid`: `priority=urgente` → 422, cuerpo contiene `La prioridad no es válida.` y `aria-describedby="priority-error"`; `Count == 0`.
- `Test_TasksEdit_PreselectsPriority`: tarea con `baja` → `GET /tasks/{id}/edit` contiene `value="baja" selected`.
- `Test_TasksUpdate_ChangesPriority`: tarea completada con `media`; `PUT` con `title` y `priority=alta` → 303; recargada: título nuevo, `alta`, sigue completada.
- `Test_TasksUpdate_WithoutPriority_KeepsCurrent`: tarea con `baja`; `PUT` solo con `title` → 303; sigue en `baja`. Repetir con `priority=""` → sigue en `baja`.
- `Test_TasksUpdate_UnknownPriority_Invalid`: tarea con `baja`; `PUT` con `title=Otro` y `priority=urgente` → 422 con `La prioridad no es válida.`; recargada: título y prioridad sin cambios.
- `Test_TasksToggle_KeepsPriority`: tarea con `alta`; toggle → sigue en `alta`.

Los tests existentes de crear, título, toggle, edición y eliminar deben seguir pasando sin cambios (salvo los dos ajustes indicados).

### Validaciones automáticas

```bash
gofmt -l .
go vet ./...
GO_ENV=test go test -p 1 ./...
go build ./...
git diff --stat go.mod go.sum
```

`gofmt -l .` no debe listar nada y `git diff` de `go.mod`/`go.sum` debe salir vacío.

### Prueba manual (Docker en localhost:3000)

1. `docker compose up -d --build app` (la imagen compila el binario; al arrancar aplica la migración en el Postgres del contenedor).
2. Las tareas que ya existían muestran `Media`.
3. Crear una tarea sin tocar el select → `Media`; crear otra con `Alta` → `Alta`, y aparece al final de la lista (orden por creación).
4. Editar una tarea, cambiar a `Baja` → se ve `Baja` en la lista; el estado completada/pendiente no cambia.
5. Completar y reabrir una tarea con el botón: la etiqueta sigue en su sitio, el título se atenúa como antes, y al recargar la página todo se ve igual.
6. Con las herramientas del navegador, cambiar el `value` de una opción a `urgente` y enviar → mensaje `La prioridad no es válida.` y no se guarda.
7. Ver la lista a 375 px de ancho: la etiqueta, el título y las acciones caben sin scroll horizontal; el formulario de crear se reparte en dos líneas sin desbordar.

## Riesgos y recuperación

- **Tests existentes que crean `Task{Title: …}`**: si el default solo se pusiera en la acción, fallarían por "prioridad no válida". Mitigación: `BeforeValidate` en el modelo. Si aun así fallan, comprobar que el método tiene receptor puntero (`func (t *Task) BeforeValidate(...)`) y la firma exacta `BeforeValidate(*pop.Connection) error`.
- **Base de test sin migrar**: los tests fallarían con `column "priority" does not exist`. Recuperación: `soda migrate -e test` contra el Postgres de Homebrew (sin `DB_HOST`). No confundirlo con el Postgres de Docker.
- **Ejecutar tests sin `GO_ENV=test`**: usarían y truncarían `todo_development` de Homebrew. Usar siempre `GO_ENV=test`.
- **Re-pintado tras un valor desconocido**: la tarea lleva el valor inválido, ninguna opción sale marcada y el navegador muestra la primera (`Alta`). Solo ocurre si alguien manipula el formulario; se acepta. Alternativa si se quiere pulir: no mostrar el valor inválido y marcar `media` (no incluida en este plan).
- **Grid de la fila**: si se olvida cambiar `grid-template-columns`, la etiqueta pasa a una segunda fila del grid. Revisar en la prueba manual, también a ancho móvil.
- **`app.js`**: inserta `.task-status` antes de `.task-actions` y lee `.task-title`. La etiqueta está fuera de `.task-title` y `.task-status` no ocupa celda; no debería afectar. Si se ve algo raro, revisar el orden de los hijos de `.task`, no cambiar el JS.
- **Revertir**: `soda migrate down -e test` (y en Docker, volver a la imagen anterior y ejecutar el `down` en su base) quita la columna; después revertir los cambios de código con git. La migración `down` pierde las prioridades guardadas.
