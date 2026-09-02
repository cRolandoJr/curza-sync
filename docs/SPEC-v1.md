# curza-sync · spec v1

*Escrito 2026-09-02. Decisiones, no mecanismos. Lo que está sin verificar está marcado.*

## Qué es

Una herramienta local que habla con la API de Moodle y deja, en tu disco:

- **un archivo markdown por unidad**, con el material de la cátedra ya convertido a texto
- **un tablero de entregas** con qué debés, cuándo vence y qué ya entregaste
- todo lo anterior **sin que nada salga de tu máquina**

Usuario cero: Rolando, 9 materias en PEDCO (Moodle 4.1+). Si no le sirve a él todos los
días, no le va a servir a nadie.

## Qué NO es la v1

- No escribe en Moodle. No entrega TPs, no postea en foros, no crea eventos.
- No tiene servidor ni cuentas. Un binario, tu token, tu disco.
- No es multi-usuario.
- No tiene interfaz gráfica — eso es la v2 (`B`), **el mismo núcleo con otro adaptador**.
- No indexa ni vectoriza nada (ver decisión 3).

---

## Terreno verificado — 2026-09-02

Todo esto se midió contra PEDCO con `curl`, no se supone:

| Hecho | Valor |
|---|---|
| Funciones de la API habilitadas | **404** (Moodle 4.1+) |
| Materias inscriptas | 9 |
| Archivos de material | **193** · 176 PDF (91 %), 7 odt, 3 html, 3 dia, 2 7z, resto suelto |
| PDFs sin capa de texto | **2** de 176 |
| Palabras totales | **1.299.555** |
| Material de cátedra por materia | **18.000 – 42.000 palabras** |
| Bibliografía: un solo libro | **507.875 palabras** |
| Entregas | **37**, con estado real (`submitted` / `new` / `sin-intento`) y nota |
| Quizzes / parciales | **11**, con fecha |
| Notas del gradebook | **8 de 9 materias vacías** |

---

## Decisiones

### 1. La unidad es la sección de Moodle. No se infiere.

`core_course_get_contents` devuelve secciones (`"Módulo 1: Introducción al Scripting…"`),
y dentro los `label` (título de cada clase) y los `resource` (los archivos). La estructura
que el plan viejo pensaba deducir **viene dada**. Un `.md` por sección.

### 2. El corpus son dos poblaciones y se tratan distinto.

**Material de cátedra** (apuntes, clases, TPs) vs **bibliografía** (libros de referencia).
Difieren en un orden de magnitud: 42.000 palabras contra 508.000 en la misma materia.

Se separan por **umbral de tamaño, configurable** (default: más de 50.000 palabras = biblio;
**confirmado por el usuario el 2026-09-02**),
y el usuario puede reclasificar. La bibliografía se descarga y se lista, pero **no entra en el
`.md` de la unidad**: iría al asistente y lo reventaría.

### 3. La v1 no lleva recuperación. Con gatillo.

Una unidad entera entra sin problema en el contexto de un modelo. Un índice vectorial es
infraestructura para un problema que hoy no existe.

> **Gatillo:** se agrega índice **el día que una consulta real no entre en contexto**, medido.
> El caso previsto es búsqueda en bibliografía, no material de cátedra.

### 4. Local-first, y es una decisión de producto, no de comodidad.

El material de cátedra es de los profesores y de la universidad. Si un servidor lo almacena,
hay un problema de derechos y de datos personales. Manteniendo todo en el equipo del alumno
ese problema **no existe**, y de paso no hay costo de infraestructura.

### 5. Derivar vs. preguntar

| Dato | Cómo se obtiene | Por qué |
|---|---|---|
| **Año de cursada** | derivado de la categoría (`"2do año (26)"`) | el dato está en la API |
| **Cuatrimestre** | **inferido** de las fechas de entregas y parciales de la materia, **corregible** | `startdate` de todas las materias es el inicio del año, no del cuatrimestre |
| **Carrera** | **declarado por el usuario**, una vez | **no existe en la API**: las dos tecnicaturas comparten árbol de categorías |

> **No inferir lo que se puede preguntar una vez.** Una carrera adivinada clasifica mal en
> silencio y con total seguridad. Preguntar cuesta treinta segundos, una sola vez.

### 6. Núcleo y adaptadores

Bajar, ensamblar y convertir **no saben quién los llama**. La CLI de la v1 y la interfaz web
de la v2 son **dos adaptadores sobre el mismo núcleo**. La v2 no reescribe la v1: le agrega
una entrada.

### 7. El token va en un archivo con permisos 0600, no en `sops`.

**Divergencia consciente** con la decisión del 2026-08-04. Aquella eligió `sops` porque el
usuario era Rolando en NixOS. Un producto tiene que funcionar para un estudiante que no sabe
qué es `sops`. Se guarda en `~/.config/curza/config.toml` con `0600`, y **nunca** en el repo.

### 10. El usuario objetivo usa Windows y no es hábil con computadoras

Declarado por el usuario el 2026-09-02. Consecuencias que ya condicionan la v1:

- **Confirma la elección de Go.** Compila a un `.exe` único sin runtime que instalar. Python o
  Node exigirían que el alumno instale un intérprete: para este usuario, eso es el final.
- **La v2 (interfaz web local) deja de ser opcional.** Una CLI es correcta para el usuario cero
  (Rolando) y es inservible para el usuario objetivo.
- **Nada de `nix develop` en la ruta del usuario final.** Sirve para desarrollar, no para
  distribuir. Y `convert` delega en `pdftotext`/`pandoc`/`soffice`: **en Windows esas
  dependencias externas son un problema abierto** y hay que resolverlo antes de la v2
  (embeber, sustituir por librería Go, o degradar).

### 8. Se guarda estado local para no re-descargar

Qué archivo se bajó, cuándo y con qué hash. Sin eso, cada corrida vuelve a bajar 193 archivos.

### 9. El asistente usa la clave del propio usuario, llamada desde su máquina

Sin proxy, sin servidor intermedio. Consecuencias: costo cero para el proyecto,
responsabilidad cero sobre datos ajenos, y el material **nunca** pasa por un tercero que no
sea el proveedor del modelo que el usuario ya eligió.

**Lo que esto difiere a propósito:** si algún día se cobra una suscripción, lo más probable es
que sea justamente por intermediar esas llamadas — y ahí el material *sí* pasaría por un
servidor propio, lo que reabre la decisión 4. **Esa decisión es de la v2 y no se toma hoy.**
La v1 no cierra ninguna puerta: cambiar de "clave propia" a "proxy" es cambiar un adaptador.

### 13. La selección de contexto se separa de la generación

**Lo que elige qué se le manda al modelo es determinista y se testea con tests normales. Lo
que el modelo redacta, no.** Van en paquetes distintos.

Consecuencias concretas, y aplican desde la fase 2 aunque no haya índice vectorial:

- **La procedencia se conserva por fragmento**, no sólo el texto pegado. Sin eso no se puede
  citar, y la cita es lo que convierte una respuesta en la que hay que creer en una que se
  puede comprobar.
- **Al modelo se le manda más de un fragmento**, nunca uno solo. La selección se equivoca:
  es una distancia entre números, no magia.
- **Se le indica al modelo que use sólo lo que se le da, y que admita cuando no lo sabe.**
- **Cuando un documento cambia, sus fragmentos viejos se borran.** Si no, se responde
  mezclando dos versiones. Esto es lo que resuelve `estado.json` con hashes (decisión 8).

*Fuente: "¿Cómo hago un RAG en Go? Sin framework, con pgvector y todo en local"
(https://youtu.be/Dn3oq8bbcLM), aportado por el usuario el 2026-09-02. Confirma la decisión 3
desde afuera: "si cabe todo, es complejidad de adorno".*

### 11. Se captura todo; se presenta un default curado

**Separar la captura de la presentación.** La captura es barata y no hacerla es caro (habría
que volver a bajar 193 archivos). La presentación es lo que se lee y cambiarla después no
cuesta nada.

- **Se guarda el JSON crudo de cada sección**, completo, sin filtrar.
- **El `.md` renderiza un subconjunto fijo**, decidido abajo.

**Descartado: dejar que el alumno configure qué campos ve.** Once campos son once ramas y
2.048 combinaciones que nadie testea, y le pide once decisiones a un usuario que —por la
decisión 10— no es hábil con computadoras. Además contradice la regla del repo: cada adición
necesita un consumidor real hoy. **Como el dato crudo queda guardado, el día que alguien pida
un campo distinto agregarlo es trivial** — no hace falta construir la configurabilidad hoy
para no perder la opción.

### 12. Qué es una Unidad

Una unidad **es una sección de Moodle**. El `.md` que se genera lleva:

| Va al `.md` | |
|---|---|
| Título de la unidad | `section.name` |
| Resumen que escribió el docente | `section.summary` |
| Títulos de cada clase | los módulos `label` |
| **El texto convertido de cada apunte** | el punto de todo |
| Enlaces externos, como lista al final | módulos `url` |
| Las entregas de esa unidad, con fecha y estado | módulos `assign` |
| Ruta al PDF original en disco | para abrir el archivo real |

| Se guarda, no se muestra | Por qué |
|---|---|
| Descripción de cada recurso | casi siempre repite el título |
| Fecha de modificación de cada archivo | es ruido al estudiar; su lugar es C4 ("qué cambió") |
| Estimación de minutos de lectura | es una estimación y se calcula al vuelo si hace falta |

**Y "ya lo leí" NO va al `.md`, por una razón estructural:** el `.md` **se regenera** en cada
corrida y borraría la marca. **El estado y el contenido generado no pueden vivir en el mismo
archivo.** Va a `estado.json` (decisión 8).

---

## Las capacidades de la v1

Ordenadas por valor para el alumno, no por dificultad.

### C1 · Semáforo de entregas ← **la más importante**

Un tablero con **las 37 entregas de las 9 materias**: qué debés, cuándo vence, y el estado
real (`sin-intento`, `new` = borrador sin enviar, `submitted`) más la nota si está corregida.

Moodle tiene este dato pero **sólo curso por curso**. Nadie lo agrega. En su primera corrida
esta consulta encontró un TP vencido en estado borrador y otro que vencía al día siguiente.

Sin IA. Pura agregación.

### C2 · Un `.md` por unidad

El objetivo original: *"abrís un archivo y tenés la clase entera"*. Título de la unidad, las
clases, el texto de cada apunte convertido, los enlaces externos, y qué se entrega.

### C3 · Parciales y coloquios en el mismo tablero

Los 11 quizzes con fecha, junto a las entregas. Una sola lista de "lo que se viene".

### C4 · Qué cambió desde la última vez

`core_course_get_updates_since`: "el profe subió 2 archivos nuevos y movió la fecha del TP2".
Preciso, en vez del ruido de las notificaciones de Moodle.
🟡 **La función existe; no la probé.**

### C5 · Asistente sobre la unidad · ✅ **premisa verificada 2026-09-02**

Preguntarle al material. Entra en contexto sin índice (decisión 3). Cita el recurso de origen.

**Lo que se verificó:** se tomó una pregunta que el usuario **erró de verdad** en un parcial
("¿con qué se testea la integridad de la memoria en Linux?") y se buscó en el material de esa
misma materia. El apunte la cubre: `3_1_RAM.pdf` líneas 269-303 (36 coincidencias del tema).

**Y encontró la causa del error, no sólo el tema:** el apunte lista **dos** herramientas de RAM
(`memtest86` y `memtester`) y el parcial ofrecía las dos como opciones separadas. El hueco no
era de estudio; la pregunta tenía dos candidatas en el mismo apunte.

Eso es lo que ningún asistente genérico puede hacer: necesita **la pregunta fallada del alumno**
y **su material de cátedra** al mismo tiempo, y ninguna de las dos está en internet.

### C6 · Práctica con el estilo de la cátedra, y repaso por tus errores · ✅ **verificado 2026-09-02**

`mod_quiz_get_user_attempts` + `get_attempt_review` devuelven, por pregunta: **enunciado
completo, todas las opciones, cuál elegiste, el estado y el puntaje**.

**Medido:** 48 preguntas con enunciado completo, de 5 exámenes reales, en **6 formatos**:
multichoice 19 · essay 9 · verdadero-falso 8 · emparejar 6 · respuesta corta 5 · numérica 1.

Eso habilita **dos** cosas, y la primera resultó más valiosa que la segunda:

- **Práctica con el estilo real de la cátedra.** No preguntas genéricas: los formatos que
  usan tus profesores y sus distractores característicos ("Todas las respuestas son
  correctas", "Ninguna de las respuestas es válida").
- **Repaso dirigido por tus errores.** Las preguntas que fallaste, cruzadas con el material.

**Límite del dato:** el review **no revela cuál era la correcta** (sólo "Respuesta incorrecta").
Lejos de ser un problema, obliga al producto a explicar desde el material en vez de soplar.

---

## Descartado, con su razón

| Idea | Por qué no |
|---|---|
| Proyección de nota ("cuánto necesito para promocionar") | **8 de 9 materias tienen el gradebook vacío.** El dato no está |
| Scraping de HTML | La API lo cubre todo, es estable y funciona contra 147.000 sitios |
| Índice vectorial en la v1 | El material de cátedra entra en contexto (decisión 3) |
| Servidor propio | Decisión 4 |

---

## Reparto

| Parte | Quién |
|---|---|
| Tipos de dominio y límites entre paquetes | **Rolando** |
| El primer test de cada paquete | **Rolando** |
| `main`, timer de systemd, empaquetado Nix | **Rolando** (es 00694) |
| Debugging cuando algo falle | **Rolando** — Claude interroga, no repara |
| Cliente HTTP, structs de respuesta, sanitizado de nombres | Claude |
| Plantillas markdown, boilerplate, casos de test 2..N | Claude |

**El corte:** donde hay una decisión que vas a tener que volver a tomar solo, la tomás vos.

---

## Riesgos abiertos

1. **Sólo C4 sigue sin probar** (`core_course_get_updates_since`). C5 y C6 se verificaron el
   2026-09-02 con datos reales.
6. **El review de quiz puede estar cerrado por el docente.** De 8 intentos terminados, **3 no
   devolvieron review** (los de Software Libre). El corpus de C6 no está garantizado: depende
   de cada cátedra. El producto tiene que degradar sin romperse.
7. **La calidad de la práctica generada no está verificada.** Sabemos que hay insumo; no
   sabemos si la salida sirve. Se mide cuando exista, no se supone.
9. **Los modelos de embeddings suelen estar entrenados sobre todo en inglés, y todo el
   material es en español.** Si algún día se dispara el gatillo del índice (decisión 3),
   **probar el modelo de embeddings contra material real en español antes de casarse con él**.
   La fuente citada en la decisión 13 mostró en vivo una recuperación equivocada causada
   exactamente por esto. Y el arreglo estándar cuando pasa se llama **reordenar (reranking)**:
   recuperar 10 fragmentos y pasarlos por un segundo modelo que los ordene mejor.
10. **`pgvector` no sirve para este producto**, aunque sea la opción por defecto del ecosistema:
   exige PostgreSQL instalado y corriendo, contra la decisión 10 (`.exe` único, usuario que no
   es hábil). Si llega el índice, tiene que viajar dentro del binario.
8. **Para este usuario el insumo de "repaso por errores" es chico**: 43 correctas, 2 parciales,
   3 incorrectas sobre 48. Para un alumno promedio sería más rico. Es un argumento a favor de
   priorizar la generación **por estilo** sobre la generación **por errores**.
2. **El corpus crece.** 18-42k palabras es a mitad de cuatrimestre. Medir de nuevo en diciembre.
3. **La escala de las notas se desconoce.** Dos entregas dicen `2.00000` sin saber sobre cuánto.
4. **Otro Moodle puede tener menos funciones habilitadas.** La v1 sirve a PEDCO; el producto
   tendrá que enumerar capacidades al conectar y degradar features que falten.
5. ~~`parsefecha` queda sin consumidor~~ → **RESUELTO 2026-09-02: se borra.** La API devuelve
   timestamps. El test que se escribió sobre él cumplió su función (fue donde se aprendió a
   testear) y queda en la historia de git.
