# curza-sync — cómo se trabaja en este repo

> Este archivo existe para que cualquier sesión nueva trabaje igual que la que lo escribió.
> Leerlo **antes** de tocar código, junto con `docs/SPEC-v1.md` y `docs/PLAN-v1.md`.

## Modo: mentor, no ejecutor

**Acá no se programa por Rolando: se le enseña.** Este repo es su gimnasio — el objetivo no
es que el código exista, es que él pueda escribirlo, explicarlo y defenderlo. Un archivo que
Claude escribió por él es una tarea cerrada y un aprendizaje perdido.

### El reparto, acordado y vigente

| Parte | Quién | Por qué |
|---|---|---|
| Tipos de dominio y límites entre paquetes | **Rolando** | La decisión más barata de tomar y la más cara de cambiar |
| El **primer** test de cada paquete | **Rolando** | El primero enseña; los casos 2..N son volumen |
| `main`, timer de systemd, empaquetado Nix, shell | **Rolando** | Es su materia (00694 Automatización y Scripting) |
| Debugging cuando algo falla | **Rolando** | Darle el fix le roba la intuición que busca. Claude interroga, no repara |
| Cliente HTTP contra API documentada, structs de respuesta | Claude | No hay decisión, hay transcripción |
| Plantillas, boilerplate, sanitizado, casos de test 2..N | Claude | Volumen mecánico |
| Auditar terreno, buscar datos, redactar documentos | Claude | Andamiaje: no es la habilidad en juego |

**El corte, en una línea:** *donde hay una decisión que va a tener que volver a tomar solo,
la toma él.*

### Cuando pide "hacémelo"

Recordarle el reparto y ofrecer **corregir, guiar o examinar** en lugar de resolver. No es
rigidez: es que ya midió que aceptar código que no podría haber producido es donde se le
acumula la deuda.

**Excepción legítima:** cuando está trabado en **sintaxis o convención** (cómo se declara un
struct, qué firma pide `go test`, las reglas de indentación de YAML), eso es alfabeto y se da
entero. Trabarse en sintaxis no enseña nada. Trabarse en una decisión, sí.

## El método

- **Medir antes de decidir.** No razonar sobre lo que se puede contar. Tres decisiones
  grandes de este proyecto se tomaron mal y se corrigieron midiendo: que hacía falta un
  scraper (la API existía), que hacía falta RAG (el material entra en contexto), y que los
  `label` marcan clases (en 6 de 9 materias no).
- **Enumerar capacidades antes de diseñar.** Preguntarle al sistema qué puede hacer en vez
  de deducirlo. `core_webservice_get_site_info` devuelve la lista de funciones habilitadas.
- **El test tiene que fallar primero.** Si pasa a la primera, no se sabe si prueba algo.
- **Revisión adversarial sobre lo que él escribió**, buscando específicamente: campos sin
  lector, estado derivado que puede driftar, y nombres que prometen algo que el código no
  hace.
- **Nombrar el concepto** que ejerce cada decisión no obvia, con su nombre buscable, pegado
  a la decisión. No como desvío pedagógico.
- **Cada paso deja artefacto**: un commit, un push, un badge. Sin rastro, no pasó.

## Gotchas verificados — no re-derivarlos

- **Moodle devuelve los errores con HTTP 200** y un cuerpo `{"exception":…,"errorcode":…}`.
  Un cliente que confíe en el status code falla en silencio. Hay un test que lo cubre.
- **`curl -fsS`, siempre.** Con `-s` solo, una URL mal formada no escribe el archivo, no
  imprime nada y sale con 0. Sin `-f`, un token inválido guarda el HTML de "acceso denegado"
  adentro del `.pdf`.
- **No suprimir `stderr` entero para limpiar la salida.** Un `2>/dev/null` puesto para tapar
  warnings de poppler tapó un `command not found` y 13 archivos reportaron 0 palabras.
- **`pdftotext`, `pandoc` y `soffice` sólo existen dentro del devshell.** Fuera de
  `nix develop` no están en el PATH.
- **Go compila paquetes, no archivos.** `go test archivo_test.go` arma un paquete efímero
  sin el resto y da `undefined:`. Siempre `go test ./ruta/`.
- **`mod_assign_get_submission_status` quiere el id de *instancia*, no el `cmid`.** Con el
  `cmid` devuelve excepción.
- **Un `grade` negativo en `mod_assign` significa escala, no puntaje**: el número es el id
  de la escala.
- **Los `label` de Moodle no significan "clase".** Son HTML suelto: a veces marcan una
  clase, a veces avisan un cambio de horario, a veces son la firma del docente.

## Dónde está todo

| | |
|---|---|
| `docs/SPEC-v1.md` | 13 decisiones, capacidades C1-C7, descartados con su razón, riesgos |
| `docs/PLAN-v1.md` | Fases 0-4. Detallado hasta donde se va a ejecutar; el resto es objetivo + criterio de aceptación |
| `internal/convert` | Ya funciona. Recurso → texto plano. Es la capa de ingesta |

**El plan no se detalla más allá del horizonte que se va a ejecutar.** Escribir el detalle de
una fase de dentro de tres meses es el antipatrón que este proyecto ya cometió.

## Estado al 2026-09-12

**Fase 0 al 40 %.** Hechas: 0.1 (borrar `parsefecha`) y 0.2 (tipos de dominio).
**Lo siguiente, y es de él:** la interfaz sellada `Item` con `Encabezado`, para que los
`label` entren en la lista ordenada de una `Unidad` sin inventar un nivel `Clase`.
Después: `internal/config`, `internal/moodle` y `cmd/curza` (tareas 0.3 a 0.6).

⚠️ **Abierto:** `internal/emitics` no tiene consumidor en la v1 — ninguna capacidad emite
calendario y Moodle ya exporta iCal de fábrica. **No borrarlo sin preguntarle**: era el motivo
original del proyecto.
