// Package parsefecha interpreta las fechas que Moodle muestra en el calendario.
//
// Formatos reales, medidos sobre un volcado del calendario de PEDCO:
//
//	"Hoy , 18:00"
//	"viernes, 15 mayo , 23:55"
//	"viernes, 15 mayo , 18:00 » 20:00"
//
// Tres trampas:
//
//  1. NO trae año. Un "15 enero" leído en diciembre es del año que viene, y un
//     "20 diciembre" leído en enero es del año pasado. Se resuelve eligiendo el
//     año que deje la fecha más cerca de la referencia.
//  2. Hay formas relativas ("Hoy", "Mañana", "Ayer").
//  3. Los rangos usan «»», y el lado derecho normalmente es solo una hora.
//
// El nombre del día ("viernes") se descarta: es redundante y no se valida
// contra la fecha porque Moodle ya la da resuelta.
package parsefecha

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Rango es lo que Moodle informó. Fin queda en cero si no había rango.
type Rango struct {
	Inicio time.Time
	Fin    time.Time
}

var meses = map[string]time.Month{
	"enero": time.January, "febrero": time.February, "marzo": time.March,
	"abril": time.April, "mayo": time.May, "junio": time.June,
	"julio": time.July, "agosto": time.August, "septiembre": time.September,
	"setiembre": time.September, // variante usada en el Río de la Plata
	"octubre": time.October, "noviembre": time.November, "diciembre": time.December,
}

var (
	reHora = regexp.MustCompile(`\b(\d{1,2}):(\d{2})\b`)
	reDia  = regexp.MustCompile(`\b(\d{1,2})\b`)
)

// Parse interpreta texto usando ref como "ahora".
//
// ref se inyecta en vez de llamar a time.Now() adentro por dos razones: sin eso
// la función no es testeable, y hace falta para resolver "Hoy" y el año que
// Moodle no manda. El resultado queda en la zona de ref.
func Parse(texto string, ref time.Time) (Rango, error) {
	limpio := normalizar(texto)
	if limpio == "" {
		return Rango{}, fmt.Errorf("texto de fecha vacío")
	}

	izq, der, hayRango := partirRango(limpio)

	inicio, err := unMomento(izq, ref)
	if err != nil {
		return Rango{}, err
	}
	if !hayRango {
		return Rango{Inicio: inicio}, nil
	}

	// El lado derecho suele ser solo la hora ("18:00 » 20:00"). Si además trae
	// fecha (un rango que cruza el día), se parsea completo.
	fin, err := unMomento(der, ref)
	if err != nil || !tieneFecha(der) {
		h, m, okHora := hora(der)
		if !okHora {
			// Hay «»» pero el lado derecho no tiene hora usable: se devuelve
			// el inicio en vez de fallar, que es el dato que importa.
			return Rango{Inicio: inicio}, nil
		}
		fin = time.Date(inicio.Year(), inicio.Month(), inicio.Day(), h, m, 0, 0, inicio.Location())
		// Un rango que "termina antes de empezar" cruzó la medianoche.
		if fin.Before(inicio) {
			fin = fin.AddDate(0, 0, 1)
		}
	}
	return Rango{Inicio: inicio, Fin: fin}, nil
}

func unMomento(s string, ref time.Time) (time.Time, error) {
	h, m, okHora := hora(s)
	if !okHora {
		h, m = 0, 0
	}

	switch {
	case strings.Contains(s, "hoy"):
		return enDia(ref, 0, h, m), nil
	case strings.Contains(s, "manana"):
		return enDia(ref, 1, h, m), nil
	case strings.Contains(s, "ayer"):
		return enDia(ref, -1, h, m), nil
	}

	mes, okMes := buscarMes(s)
	if !okMes {
		return time.Time{}, fmt.Errorf("no encontré mes ni forma relativa en %q", s)
	}
	dia, okDia := buscarDia(s, h, m, okHora)
	if !okDia {
		return time.Time{}, fmt.Errorf("no encontré día del mes en %q", s)
	}

	return conAnioMasCercano(dia, mes, h, m, ref), nil
}

// conAnioMasCercano elige entre el año de ref, el anterior y el siguiente, el
// que deje la fecha más cerca de ref. Así "enero" leído en diciembre cae en el
// año siguiente y "diciembre" leído en enero en el anterior, sin reglas ad hoc.
func conAnioMasCercano(dia int, mes time.Month, h, m int, ref time.Time) time.Time {
	mejor := time.Time{}
	var mejorDist time.Duration
	for _, off := range []int{0, 1, -1} {
		c := time.Date(ref.Year()+off, mes, dia, h, m, 0, 0, ref.Location())
		d := c.Sub(ref)
		if d < 0 {
			d = -d
		}
		if mejor.IsZero() || d < mejorDist {
			mejor, mejorDist = c, d
		}
	}
	return mejor
}

func enDia(ref time.Time, offset, h, m int) time.Time {
	d := ref.AddDate(0, 0, offset)
	return time.Date(d.Year(), d.Month(), d.Day(), h, m, 0, 0, ref.Location())
}

// normalizar baja a minúsculas, saca acentos y colapsa espacios. Moodle emite
// "Hoy , 18:00" con espacio antes de la coma, así que no se puede confiar en
// la puntuación como separador.
func normalizar(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.NewReplacer(
		"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n",
		",", " ",
	).Replace(s)
	return strings.Join(strings.Fields(s), " ")
}

// partirRango busca solo «»», que es el único separador que Moodle emite. No se
// acepta "-": partiría mal una fecha escrita 15-05-2026.
func partirRango(s string) (izq, der string, hay bool) {
	i := strings.Index(s, "»")
	if i < 0 {
		return s, "", false
	}
	izq = strings.TrimSpace(s[:i])
	der = strings.TrimSpace(s[i+len("»"):])
	if izq == "" || der == "" {
		return s, "", false
	}
	return izq, der, true
}

func hora(s string) (h, m int, ok bool) {
	g := reHora.FindStringSubmatch(s)
	if g == nil {
		return 0, 0, false
	}
	h, _ = strconv.Atoi(g[1])
	m, _ = strconv.Atoi(g[2])
	if h > 23 || m > 59 {
		return 0, 0, false
	}
	return h, m, true
}

func buscarMes(s string) (time.Month, bool) {
	for nombre, mes := range meses {
		if strings.Contains(s, nombre) {
			return mes, true
		}
	}
	return 0, false
}

// buscarDia toma el primer número de 1 a 31 que no sea parte de la hora.
func buscarDia(s string, h, m int, hayHora bool) (int, bool) {
	if hayHora {
		s = reHora.ReplaceAllString(s, " ")
	}
	for _, g := range reDia.FindAllStringSubmatch(s, -1) {
		n, err := strconv.Atoi(g[1])
		if err == nil && n >= 1 && n <= 31 {
			return n, true
		}
	}
	return 0, false
}

func tieneFecha(s string) bool {
	if strings.Contains(s, "hoy") || strings.Contains(s, "manana") || strings.Contains(s, "ayer") {
		return true
	}
	_, ok := buscarMes(s)
	return ok
}
