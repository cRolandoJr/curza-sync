package parsefecha

import (
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	ref := time.Date(2026, time.March, 3, 0, 0, 0, 0, time.Local)

	got, err := Parse("viernes, 15 mayo , 23:55", ref)
	if err != nil {
		t.Fatalf("no esperaba error, vino: %v", err)
	}

	quiero := time.Date(2026, time.May, 15, 23, 55, 0, 0, time.Local)

	if !got.Inicio.Equal(quiero) {
		t.Errorf("Inicio = %v, quiero %v", got.Inicio, quiero)
	}
}
