// Package emitics escribe los vencimientos como .ics en el calendario local.
//
// Es el hueco que cierra: pedco-bot ya scrapea las entregas y los parciales dos
// veces por día, pero desde jun-2026 no escribe .ics, así que eso vive solo en
// mensajes de Telegram. Lo que decide las notas es lo único que no aparece en el
// calendario.
//
// Las reglas de abajo no son estilo: cada una salió de un fallo medido contra
// khal 0.14 / el modelo vdir.
package emitics

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/cRolandoJr/curza-sync/internal/domain"
)

// prefijo delimita el namespace de archivos que este paquete posee. Todo
// `moodle-*.ics` del directorio es suyo: eso permite borrar huérfanos
// comparando contra el disco, sin necesidad de un archivo de estado que
// registre "lo que emití la vez pasada".
const prefijo = "moodle-"

const tzID = "America/Argentina/Buenos_Aires"

type Resumen struct {
	Escritos []string
	Borrados []string
}

// Emitir sincroniza dir con entregas: escribe una por archivo y borra los
// `moodle-*.ics` que ya no correspondan.
func Emitir(entregas []domain.Entrega, dir string) (Resumen, error) {
	var res Resumen

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return res, err
	}

	quedan := map[string]bool{}
	for _, e := range entregas {
		if e.MoodleID == "" {
			return res, fmt.Errorf("entrega %q sin MoodleID: sin identidad estable no hay upsert", e.Titulo)
		}
		if e.Vence.IsZero() {
			return res, fmt.Errorf("entrega %q sin fecha", e.Titulo)
		}
		nombre := prefijo + sanear(e.MoodleID) + ".ics"
		quedan[nombre] = true
		if err := escribirAtomico(filepath.Join(dir, nombre), vevent(e)); err != nil {
			return res, err
		}
		res.Escritos = append(res.Escritos, nombre)
	}

	huerfanos, err := filepath.Glob(filepath.Join(dir, prefijo+"*.ics"))
	if err != nil {
		return res, err
	}
	for _, ruta := range huerfanos {
		if quedan[filepath.Base(ruta)] {
			continue
		}
		if err := os.Remove(ruta); err != nil {
			return res, err
		}
		res.Borrados = append(res.Borrados, filepath.Base(ruta))
	}
	sort.Strings(res.Escritos)
	sort.Strings(res.Borrados)
	return res, nil
}

// escribirAtomico escribe a un temporal y renombra.
//
// Y NUNCA preserva el mtime: khal invalida su caché por mtime, no por
// contenido, así que un `cp -p` o un os.Chtimes deja servido el evento viejo.
// Un .ics a medio escribir hace que khal saltee el archivo ENTERO con exit 0 y
// el warning solo por stderr, o sea que la clase desaparece sin que nada avise.
func escribirAtomico(ruta, contenido string) error {
	tmp, err := os.CreateTemp(filepath.Dir(ruta), ".curza-*.tmp")
	if err != nil {
		return err
	}
	nombreTmp := tmp.Name()
	defer os.Remove(nombreTmp) // no-op si el rename salió bien

	if _, err := tmp.WriteString(contenido); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(nombreTmp, 0o644); err != nil {
		return err
	}
	return os.Rename(nombreTmp, ruta)
}

// vevent arma el archivo. UN VEVENT y UN UID por archivo: khal descarta el
// archivo completo si encuentra dos UIDs distintos adentro. Y el basename tiene
// que coincidir con el UID, porque en un store vdir la clave de deduplicación
// es la RUTA, no el UID — dos archivos con el mismo UID muestran dos eventos.
func vevent(e domain.Entrega) string {
	fin := e.Hasta
	if fin.IsZero() {
		// Un vencimiento no tiene duración, pero un VEVENT de largo cero
		// algunos clientes lo esconden. Un minuto lo hace visible sin cruzar
		// la medianoche cuando el TP vence 23:55.
		fin = e.Vence.Add(time.Minute)
	}

	var b strings.Builder
	l := func(s string) { b.WriteString(plegar(s)) }

	l("BEGIN:VCALENDAR")
	l("VERSION:2.0")
	l("PRODID:-//curza-sync//vencimientos//ES")
	l("CALSCALE:GREGORIAN")
	b.WriteString(bloqueTZ)
	l("BEGIN:VEVENT")
	l("UID:" + prefijo + sanear(e.MoodleID))
	l("SUMMARY:" + escapar(etiqueta(e.Tipo)+" "+e.Titulo))
	l("DTSTAMP:" + e.Vence.UTC().Format("20060102T150405Z"))
	l("DTSTART;TZID=" + tzID + ":" + local(e.Vence))
	l("DTEND;TZID=" + tzID + ":" + local(fin))
	if e.Materia != "" {
		l("CATEGORIES:" + escapar(e.Materia))
		l("DESCRIPTION:" + escapar(e.Materia))
	}
	if e.URL != "" {
		l("URL:" + escapar(e.URL))
	}
	l("END:VEVENT")
	l("END:VCALENDAR")
	return b.String()
}

func etiqueta(t domain.Tipo) string {
	switch t {
	case domain.Tarea:
		return "TP:"
	case domain.Cuestionario:
		return "Cuestionario:"
	case domain.Examen:
		return "EXAMEN:"
	}
	return "Vence:"
}

func local(t time.Time) string { return t.Format("20060102T150405") }

// escapar aplica RFC 5545 §3.3.11. Los títulos vienen de la cátedra y traen
// comas casi siempre ("TP 3: procesos, señales y prioridades").
func escapar(s string) string {
	return strings.NewReplacer(
		`\`, `\\`,
		";", `\;`,
		",", `\,`,
		"\r\n", `\n`,
		"\n", `\n`,
		"\r", `\n`,
	).Replace(s)
}

// plegar corta a 75 octetos con CRLF + espacio, como pide RFC 5545 §3.1. khal
// tolera líneas largas y LF suelto, pero el repo es público y un import a
// Google Calendar o iOS no perdona.
func plegar(linea string) string {
	const max = 75
	if len(linea) <= max {
		return linea + "\r\n"
	}
	var b strings.Builder
	resto := linea
	primera := true
	for len(resto) > 0 {
		limite := max
		if !primera {
			limite = max - 1 // el espacio de continuación cuenta
		}
		if len(resto) <= limite {
			if !primera {
				b.WriteString(" ")
			}
			b.WriteString(resto)
			break
		}
		// No cortar en medio de un carácter UTF-8.
		corte := limite
		for corte > 0 && resto[corte]&0xC0 == 0x80 {
			corte--
		}
		if !primera {
			b.WriteString(" ")
		}
		b.WriteString(resto[:corte])
		b.WriteString("\r\n")
		resto = resto[corte:]
		primera = false
	}
	b.WriteString("\r\n")
	return b.String()
}

// sanear deja el MoodleID usable como nombre de archivo. Hoy son numéricos,
// pero el basename es la clave de deduplicación: si llegara algo con "/" el
// archivo terminaría en otro directorio.
func sanear(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			return r
		}
		return '-'
	}, s)
}

// bloqueTZ va en cada archivo: un TZID sin VTIMEZONE hace que khal avise y
// siga, y deja el archivo a merced de la tzdata del consumidor. Argentina no
// tiene DST desde 2009, así que un solo componente STANDARD alcanza.
const bloqueTZ = "BEGIN:VTIMEZONE\r\n" +
	"TZID:" + tzID + "\r\n" +
	"BEGIN:STANDARD\r\n" +
	"DTSTART:20090314T230000\r\n" +
	"TZNAME:-03\r\n" +
	"TZOFFSETFROM:-0200\r\n" +
	"TZOFFSETTO:-0300\r\n" +
	"END:STANDARD\r\n" +
	"END:VTIMEZONE\r\n"
