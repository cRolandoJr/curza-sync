# curza-sync

Baja el material de las materias de CURZA (Moodle/PEDCO) y lo deja en **un archivo
markdown por unidad**, para no tener que ir a buscar recurso por recurso antes de cada
clase.

Estado: **en construcción.** Hoy funciona solo `internal/convert`.

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
