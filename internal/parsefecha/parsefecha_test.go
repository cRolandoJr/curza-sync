package parsefecha

import (
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	ref := time.Date(2026, time.March, 3, 0, 0, 0, 0, time.Local)

	casos := []struct {
		nombre string
		texto  string
		quiero time.Time
	}{
		{
			nombre: "fecha con mes y hora",
			texto:  "viernes, 15 mayo , 23:55",
			quiero: time.Date(2026, time.May, 15, 23, 55, 0, 0, time.Local),
		},
		{
			nombre: "fecha con mes y rango tome hora de apertura",
			texto:  "viernes, 14 agosto , 18:00 » 20:00",
			quiero: time.Date(2026, time.August, 14, 18, 0, 0, 0, time.Local),
		},
		{
			nombre: "dia y hora",
			texto:  "Mañana, 20:00",
			quiero: time.Date(2026, time.March, 4, 20, 0, 0, 0, time.Local),
		},
		{
			nombre: "dia y hora",
			texto:  "hoy, 18:30",
			quiero: time.Date(2026, time.March, 3, 18, 30, 0, 0, time.Local),
		},
	}

	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			got, err := Parse(c.texto, ref)
			if err != nil {
				t.Fatalf("no esperaba error, vino: %v", err)
			}
			if !got.Inicio.Equal(c.quiero) {
				t.Errorf("Inicio = %v, quiero: %v", got.Inicio, c.quiero)
			}
		})
	}

	/*got, err := Parse("viernes, 15 mayo , 23:55", ref)
	if err != nil {
		t.Fatalf("no esperaba error, vino: %v", err)
	}

	quiero := time.Date(2026, time.May, 15, 23, 55, 0, 0, time.Local)

	if !got.Inicio.Equal(quiero) {
		t.Errorf("Inicio = %v, quiero %v", got.Inicio, quiero)
	}*/
}
