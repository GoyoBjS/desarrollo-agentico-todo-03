# Trigger de issues

Este lanzador consulta las issues abiertas con la etiqueta configurada, de más recientes a más antiguas por **fecha de creación**, e invoca Claude de una en una. Espera a que termine cada proceso antes de lanzar el siguiente. Cada consulta recoge hasta 100 issues; al terminar la lista espera el intervalo configurado y vuelve a consultar.

## Preparación

Necesitas Go (la versión del proyecto), Bash, GitHub CLI (`gh`) y Claude Code (`claude`) en `PATH`, autenticados, y el entorno de desarrollo del proyecto ya disponible. El lanzador reutiliza la librería YAML que ya incluye el proyecto. `--jq` usa el formateador integrado de `gh`: no hace falta instalar `jq`.

El trigger no contiene el pipeline: invoca el workflow `/adw`, definido en [`.claude/skills/adw/SKILL.md`](../.claude/skills/adw/SKILL.md). Ese archivo describe sus pasos y sus límites. También puedes elegir otro prompt que pueda completar el encargo sin preguntas intermedias.

Las piezas disponibles son `/feature`, `/bug`, `/chore` y `/implement`, en `.claude/skills/`. Las tres primeras solo escriben un plan; `/implement` ejecuta un plan aprobado sin hacer commit, push ni PR. Tu workflow debe coordinar esas fases y definir sus pasos posteriores y sus límites.

## Configurar

Edita **`config/adw.yml`**. No necesitas editar el código ni establecer variables de entorno:

```yaml
repo: OWNER/REPO
label: adw
interval_seconds: 20
prompt: >-
  /adw {{issue_url}} --remote origin --base main
```

- `repo`: repositorio donde buscar las issues. Se entrega vacío para que lo configures antes de arrancar.
- `label`: etiqueta que activa el trabajo.
- `interval_seconds`: espera entre consultas, una vez terminada la lista anterior.
- `prompt`: texto que recibe Claude. Puede invocar tu propia skill o comando; `{{issue_url}}` se sustituye por la URL de cada issue. No se evalúa como código de shell.

`origin` y `main` son argumentos del `/adw` del ejemplo, no opciones del trigger. Cámbialos según tu workflow. El bloque `>-` permite escribir un prompt largo en varias líneas: YAML las une con espacios.

Usa issues revisadas y listas para ejecutar; poner la etiqueta configurada significa que autorizas el encargo dentro de los límites de tu workflow.

## Arrancar

Desde la raíz del repositorio:

```bash
bash adws/trigger-github.sh
```

Para consultar y procesar la lista una sola vez:

```bash
bash adws/trigger-github.sh --once
```

También puedes ejecutar `go run ./adws` directamente. El script sitúa la ejecución en este checkout; no prepara otro entorno. Ambos admiten `--config ruta/al/config.yml` para elegir otro fichero. `--once` ejecuta Claude: no es una simulación.

Cada invocación equivale a `claude -p "<prompt configurado con la URL>"`. El trigger no se arranca automáticamente. Detén el sondeo con `Ctrl+C`. Si `gh` o `claude` devuelve un código de error, el lanzador termina.

## Límites de esta versión

- No modifica etiquetas, cierra issues ni recuerda las que ha ejecutado. Si una issue sigue abierta y conserva la etiqueta, puede volver a ejecutarse en la siguiente consulta o al reiniciar el trigger. Abrir un PR no cierra la issue por sí solo.
- No crea entornos, cambia de rama ni limpia el checkout entre encargos. El estado que deja un workflow afecta al siguiente.
- No gestiona feedback, reintentos, recuperación ni coordinación con otros procesos.
- No interpreta el resultado de Claude. Un código cero no demuestra que exista un PR; el workflow puede haber informado de un bloqueo.
- No desactiva los permisos de Claude. Tu workflow y su configuración deben permitir las acciones previstas; sin una respuesta humana, una ambigüedad o un permiso pendiente puede impedir completar el encargo.

## Comprobar el lanzador

```bash
go test -p 1 ./adws
```

Los tests usan `gh` y `claude` simulados: no consultan GitHub ni ejecutan un agente real. Comprueban la configuración YAML, la consulta, el orden secuencial, la sustitución del prompt, las paradas por error y el sondeo sin memoria. No sustituyen un ensayo completo del workflow hasta el PR.
