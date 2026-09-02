# curza-sync v1 — Plan de implementación

**Spec:** `docs/SPEC-v1.md` (12 decisiones, 6 capacidades). El plan argumenta desde el spec;
se leen juntos.

**Objetivo:** un binario Go que baja de Moodle las materias del alumno y deja en su disco un
tablero de entregas y un markdown por unidad, sin que nada salga de su máquina.

**Arquitectura:** núcleo (cliente Moodle · dominio · almacenamiento · render) con adaptadores
intercambiables. La CLI de la v1 y la UI web de la v2 son dos entradas al mismo núcleo.

**Stack:** Go 1.25 · sin dependencias externas más allá de `golang.org/x/text` · `pdftotext`,
`pandoc` y `soffice` como binarios del sistema (vía `internal/convert`, ya escrito).

---

## Cómo se usa este plan

**Divergencia consciente con la skill de planes.** El formato estándar trae el código escrito
en cada paso, para que lo ejecute un agente. Acá el reparto es otro:

| | Formato |
|---|---|
| **Tareas de Rolando** | objetivo, criterio de aceptación y las decisiones a tomar. **Sin código.** Claude entra después como revisor y examinador |
| **Tareas de Claude** | código completo |

**Y el plan se detalla hasta el horizonte que se va a ejecutar.** Las fases 0 y 1 están
completas. Las fases 2-4 son objetivo + criterio de aceptación, y se detallan al llegar.
Escribir el detalle de algo que se hará en noviembre es exactamente el antipatrón que este
repo tiene documentado.

---

## Restricciones globales

- **Go 1.25**, módulo `github.com/cRolandoJr/curza-sync`
- **Moodle devuelve los errores con HTTP 200.** Todo llamado a la API debe inspeccionar el
  cuerpo antes de asumir éxito. *(Spec: riesgo transversal; medido el 2026-09-02.)*
- **Nada de credenciales en el repo.** `~/.config/curza/config.toml` con permisos `0600`
- **Nada de material de cátedra en el repo.** El `.gitignore` ya cubre `material/`, `dump/`
- **Toda fixture se sanea antes de commitear**: `sesskey`, `userid`, `token`, nombres de
  terceros. El repo es público
- **El CI tiene que quedar verde en cada commit** (`go vet` + `go build` + `go test`)
- Umbral bibliografía: **50.000 palabras** (spec, decisión 2)

---

## Estructura de archivos

```
cmd/curza/main.go            ← Rolando · composition root, subcomandos
internal/
  config/config.go           ← Claude  · leer/escribir config.toml 0600
  moodle/
    client.go                ← Claude  · cliente RPC + detección de error HTTP-200
    courses.go               ← Claude  · get_users_courses, get_course_contents
    assign.go                ← Claude  · get_assignments, get_submission_status
    quiz.go                  ← Claude  · get_quizzes_by_courses  (fase 1)
  domain/domain.go           ← Rolando · Unidad, Entrega, Recurso, Materia
  store/store.go             ← Claude  · estado.json + JSON crudo por sección
  render/semaforo.go         ← Claude  · el tablero de entregas
  render/unidad.go           ← Claude  · el markdown por unidad  (fase 2)
  convert/                   ← YA EXISTE, no se toca
```

**Se borra:** `internal/parsefecha` (spec, riesgo 5 resuelto).

⚠️ **A decidir en la fase 2:** `internal/emitics` (escritura de `.ics`) **tampoco tiene
consumidor en la v1** — ninguna de las 6 capacidades emite calendario, y Moodle ya exporta
iCal de fábrica. No lo borro sin que lo decidas: era el motivo original del proyecto.

---

# FASE 0 · El esqueleto

**Entregable:** `curza courses` lista tus 9 materias leyendo el token del config.
**Por qué primero:** prueba autenticación, cliente y composition root de una sola vez.

---

### Tarea 0.1 · Borrar `parsefecha` — **Rolando**

**Archivos:** eliminar `internal/parsefecha/`

- [ ] **Paso 1:** `git rm -r internal/parsefecha`
- [ ] **Paso 2:** `go build ./... && go test ./...` — tiene que seguir compilando (nadie lo importa)
- [ ] **Paso 3:** commit `chore: borrar parsefecha, la API devuelve timestamps`
- [ ] **Paso 4:** confirmar en la pestaña Actions que el CI quedó **verde**

> El test que escribiste sobre él cumplió su función y queda en la historia de git. No se
> mantiene código sin consumidor por nostalgia.

---

### Tarea 0.2 · Los tipos de dominio — **Rolando** ← *la decisión más formativa del plan*

**Archivos:** crear `internal/domain/domain.go` (el actual se reescribe)

**Objetivo:** definir los tipos que representan lo que el sistema manipula. **Nada más.**

**Lo que hay que decidir, y son decisiones, no sintaxis:**

1. ¿Qué tipos existen? Candidatos según el spec: `Materia`, `Unidad`, `Recurso`, `Entrega`.
   **¿Los cuatro se ganan el lugar hoy, o alguno no tiene lector todavía?**
2. Por cada tipo, qué campos. **La decisión 12 del spec dice qué va al `.md`** — ese es tu
   consumidor real. Un campo sin lector no entra.
3. ¿`Unidad` contiene `[]Recurso`, o guarda ids y alguien más resuelve? Es la diferencia entre
   un árbol y un grafo, y cambia cómo se testea.
4. ¿Dónde vive el texto convertido de un recurso: en `Recurso`, o aparte? Un `Recurso` con
   200.000 palabras adentro es un tipo que no se puede pasar por valor con tranquilidad.

**Criterio de aceptación:**
- `go build ./...` pasa
- **Cada campo se puede justificar señalando quién lo lee**, hoy, en la decisión 12
- Sin punteros/interfaces "por si acaso": no hay segunda implementación que intercambiar

- [ ] **Paso 1:** escribir los tipos
- [ ] **Paso 2:** `go build ./...`
- [ ] **Paso 3:** commit `domain: tipos de la v1`
- [ ] **Paso 4:** pedirle a Claude la revisión adversarial: *"atacá estos tipos, buscá campos sin lector"*

> **Concepto: el modelo de dominio es la decisión más barata de tomar y la más cara de
> cambiar.** Todo lo de abajo se escribe contra estos tipos. El 10-ago esta pieza la escribió
> Claude y quedó anotada como deuda de drill — esta vez es tuya.

---

### Tarea 0.3 · `internal/config` — **Claude**

**Archivos:** crear `internal/config/config.go`, `internal/config/config_test.go`

**Produce:** `config.Load() (Config, error)`, `config.Save(Config) error`,
`Config{Site, Token, UserID, ClasesDir string; UmbralBiblio int}`

- [ ] **Paso 1: test que falla** — `TestSaveCreaArchivoCon0600`: guardar, y verificar que
      `os.Stat().Mode().Perm() == 0600`
- [ ] **Paso 2:** correr, verificar FAIL
- [ ] **Paso 3:** implementar con `os.WriteFile(path, data, 0600)` y `os.MkdirAll(dir, 0700)`
- [ ] **Paso 4:** correr, verificar PASS
- [ ] **Paso 5:** commit `config: leer y escribir config.toml con 0600`

---

### Tarea 0.4 · `internal/moodle` — el cliente RPC — **Claude**

**Archivos:** crear `internal/moodle/client.go`, `internal/moodle/client_test.go`

**Produce:**
```go
type Client struct{ /* ... */ }
func New(site, token string) *Client
func (c *Client) Call(ctx context.Context, fn string, params url.Values, out any) error
```

**Consume:** nada.

**El punto entero de esta tarea:** Moodle responde los errores con **HTTP 200** y un cuerpo
`{"exception":..., "errorcode":..., "message":...}`. Un cliente que confíe en el status code
falla en silencio. Esto se testea **primero**.

- [ ] **Paso 1: el test que importa, y falla**

```go
func TestCallDetectaErrorConHTTP200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK) // Moodle contesta 200 igual
		io.WriteString(w, `{"exception":"moodle_exception","errorcode":"invalidtoken","message":"Token inválido"}`)
	}))
	defer srv.Close()

	var out []struct{}
	err := New(srv.URL, "token-malo").Call(context.Background(), "core_enrol_get_users_courses", nil, &out)
	if err == nil {
		t.Fatal("quiero error y vino nil: el cliente se tragó un error con HTTP 200")
	}
	if !strings.Contains(err.Error(), "invalidtoken") {
		t.Errorf("el error debería nombrar el errorcode, vino: %v", err)
	}
}
```

- [ ] **Paso 2:** `go test ./internal/moodle/ -run TestCallDetectaError -v` → FAIL
- [ ] **Paso 3: implementar**

```go
package moodle

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	endpoint string
	token    string
	http     *http.Client
}

func New(site, token string) *Client {
	return &Client{
		endpoint: strings.TrimSuffix(site, "/") + "/webservice/rest/server.php",
		token:    token,
		http:     &http.Client{Timeout: 90 * time.Second},
	}
}

// APIError es el error que Moodle devuelve con HTTP 200.
type APIError struct {
	Exception string `json:"exception"`
	ErrorCode string `json:"errorcode"`
	Message   string `json:"message"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("moodle %s: %s", e.ErrorCode, e.Message)
}

func (c *Client) Call(ctx context.Context, fn string, params url.Values, out any) error {
	form := url.Values{}
	for k, vs := range params {
		for _, v := range vs {
			form.Add(k, v)
		}
	}
	form.Set("wstoken", c.token)
	form.Set("wsfunction", fn)
	form.Set("moodlewsrestformat", "json")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("%s: armando la petición: %w", fn, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%s: %w", fn, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("%s: leyendo la respuesta: %w", fn, err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: http %d", fn, resp.StatusCode)
	}

	// Moodle contesta los errores con 200. Hay que mirar el cuerpo.
	var apiErr APIError
	if json.Unmarshal(body, &apiErr) == nil && apiErr.Exception != "" {
		return fmt.Errorf("%s: %w", fn, &apiErr)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("%s: decodificando: %w", fn, err)
	}
	return nil
}
```

- [ ] **Paso 4:** `go test ./internal/moodle/ -v` → PASS
- [ ] **Paso 5:** segundo test, respuesta válida en forma de array

```go
func TestCallDecodificaUnArray(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `[{"id":10337,"shortname":"curza-2026-automatizacion"}]`)
	}))
	defer srv.Close()

	var out []struct {
		ID        int    `json:"id"`
		ShortName string `json:"shortname"`
	}
	if err := New(srv.URL, "t").Call(context.Background(), "core_enrol_get_users_courses", nil, &out); err != nil {
		t.Fatalf("no esperaba error: %v", err)
	}
	if len(out) != 1 || out[0].ID != 10337 {
		t.Errorf("decodificó mal: %+v", out)
	}
}
```

- [ ] **Paso 6:** `go test ./... ` → PASS
- [ ] **Paso 7:** commit `moodle: cliente RPC que detecta los errores con HTTP 200`

---

### Tarea 0.5 · `moodle/courses.go` — **Claude**

**Archivos:** crear `internal/moodle/courses.go`

**Consume:** `Client.Call`
**Produce:** `func (c *Client) Cursos(ctx context.Context, userID int) ([]Curso, error)` con
`Curso{ID int; ShortName, FullName string; Categoria int; StartDate int64}`

- [ ] **Paso 1:** escribir el tipo y el método usando `core_enrol_get_users_courses`
- [ ] **Paso 2:** test con `httptest` y una fixture **saneada** (ids reales están bien, tokens no)
- [ ] **Paso 3:** `go test ./...` → PASS
- [ ] **Paso 4:** commit `moodle: listar las materias del usuario`

---

### Tarea 0.6 · `cmd/curza` y el subcomando `courses` — **Rolando**

**Archivos:** crear `cmd/curza/main.go`

**Objetivo:** el composition root. Lee la config, arma el cliente, ejecuta el subcomando.

**Decisiones tuyas:**
1. Cómo se parsean los subcomandos. `flag` de la stdlib alcanza — **¿hace falta algo más?**
2. ¿Qué pasa si no hay config? ¿Falla, o guía al usuario a crearla?
3. Dónde termina el `main` y empieza el paquete. **Regla: el `main` decide, los paquetes hacen.**

**Criterio de aceptación:** `go run ./cmd/curza courses` imprime tus 9 materias.

- [ ] **Paso 1:** escribirlo
- [ ] **Paso 2:** correrlo contra PEDCO de verdad y ver las 9 materias
- [ ] **Paso 3:** commit `cmd/curza: subcomando courses`
- [ ] **Paso 4:** confirmar CI verde

> **Concepto: composition root.** "Qué site, qué token, qué userid" son decisiones del
> entorno. Viven acá y no dentro de `internal/`. Es la misma razón por la que `parsefecha`
> recibía `ref` en vez de llamar a `time.Now()`.

**🔵 CHECKPOINT — fin de fase 0.** No seguir sin: CI verde, `curza courses` andando contra
PEDCO, y la revisión adversarial de los tipos de dominio hecha.

---

# FASE 1 · C1 + C3 · El semáforo

**Entregable:** `curza todo` imprime las 37 entregas y los 11 parciales de las 9 materias, con
fecha, estado (`sin-intento` / `new` / `submitted`) y nota.

**Por qué esta fase antes que el resto:** es la capacidad de mayor valor diario y la única que
ya demostró servir — en su primera corrida encontró un TP vencido en estado borrador.

### Tarea 1.1 · `moodle/assign.go` — **Claude**
`mod_assign_get_assignments` (todas las materias de una) + `mod_assign_get_submission_status`
por entrega. **Gotcha medido:** `submission_status` quiere el **id de instancia**, no el
`cmid`; pasarle el `cmid` devuelve excepción.

### Tarea 1.2 · `moodle/quiz.go` — **Claude**
`mod_quiz_get_quizzes_by_courses`. Los parciales entran al mismo tablero.

### Tarea 1.3 · El primer test de `assign` — **Rolando**
Con una fixture JSON real y saneada. Es tu primer test contra una respuesta de API: el patrón
lo aprendiste ayer, el dominio es nuevo.

### Tarea 1.4 · `render/semaforo.go` — **Claude**
Ordenado por fecha de vencimiento, no por materia. Lo vencido primero.

### Tarea 1.5 · `curza todo` — **Rolando**
Cablearlo en el `main`.

**🔵 CHECKPOINT.** `curza todo` te dice qué debés. Desde acá la herramienta ya sirve todos los
días, aunque no exista nada más.

---

# FASE 2 · C2 · Un markdown por unidad

**Entregable:** `curza pull` deja `~/Clases/<cuatri>/<materia>/<Unidad>.md` con lo que define
la decisión 12 del spec, más los originales en `material/`.

**Incluye:** descarga con reintento, `store` (JSON crudo + `estado.json` con hashes para no
re-bajar), separación cátedra/bibliografía por el umbral de 50.000 palabras, y la decisión
pendiente sobre `emitics`.

**Criterio de aceptación:** abrís el `.md` de una unidad de Automatización y tenés la clase
entera, sin entrar a PEDCO.

---

# FASE 3 · C5 + C6 · El asistente

**Entregable:** `curza ask <materia> <unidad> "pregunta"` responde citando el archivo y la
línea; y `curza practica <materia>` genera preguntas en el estilo de la cátedra.

**Ya verificado (2026-09-02):** el material cubre las preguntas realmente falladas, con cita
a archivo y línea; y hay 48 preguntas de examen reales en 6 formatos como corpus de estilo.

**A decidir al llegar:** qué proveedor de modelo, cómo se configura la clave del usuario
(decisión 9), y cómo se degrada cuando el docente cerró el review del quiz (riesgo 6).

---

# FASE 4 · C4 · Qué cambió

**Entregable:** `curza diff` dice qué subieron o modificaron desde la última corrida.

⚠️ **`core_course_get_updates_since` es la única función del plan que NO se probó.** Antes de
implementar: spike de 10 minutos. Si no devuelve lo que se espera, esta fase se rediseña.

---

## Auto-revisión del plan

- **Cobertura del spec:** C1→fase 1 · C2→fase 2 · C3→fase 1 · C4→fase 4 · C5→fase 3 ·
  C6→fase 3. Decisiones 1,2,8,11,12→fase 2 · 3,9→fase 3 · 5→fase 2 · 6→estructura ·
  7→tarea 0.3 · 10→afecta la v2, no la v1.
- **Huecos conocidos y declarados:** el destino de `emitics` (fase 2) y el spike de C4 (fase 4).
- **Consistencia de tipos:** `Client.Call` se usa igual en 0.5, 1.1 y 1.2. `Config` se consume
  sólo en `cmd/curza`.
