package convert

import "os"

func mkdirTemp() (string, error) { return os.MkdirTemp("", "curza-convert-*") }
func removeAll(dir string)       { _ = os.RemoveAll(dir) }

func leer(ruta string) (string, error) {
	b, err := os.ReadFile(ruta)
	return string(b), err
}
