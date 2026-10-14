# Chore: permisos de proyecto para que el trigger ejecute `/adw` con `claude -p`

## Encargo y resultado esperado

Encargo:

> Crea `.claude/settings.json` con los permisos mínimos para que el trigger de `adws/` pueda ejecutar `/adw` en modo `claude -p` sin intervención humana: editar archivos del proyecto, git (incluido push de ramas adw/…), gh pr/issue, go, gofmt, soda y node --check. Debe denegar push a main, force-push, merges de PR y lo peligroso. No toques el código del trigger ni la skill. No añadas dependencias.

Resultado esperado:

1. Existe `.claude/settings.json`, versionado, con `permissions.allow` y `permissions.deny`.
2. Una ejecución `claude -p "/adw <issue_url> --remote origin --base main"`, lanzada por el trigger desde la raíz del repo, puede hacer **todos** los pasos de `/adw` sin que se deniegue ningún permiso. Eso incluye las skills que invoca (`/feature`, `/bug`, `/chore`, `/implement`) y sus validaciones.
3. Quedan **denegados** de forma explícita: push a `main`, force-push, `gh pr merge`, editar los propios ajustes de permisos, leer o editar `.env`, `sudo`, `rm -rf`, y descargar o ejecutar código remoto (`curl`, `wget`).
4. Lo que no esté en `allow` se sigue denegando en `-p`, o preguntando en una sesión interactiva, como hasta ahora.

## Estado actual y restricciones

Verificado:

- **Rama `chore/adw-permisos`**, creada desde `main` (`0154185`, ya en `origin/main`). No existe `.claude/settings.json` ni `.claude/settings.local.json`. `.gitignore` no ignora ninguno de los dos.
- **El trigger** (`adws/trigger.go`) ejecuta `claude -p <prompt>` desde la raíz del repo, con stdin cerrado y **sin flags**. Los permisos solo pueden venir de los archivos de ajustes.
- **Comandos que usa `/adw`** (`.claude/skills/adw/SKILL.md`):
  - git: `remote get-url`, `status --porcelain`, `fetch`, `switch`, `switch -c`, `merge --ff-only`, `ls-remote --heads`, `add -- …`, `diff --cached --check`, `commit`, `push -u <remoto> adw/…`, `branch -D adw/…`;
  - gh: `issue view`, `issue comment`, `pr list`, `pr create`;
  - base de test: `PATH="$PATH:$HOME/go/bin" soda migrate -e test`;
  - borrar `migrations/schema.sql`.
- **Comandos que usan `/implement`, `/bug` y `/feature`** (observados en esta sesión): `gofmt -l .`, `go vet ./...`, `GO_ENV=test go test … ./...`, `go build ./...`, `go env GOMODCACHE`, `node --check public/assets/app.js`, `git log`, `git diff`, `git show` y lecturas (`cat`, `grep`, `sed -n`, `ls`).
- **CLI instalada:** `claude 2.1.299` en `~/.local/bin`, ya en el `PATH` de `~/.zshrc`. **Sin sesión iniciada** (`claude auth status` → `loggedIn: false`).
- **Reglas de permisos** según la documentación oficial (`code.claude.com/docs/en/permissions`, `/permission-modes`, `/headless`):
  - Forma: `{"permissions": {"allow": [...], "deny": [...]}}`. Se evalúa primero `deny`, después `ask` y por último `allow`.
  - Bash se escribe `Bash(<prefijo> *)`; `*` admite cualquier texto, incluidos espacios.
  - Los comandos compuestos (`&&`, `;`, `|`) se separan y **cada parte** tiene que coincidir con una regla `allow`. Las reglas `deny` se aplican a cualquier parte.
  - Las reglas comparan el texto literal del comando: `git -C . push` no coincide con `Bash(git push *)`. No es una barrera de seguridad completa.
  - `Edit` sin argumentos cubre todas las herramientas de edición; las reglas `Write(...)` no se consultan nunca. `Read`, `Grep`, `Glob` y los Bash de solo lectura (`ls`, `cat`, `grep`, `find`) no necesitan regla dentro del directorio de trabajo.
  - La documentación no dice que invocar una skill necesite regla. Se añade `Skill` por si acaso, porque no tiene coste.
  - En `-p`, lo que no está permitido se deniega sin preguntar y la ejecución continúa.
  - **Confianza:** si la carpeta no se ha aceptado nunca como confiable, `claude -p` **no aplica las reglas `allow`** del proyecto (avisa por stderr con "this workspace has not been trusted"); `deny` sí se aplica. Se resuelve abriendo `claude` una vez en el repo y aceptando el diálogo.
- **Los ajustes de proyecto se aplican a todas las sesiones de Claude Code en este repo**, también a las interactivas y a esta app. Las reglas `allow` harán que esos comandos dejen de pedir confirmación también ahí.

Restricciones: no se tocan `adws/*`, `.claude/skills/*` ni `.claude/commands/*`, no hay dependencias nuevas y no se usa `defaultMode`. `acceptEdits` cambiaría el comportamiento de las sesiones interactivas más de lo necesario, y los modos `auto` o `bypassPermissions` se ignoran en ajustes de proyecto.

## Alcance

Incluido:

- Crear `.claude/settings.json` con el contenido exacto de la sección siguiente.
- Añadir en `adws/README.md`, sección "Preparación", **un párrafo** que diga que los permisos de `claude -p` están en `.claude/settings.json` y que hay que abrir `claude` una vez en el repo para iniciar sesión y aceptar la confianza.

Excluido:

- Login de la CLI y aceptar la confianza de la carpeta (el usuario).
- `~/.claude/settings.json` (ajustes globales del usuario) y `.claude/settings.local.json`.
- Permisos para Docker, `psql`, `curl` o `python3`. `/adw` no los necesita: la verificación manual en la app queda para la revisión humana del PR.

## Contenido de `.claude/settings.json`

```json
{
  "permissions": {
    "allow": [
      "Edit",
      "Skill",
      "Bash(git status *)",
      "Bash(git diff *)",
      "Bash(git log *)",
      "Bash(git show *)",
      "Bash(git remote get-url *)",
      "Bash(git ls-remote *)",
      "Bash(git fetch *)",
      "Bash(git switch *)",
      "Bash(git merge --ff-only *)",
      "Bash(git add -- *)",
      "Bash(git restore *)",
      "Bash(git commit *)",
      "Bash(git branch --list *)",
      "Bash(git branch -D adw/*)",
      "Bash(git push -u origin adw/*)",
      "Bash(gh issue view *)",
      "Bash(gh issue comment *)",
      "Bash(gh pr list *)",
      "Bash(gh pr view *)",
      "Bash(gh pr create *)",
      "Bash(gofmt *)",
      "Bash(go env *)",
      "Bash(go vet *)",
      "Bash(go build *)",
      "Bash(go test *)",
      "Bash(GO_ENV=test go test *)",
      "Bash(soda migrate -e test)",
      "Bash(PATH=* soda migrate -e test)",
      "Bash(node --check *)",
      "Bash(rm migrations/schema.sql)",
      "Bash(rm -f migrations/schema.sql)"
    ],
    "deny": [
      "Bash(git push * main)",
      "Bash(git push * main:*)",
      "Bash(git push * HEAD:main)",
      "Bash(git push *--force*)",
      "Bash(git push * -f *)",
      "Bash(git push -f *)",
      "Bash(git reset --hard *)",
      "Bash(git clean *)",
      "Bash(gh pr merge *)",
      "Bash(gh repo *)",
      "Bash(sudo *)",
      "Bash(rm -rf *)",
      "Bash(curl *)",
      "Bash(wget *)",
      "Edit(.claude/settings.json)",
      "Edit(.claude/settings.local.json)",
      "Edit(.env)",
      "Edit(.env.*)",
      "Read(.env)",
      "Read(.env.*)"
    ]
  }
}
```

Notas del diseño:

- **Push:** solo se permite `git push -u origin adw/…`. Todo lo demás (otras ramas, `main` o otros remotos) no tiene regla `allow` y se deniega en `-p`. Las reglas `deny` sobre `main` y `--force` son una segunda barrera, también para las sesiones interactivas.
- **El remoto `origin` está fijado** en la regla de push porque es el que configura `config/adw.yml`. Si se cambia `--remote`, hay que cambiar esta regla.
- **Variables de entorno delante del comando:** hay reglas para `GO_ENV=test go test` y `PATH=… soda`, porque la regla se compara con el texto del comando. Si la sesión del trigger ya tiene `~/go/bin` en el `PATH`, basta `soda migrate -e test`.
- **`gh pr merge` y `gh repo`** se deniegan expresamente, para que tampoco pasen por error en sesiones interactivas.

## Archivos relevantes

| Ruta | Responsabilidad |
| --- | --- |
| `.claude/settings.json` (nuevo) | Permisos de proyecto para `claude -p` y para las sesiones interactivas en este repo. |
| `adws/README.md` | Un párrafo nuevo en "Preparación". |
| `.claude/skills/adw/SKILL.md` | Fuente de los comandos que hay que permitir. Sin cambios. |
| `adws/trigger.go` | Lanza `claude -p` sin flags. Sin cambios. |
| `config/adw.yml` | Fija `--remote origin --base main`. Sin cambios. |

## Implementación paso a paso

1. Confirmar el estado: `git status --short --branch` (rama `chore/adw-permisos`, solo este plan) y `test ! -e .claude/settings.json`.
2. Crear `.claude/settings.json` con el contenido exacto de arriba.
3. Validar que es JSON: `python3 -m json.tool .claude/settings.json > /dev/null`. `python3` ya está en el sistema; no es una dependencia del proyecto.
4. Añadir a `adws/README.md`, al final de la sección "## Preparación", este párrafo:
   > Los permisos que necesita `claude -p` para ejecutar `/adw` sin preguntar están en `.claude/settings.json`: editar archivos, git sobre ramas `adw/…`, `gh pr` e `issue`, `go`, `gofmt`, `soda` y `node --check`. Están denegados el push a `main`, el force-push y `gh pr merge`. Antes del primer uso, abre `claude` una vez en la raíz del repo para iniciar sesión y aceptar la confianza de la carpeta; sin eso, `claude -p` no aplica las reglas `allow`.
5. Ejecutar las validaciones.

## Tests y validación

Automáticas (no hay código Go nuevo):

```bash
python3 -m json.tool .claude/settings.json > /dev/null
gofmt -l .
go vet ./...
go test -count=1 -p 1 ./adws
git diff --name-only
git status --short
```

- El JSON debe ser válido.
- `go test ./adws` debe seguir en verde: el trigger y su configuración no cambian.
- `git status` debe mostrar solo `.claude/settings.json`, `adws/README.md` y este plan.

Funcionales (manuales, del usuario, **después** de hacer login y aceptar la confianza; consumen uso de Claude):

1. `claude -p "Ejecuta 'git status --porcelain' y 'go env GOMODCACHE' y muestra la salida"` desde la raíz del repo: debe ejecutar los dos sin denegaciones y sin el aviso "workspace has not been trusted".
2. `claude -p "Ejecuta 'git push --dry-run origin main' y dime qué ha pasado"`: debe responder que el permiso se ha denegado.
3. Ensayo completo: crear una issue pequeña con las etiquetas `adw` y `enhancement` y ejecutar `bash adws/trigger-github.sh --once`. Debe acabar en un PR `adw/<n>-…` con un comentario en la issue, y el checkout en `main` limpio.

## Riesgos y recuperación

- **Falta un permiso.** Si `/adw` necesita un comando que no está en `allow`, en `-p` se deniega y el trabajo se queda a medias. En el peor caso hay una rama local `adw/…` y ningún PR, y el trigger no lo detecta. Recuperación: ver en la salida qué se denegó, añadir una regla concreta y repetir. Para limpiar: `git switch main` y `git branch -D adw/…`.
- **Efecto en las sesiones interactivas:** los comandos de `allow` dejan de pedir confirmación en cualquier sesión de Claude Code en este repo, incluida esta app. Si no se quiere, mover `allow` a `.claude/settings.local.json` (sin versionar) y dejar `deny` en `settings.json`.
- **Las reglas comparan texto:** formas alternativas (`git -C . push`, alias, scripts) pueden esquivar `deny`. La protección de fondo es que `allow` es una lista cerrada y que los límites de `/adw` prohíben esas acciones. Como refuerzo opcional, fuera de este plan, se puede proteger la rama `main` en GitHub.
- **Confianza o login pendientes:** sin ellos, `-p` ignora `allow` (o no arranca) y `/adw` acaba sin PR. Se detecta por el aviso en stderr.
- **Revertir:** `git rm .claude/settings.json` y `git checkout main -- adws/README.md`.
