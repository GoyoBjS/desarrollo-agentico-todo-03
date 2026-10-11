# Práctica SDD manual: tú decides cómo continuar

## Copia de inicio

- Repositorio acordado: `Entaina/desarrollo-agentico-todo-03`.
- Rama de origen: `j2-openspec-base`.
- Base de código: J1, `5193761` (`j1-agent-commands` / `todo-02/main`).
- Kit: OpenSpec **1.14.1**, integración oficial para Claude Code, perfil `core`.

Esta copia contiene la aplicación de J1 y el kit. **No contiene prioridades ni un cambio OpenSpec planificado o implementado.** Conserva los siete comandos generales de J1. No contiene la metodología SDD propia ni el ADW de J2.

Estado de preparación: la base se conserva en esta rama. La publicación requiere autorización y el entorno dedicado debe validarse por separado. No se presenta todavía como una rama descargable ni como un entorno de aula ensayado.

Usa tu propia copia preparada, no la rama `j2-openspec-eval`: esa rama ya contiene el resultado. Antes de empezar, la app debe responder y el entorno debe poder ejecutar migraciones y tests. El kit no se instala durante los 45 minutos de práctica. Si falta una herramienta o la app no arranca, avisa antes de continuar.

## Abre Claude Code

Desde la raíz de tu copia, con OpenSpec disponible en el `PATH` de esa misma terminal:

```bash
openspec --version
claude
```

La versión esperada es `1.14.1`. Abre Claude normalmente, **sin `--restricted`**: ese modo no carga estas skills y aliases del proyecto. No uses `bypassPermissions` ni retires límites para resolver un bloqueo.

Los comandos siguientes se escriben en la consola de Claude, no en la terminal del sistema.

## Encargo común

Añade prioridad **alta, media o baja** a las tareas. Permite elegirla al crear, verla en la lista y cambiarla después.

- Media por defecto y para las tareas existentes.
- Rechazo de valores desconocidos.
- Conserva creación, validación del título, completar/reabrir y eliminar.
- Conserva el orden actual del listado y no añadas dependencias.

## 1. Pide la planificación

En Claude Code, envía este encargo como una sola petición:

```text
/opsx:propose Añade prioridad alta, media o baja a las tareas. Media por defecto y para las tareas existentes. Permite elegirla al crear, verla en la lista y cambiarla después. Rechaza valores desconocidos. Conserva el resto del comportamiento y no cambies el orden del listado. No añadas dependencias.
```

Responde en esta consola si el agente necesita aclaraciones. En `core`, `propose` prepara conjuntamente propuesta, especificación, diseño y tareas; no son cuatro comandos que debas lanzar por separado. Al terminar debe detenerse **sin implementar**. Anota el nombre real del cambio y abre las rutas que te indique.

## 2. Revisa antes de continuar

Lee primero la especificación: ¿describe el comportamiento solicitado y sus errores? Después revisa el diseño y las tareas: ¿las decisiones y comprobaciones propuestas permiten cumplir ese comportamiento sin ampliar el alcance?

Los artefactos del caso, con el esquema `spec-driven`, se ubican bajo `openspec/changes/<nombre-real>/`: `proposal.md`, `specs/<capacidad>/spec.md`, `design.md` y `tasks.md`. Usa las rutas reales que indique OpenSpec.

Si falta algo o no estás conforme, explica la diferencia en Claude y pide corregir los artefactos **sin implementar todavía**. Las dudas se resuelven en esta consola, no en una issue ni mediante un observador de GitHub.

## 3. Pide la implementación por separado

Solo cuando estés conforme con el plan:

```text
/opsx:apply <nombre-real-del-cambio>
```

Sustituye el marcador por el nombre que te dio OpenSpec. Esta es una petición nueva: planificar no autorizó implementar. El agente trabaja con los artefactos y actualiza las tareas. Si encuentra un bloqueo, lee el motivo y decide si aclarar, corregir o pedir ayuda; no le pidas saltarse permisos o comprobaciones.

## 4. Comprueba el resultado

En la URL de **tu** app preparada:

1. Crea tareas con las tres prioridades y comprueba sus etiquetas.
2. Comprueba el valor media por defecto y en las tareas que ya existían.
3. Cambia una prioridad y recarga: debe mantenerse.
4. Completa, recarga, reabre y recarga: estado y prioridad deben mantenerse.
5. Comprueba que siguen funcionando el título obligatorio y eliminar, y que el listado conserva su orden.

Pide al agente el resultado de sus tests y de la comprobación de valores desconocidos. Si no pudo ejecutarlos, conserva el bloqueo y no des el trabajo por comprobado. Esta revisión breve no sustituye las pruebas ni desarrolla todavía un dossier de evidencias.

No hace falta crear un PR, hacer merge, archivar OpenSpec ni publicar el código para terminar esta práctica.

## Puesta en común

Conserva el nombre del cambio y una decisión concreta que hayas tomado:

- ¿Qué has tenido que lanzar tú?
- ¿Qué has aclarado o corregido antes de implementar?
- ¿Quién ha decidido cómo continuar?

El comentario «Quiero que las tareas de prioridad alta aparezcan primero» se reserva para el contraste posterior. No forma parte de este encargo.
