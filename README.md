# curza-sync

Baja el material de las materias de CURZA (Moodle/PEDCO) y lo deja en **un archivo
markdown por unidad**, para no tener que ir a buscar recurso por recurso antes de cada
clase.

Estado: **en construcción.** Andan tres paquetes; falta el `main` que los cablea, que
necesita credenciales de PEDCO.

## Por qué existe, en una línea

`pedco-bot` ya scrapea las entregas y los parciales dos veces por día, pero desde jun-2026
no escribe `.ics`: eso vive solo en mensajes de Telegram. **Lo que decide las notas es lo
único que no aparece en el calendario.**

## Cómo correrlo

```sh
nix develop     # go + pandoc + poppler-utils
```

LibreOffice no está en el devshell a propósito: ya viene con el sistema y son ~1 GB.

## `internal/convert`

Pasa un recurso a texto plano. Delega en herramientas externas porque para PDF no hay
nada en Go que se acerque a poppler.

| Extensión | Herramienta |
|---|---|
| `.pdf` | `pdftotext -layout` |
| `.html .htm .docx .odt .epub .rtf` | `pandoc` |
| `.ppt .pptx .doc .xls .xlsx` | `soffice --headless` |
| `.txt .md` | tal cual |

**"Convirtió" no es exit 0, es *extrajo texto*.** Un PDF escaneado sin capa de texto sale
con exit 0 y una salida de saltos de línea: por debajo de `MinPalabras` (20) devuelve
`ErrSinTexto` y el llamador tiene que marcar el recurso ⚠ y dejar el original.

### El defecto de los acentos

Los apuntes de la cátedra vienen de LaTeX, que emite la **i sin punto** (`U+0131`) más el
acento como carácter combinante aparte. **NFC no lo arregla**: no existe forma
precompuesta de `U+0131`+`U+0301`. Medido en un apunte real: 131 acentos y 30 tildes
rotos en un solo archivo. Sin normalizar, "Guía" queda "Guı́a" y ni el `grep` la
encuentra. `Normalizar()` mapea los casos de la i sin punto y después aplica NFC, que sí
compone el resto (n+tilde → ñ).

### Verificado

Contra 61 archivos reales de `~/Clases/` (dos cuatrimestres cursados): **60 convertidos**,
de 263 a 91.504 palabras, incluido un PDF de 7,4 MB con imágenes. Los 2 restantes son un
README de 14 palabras y un log vacío — el umbral haciendo su trabajo.

Gotcha del terreno: los archivos de cátedra **tienen espacios y acentos en el nombre**
("Trabajo practico Nro 1.pdf"). Cualquier pegamento en shell tiene que citar bien.


## `internal/domain`

**Un** tipo, no cinco. `Entrega` es un vencimiento de Moodle, que es el único consumidor
real que hay hoy; los tipos del material se agregan cuando exista quien los lea. Tampoco
hay puertos: un puerto se gana su lugar cuando hay una segunda implementación que
intercambiar.

## `internal/parsefecha`

Moodle muestra las fechas así, medido sobre un volcado real:

```
"Hoy , 18:00"
"viernes, 15 mayo , 23:55"
"viernes, 15 mayo , 18:00 » 20:00"
```

Tres trampas: **no trae año** (un "15 enero" leído en diciembre es del año que viene y un
"20 diciembre" leído en enero es del anterior), hay formas relativas, y los rangos usan `»`.
El año se resuelve eligiendo entre año-1, año y año+1 el que deje la fecha más cerca de la
referencia, sin reglas ad hoc. `ref` se inyecta en vez de llamar a `time.Now()` adentro:
sin eso la función no es testeable.

Verificado contra los 10 formatos del volcado, los dos bordes de año, `Hoy`/`Mañana`/`Ayer`,
un rango que cruza medianoche, y basura (que falla con error claro, no en silencio).

## `internal/emitics`

Escribe los vencimientos en `~/.local/share/calendars/pedco/`. Las reglas no son estilo:
cada una salió de un fallo medido contra khal 0.14 y el modelo vdir.

- **Un UID por archivo, y `basename == UID`.** En un store vdir la clave de deduplicación es
  la RUTA, no el UID: dos archivos con el mismo UID muestran dos eventos. Y un `.ics` con
  dos UIDs distintos hace que khal descarte el **archivo entero** con exit 0, warning solo
  por stderr.
- **Escritura atómica y NUNCA preservando el mtime**: khal invalida su caché por mtime, no
  por contenido, así que un `cp -p` deja servido el evento viejo.
- **Los huérfanos se borran comparando contra el disco**, no contra un archivo de estado:
  el paquete posee el namespace `moodle-*.ics` completo, así que el filesystem ES el
  registro de lo que emitió la vez pasada.
- **`VTIMEZONE` en cada archivo**, CRLF y plegado a 75 octetos (RFC 5545). khal tolera
  líneas largas y LF suelto, pero el repo es público y un import a Google Calendar o iOS no
  perdona.
- Escapado RFC de `,` `;` `\` y saltos: los títulos vienen de la cátedra y traen comas casi
  siempre.

Verificado punta a punta con un khal aislado: los eventos se listan sin un solo warning, los
títulos con comillas y punto y coma sobreviven, la segunda corrida actualiza en vez de
duplicar, y el huérfano se borra.

## Lo que falta

El `main`. Necesita credenciales de PEDCO por `sops` — y **antes de eso, rotar la
contraseña**: la vieja viajó al webhook de un atacante en ago-2026 y sigue sin rotar.
