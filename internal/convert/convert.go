// Package convert pasa un recurso de cátedra a texto plano legible.
//
// Delega en herramientas externas (pdftotext, pandoc, soffice) en vez de usar
// librerías Go: para PDF no hay nada en Go que se acerque a poppler.
package convert

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Umbral de éxito. Un PDF escaneado sin capa de texto devuelve exit 0 y una
// salida de saltos de línea y form-feeds: "convirtió" no es exit 0, es
// "extrajo texto". Por debajo de esto el recurso se marca ⚠ y se deja el
// original.
const MinPalabras = 20

const timeout = 90 * time.Second

type Resultado struct {
	Texto      string
	Palabras   int
	Herramienta string
}

var ErrSinTexto = fmt.Errorf("se extrajeron menos de %d palabras", MinPalabras)

// Archivo convierte según la extensión. Devuelve ErrSinTexto si el archivo se
// procesó sin error pero no salió texto útil.
func Archivo(ruta string) (Resultado, error) {
	ext := strings.ToLower(filepath.Ext(ruta))

	var crudo string
	var err error
	var herramienta string

	switch ext {
	case ".pdf":
		herramienta = "pdftotext"
		// -layout preserva columnas y sangrías; sin él los apuntes a dos
		// columnas salen intercalados y son ilegibles.
		crudo, err = correr("pdftotext", "-layout", "-enc", "UTF-8", ruta, "-")
	case ".txt", ".md", ".markdown":
		herramienta = "cat"
		crudo, err = correr("cat", ruta)
	case ".html", ".htm", ".docx", ".odt", ".epub", ".rtf":
		herramienta = "pandoc"
		crudo, err = correr("pandoc", "-t", "markdown-raw_html", "--wrap=none", ruta)
	case ".ppt", ".pptx", ".doc", ".xls", ".xlsx":
		// pandoc no lee formatos de presentación ni binarios de Office viejos.
		herramienta = "soffice"
		crudo, err = viaLibreOffice(ruta)
	default:
		return Resultado{}, fmt.Errorf("extensión no soportada: %q", ext)
	}
	if err != nil {
		return Resultado{}, fmt.Errorf("%s: %w", herramienta, err)
	}

	texto := Normalizar(crudo)
	n := contarPalabras(texto)
	res := Resultado{Texto: texto, Palabras: n, Herramienta: herramienta}
	if n < MinPalabras {
		return res, ErrSinTexto
	}
	return res, nil
}

// Normalizar arregla el texto que sale de las herramientas.
//
// Los PDFs de la cátedra vienen de LaTeX, que emite la i sin punto (U+0131)
// más el acento como carácter combinante aparte. NFC NO lo arregla: no existe
// forma precompuesta de U+0131+U+0301. Medido en un apunte real: 131 acentos y
// 30 tildes rotos. Sin esto, "Guía" queda "Guı́a" y ni el grep la encuentra.
func Normalizar(s string) string {
	r := strings.NewReplacer(
		"ı́", "í", // ı + acute
		"ı̀", "ì",
		"ı̂", "î",
		"ı̈", "ï",
		"ı", "i", // dotless suelta: en castellano no existe
		"ﬁ", "fi", // ligaduras que algunos PDFs emiten como un glifo
		"ﬂ", "fl",
		"­", "", // guion blando
		"\f", "\n", // form-feed = fin de página
	)
	s = r.Replace(s)
	// El resto de los combinantes (n+tilde, vocal+acento) sí compone NFC.
	s = norm.NFC.String(s)
	return strings.TrimSpace(s)
}

func contarPalabras(s string) int {
	n := 0
	enPalabra := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if !enPalabra {
				n++
				enPalabra = true
			}
			continue
		}
		enPalabra = false
	}
	return n
}

func correr(prog string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, prog, args...)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errBuf.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("%s", msg)
	}
	return out.String(), nil
}

// viaLibreOffice convierte a texto en un directorio temporal: soffice no
// escribe a stdout y decide el nombre de salida por su cuenta.
func viaLibreOffice(ruta string) (string, error) {
	dir, err := mkdirTemp()
	if err != nil {
		return "", err
	}
	defer removeAll(dir)

	if _, err := correr("soffice", "--headless", "--convert-to", "txt:Text",
		"--outdir", dir, ruta); err != nil {
		return "", err
	}
	base := strings.TrimSuffix(filepath.Base(ruta), filepath.Ext(ruta)) + ".txt"
	return leer(filepath.Join(dir, base))
}
