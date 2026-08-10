// Package domain tiene los tipos que cruzan capas.
//
// Hay UN tipo, no cinco. El spec listaba Materia/Unidad/Recurso/ForumPost/Horario,
// pero hoy el único consumidor real es la emisión de .ics de vencimientos: los
// tipos del material se agregan cuando exista quien los lea. El gauntlet ya
// cazó dos campos inventados (Materia.Carrera, Horario.Modalidad) que no tenían
// un solo lector en todo el diseño.
//
// Tampoco hay puertos todavía: un puerto se gana su lugar cuando hay una segunda
// implementación que intercambiar, y por ahora hay una sola de cada cosa.
package domain

import "time"

// Tipo clasifica un vencimiento. Reemplaza los strings con emoji de
// scraper-pedco ("📝 Tarea"): el emoji es presentación, no dominio.
type Tipo string

const (
	Tarea         Tipo = "tarea"         // mod_assign
	Cuestionario  Tipo = "cuestionario"  // mod_quiz
	Examen        Tipo = "examen"        // parcial / examen / recuperatorio
)

// Entrega es un vencimiento del calendario de Moodle: un TP, un cuestionario o
// un parcial. Es lo que decide las notas, y hoy es lo único que NO llega al
// calendario local (el bot lo avisa por Telegram y nada más).
type Entrega struct {
	// MoodleID es el data-event-id. Es la identidad estable y va al UID del
	// .ics, para que la segunda corrida actualice en vez de duplicar.
	MoodleID string

	Materia string // nombre del curso tal como lo da Moodle
	Titulo  string
	Tipo    Tipo

	// Vence es el inicio. Hasta es el fin y queda en cero cuando Moodle no dio
	// un rango, que es el caso de la mayoría de los vencimientos.
	Vence time.Time
	Hasta time.Time

	URL string
}

// TieneRango dice si Moodle informó hora de fin.
func (e Entrega) TieneRango() bool { return !e.Hasta.IsZero() }
