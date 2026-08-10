---
type: Tasks
title: "BRIEFING — Planungs-Primitive im beans-Fork"
description: "Drei Arbeitspakete am beans-Fork, ausgelöst vom Sproutling-Roadmap-Modell: benutzerdefiniertes Frontmatter erhalten und ausgeben, das schlafende Order-Feld anschließen, und die generischen Anteile einer Repo-Werkzeugkette in die CLI ziehen."
tags:
  - tpic
  - beans-planning-primitives
timestamp: 2026-08-10T09:52:22Z
---

# Briefing — Planungs-Primitive im beans-Fork

## Auftrag in einem Satz

beans soll ein Planungsmodell tragen können, das über Titel, Status, Typ und Priorität hinausgeht — heute verliert es jede Angabe, die es nicht selbst kennt.

## Woher der Auftrag kommt

Sproutling stellt seine Roadmap- und Release-Planung auf ein Modell um, in dem beans die einzige Wahrheit ist und ein generiertes `plan.yaml` die Bedienoberfläche darauf. Das Modell steht in `~/dev/sproutling/docs/roadmap-release-planning/DESIGN.md`. Es braucht je Container einen kundenseitigen Satz, eine Klasse, eine Release-Zuordnung und eine Notiz. Klasse und Release lassen sich als Tags abbilden, ein Satz nicht.

Die Alternative wäre, die Sätze in den Body zu schreiben. Der Einwand des PO dagegen ist der Grund für dieses Briefing: **wer nur sichten will, soll das Frontmatter lesen und nicht den Body.** Ein Body trägt Prosa, Ketten, Begründungen — ein Agent, der zwanzig beans überfliegt, zieht sonst zwanzig Aufsätze in seinen Kontext.

## Der gemessene Befund

beans löscht unbekannte Frontmatter-Schlüssel bei jedem `beans update`. Reproduktion in einem separaten Store, kein Repo wird berührt:

````bash
T=$(mktemp -d); mkdir -p "$T/.beans"
ID=$(beans --beans-path "$T/.beans" create probe -t epic --json | python3 -c 'import sys,json;print(json.load(sys.stdin)["bean"]["id"])')
F=$(ls "$T"/.beans/*.md)

python3 - "$F" <<'PY'
import sys
p = sys.argv[1]; t = open(p).read()
i = t.index("\n---\n", 3)                       # Ende des Frontmatter-Blocks
t = t[:i] + '\nrelease: 0-4-1\nklasse: bugfix\ncustomer_value: "Ein Satz."' + t[i:]
open(p, "w").write(t)
PY

beans --beans-path "$T/.beans" update "$ID" --priority high
head -12 "$F"
# Ergebnis: priority: high steht da, release / klasse / customer_value sind ersatzlos weg.
````

`beans list --json` gibt ohnehin nur ein festes Schema aus: `id`, `slug`, `path`, `title`, `status`, `type`, `priority`, `tags`, `parent`, `created_at`, `updated_at`, `etag`. Ein zusätzliches Feld wäre also selbst dann unsichtbar, wenn es die Runde überlebte.

Die Ursache liegt offen: `Parse` (`pkg/bean/bean.go:185`) liest über `github.com/adrg/frontmatter v0.2.0` (`go.mod:9`) in das Struct `frontMatter` (`pkg/bean/bean.go:170`), und `Render` (`pkg/bean/bean.go:227`) schreibt aus dem Struct `renderFrontMatter` (`pkg/bean/bean.go:212`) zurück. Was in keinem der beiden Structs steht, existiert nach dem ersten Schreibvorgang nicht mehr.

---

## Paket 1 — Benutzerdefiniertes Frontmatter erhalten und ausgeben

**Soll.** Ein beans-Datei-Frontmatter darf beliebige zusätzliche Schlüssel tragen. beans erhält sie über jeden Lese-Schreib-Zyklus, gibt sie in `--json` aus, und die CLI kann sie setzen, ändern und entfernen.

**Berührte Stellen.**

| Stelle | Änderung |
|---|---|
| `pkg/bean/bean.go:136` `Bean` | Feld `Extra map[string]any` mit `yaml:"-" json:"extra,omitempty"` |
| `pkg/bean/bean.go:185` `Parse` | zusätzlich in eine `map[string]any` parsen, die bekannten Schlüssel abziehen, den Rest nach `Extra`. Der Reader wird zweimal gebraucht — vorher puffern. |
| `pkg/bean/bean.go:212` `renderFrontMatter` | ein Struct kann keine dynamischen Schlüssel serialisieren. Auf `yaml.Node` oder eine geordnete Map umstellen: erst die bekannten Felder in der heutigen Reihenfolge, dann die zusätzlichen nach Schlüssel sortiert. |
| `internal/commands/create.go`, `update.go` | `--set key=value` (wiederholbar), `--unset key` (wiederholbar) |
| `internal/commands/list.go` | `--where key=value` als Filter auf zusätzliche Schlüssel |
| `gqlgen.yml` und `pkg/beangraph` | Schema nachziehen, sonst zeigen CLI und GraphQL verschiedene beans |

**Reservierte Schlüssel.** Die bekannten Namen sind gesperrt: `--set title=…` muss mit einem Fehler abbrechen und auf `--title` verweisen, statt ein Schattenfeld anzulegen.

**Wertebereich.** Skalare reichen für den Anwendungsfall (Zeichenkette, Zahl, Wahrheitswert). Listen und Maps sind zulässig, wenn sie ohne Mehraufwand fallen; sie müssen nicht über die CLI setzbar sein, aber sie dürfen beim Round-Trip nicht kaputtgehen.

**Akzeptanz.** Ein Test im Repo, nicht eine Vorführung:

1. Round-Trip-Test: eine Datei mit drei zusätzlichen Schlüsseln wird geparst, gerendert und ist byte-identisch bis auf `updated_at`.
2. Update-Test: `beans update <id> --priority high` erhält alle zusätzlichen Schlüssel. **Umkehrmutation:** wird die Extra-Übernahme in `Render` entfernt, muss genau dieser Test rot werden.
3. JSON-Test: `beans list --json` enthält die zusätzlichen Schlüssel unter `extra`.
4. Kollisionstest: `--set status=done` bricht mit Fehler ab.
5. Reihenfolge-Test: zwei Läufe von `Render` auf demselben bean erzeugen dieselbe Schlüsselreihenfolge — sonst produziert jeder Schreibvorgang Diff-Rauschen.

**Risiko, das benannt gehört.** Das Datenformat wird von mindestens fünf Stores benutzt: `lean-stack`, `sproutling`, `okf-tools`, `plug-in_VC-Search` und die Worktrees. Ein **älteres** Binary, das eine Datei mit zusätzlichen Schlüsseln schreibt, löscht sie genau so, wie oben gemessen — der Datenverlust wandert dann vom Format in die Versionsstände. Die Freigabe gehört deshalb an einen Versionssprung, und `beans version` sollte die Fähigkeit erkennbar machen.

---

## Paket 2 — `Bean.Order` anschließen

**Befund.** Das Feld existiert bereits: `pkg/bean/bean.go:154`, kommentiert als „fractional index string for manual sorting". Es überlebt `Parse` und `Render`. Aber **kein Kommando schreibt es**, und `beans list --sort` kennt es nicht (`created`, `updated`, `status`, `priority`, `id`). Die Fähigkeit steckt im Format und ist an der CLI nicht angeschlossen.

Das ist der Grund, warum Sproutling für den Horizont seiner Roadmap auf `priority` mit fünf Stufen ausweichen muss, obwohl es eine echte Reihenfolge bräuchte.

**Soll.**

````bash
beans list --sort order                      # sortiert nach fractional index
beans order <id> --after <id>                # setzt den Index zwischen zwei Nachbarn
beans order <id> --before <id>
beans order <id> --first | --last
beans create ... --order <wert>              # optional, für Import
````

**Fractional Index.** Kein ganzzahliger Rang. Zwischen zwei Nachbarn wird ein Wert erzeugt, der lexikografisch dazwischen liegt, damit ein Umsortieren **eine** Datei schreibt und nicht alle. Genau dafür ist ein fractional index da, und genau das war der Grund, warum ein Tag `rang-07` je Container in Sproutling verworfen wurde.

**Entschieden am 2026-08-10: `order` gilt je Elter.** Zwei Kinder verschiedener Milestones stehen in keiner gemeinsamen Reihenfolge; `--after`/`--before` verlangen denselben Elter, `--first`/`--last` beziehen sich auf die Geschwisterliste, und `--sort order` sortiert innerhalb der Elterngruppe. Der Hilfetext benennt den Geltungsbereich (R-12).

**Akzeptanz.** Ein Test, der ein Element von Position 5 auf Position 2 zieht und prüft, dass genau **eine** bean-Datei geschrieben wurde. Ohne diesen Test ist die Eigenschaft, um die es geht, nicht belegt.

---

## Paket 3 — Was aus der Repo-Werkzeugkette in die CLI gehört

Sproutling baut gerade ein Verb `just roadmap` mit sieben Aktionen. Drei davon sind nicht Sproutling-spezifisch, sondern beschreiben, was einem Issue-Tracker fehlt. Sie gehören hierher, nicht in ein Repo-Skript.

**3a — `beans apply <datei>`, deklarativ.** Der Gegenpart zu einem Export: eine YAML-Datei beschreibt den Soll-Zustand einer Menge von beans (Elter, Tags, Priorität, `order`, zusätzliche Schlüssel), und `beans apply` gleicht ab. Mit `--dry-run` als Vorschau, die jede Einzeländerung als Kommando zeigt. Nie anfassen darf es `status`, `title` und den Body — die kommen aus der Arbeit, nicht aus der Planung; welche Felder `apply` schreiben darf, gehört als feste Liste in die Hilfe.

Das ersetzt `just roadmap apply` vollständig und ist für jedes Repo nützlich, das seine Planung in einer Datei bearbeitet.

**3b — Aggregate im JSON.** `beans list --json` ist heute flach: jedes bean einzeln, Beziehungen nur über `parent`. Jede Auswertung rechnet danach dasselbe nach — Nachfahren sammeln, offene Blätter zählen, nach Tag gruppieren. Gebraucht wird eine Ausgabe, die das mitliefert:

````bash
beans list --json --rollup          # je Container: Zahl der offenen Nachfahren, je Status
beans tree --json                   # geschachtelt statt flach
````

Das ist der rechenintensive Teil von `just roadmap plan` und in jedem Repo derselbe Code.

**3c — Filtern auf zusätzlichen Schlüsseln.** Fällt mit Paket 1 zusammen: `beans list --where release=0-4-1`. Ohne das bleibt jedes Custom-Feld eine Angabe, die man nur lesen und nicht suchen kann.

**Was ausdrücklich NICHT hierher gehört.** Die Datei-Kollisionsprüfung zwischen Containern (sie liest Pfade aus bean-Bodies und ist Sproutling-Konvention), die Release-Note-Erzeugung, das Tag-Gate und die Klassen-Regeln des Sproutling-Modells. Diese bleiben Repo-Skripte. Die Trennlinie: was über die **Struktur** eines beans-Baums aussagt, gehört in die CLI; was über die **Bedeutung** der Inhalte urteilt, bleibt im Repo.

---

## Anforderungen

Nummerierte Fassung der Sollsätze oben — die Referenz, gegen die die beans ihre Abdeckung nachweisen (`_Requirements: R-##_` im Blatt-Body). Keine neue Aussage, nur eine adressierbare.

**R-01 — Unbekannte Frontmatter-Schlüssel überleben den Lesepfad.** `Parse` legt jeden Schlüssel, den es nicht selbst kennt, in `Bean.Extra` ab, statt ihn zu verwerfen.

**R-02 — Der Schreibpfad gibt sie deterministisch zurück.** `Render` schreibt die zusätzlichen Schlüssel wieder aus; zwei Läufe auf demselben bean erzeugen dieselbe Reihenfolge, und ein Round-Trip ist byte-identisch bis auf `updated_at`.

**R-03 — Die CLI kann zusätzliche Schlüssel setzen und entfernen.** `--set key=value` und `--unset key`, wiederholbar, an `create` und `update`.

**R-04 — Reservierte Namen sind gesperrt.** `--set` auf einen bekannten Schlüssel bricht mit einem Fehler ab, der auf das native Flag verweist, statt ein Schattenfeld anzulegen.

**R-05 — `beans list --json` gibt sie aus.** Die zusätzlichen Schlüssel erscheinen unter `extra`.

**R-06 — Sie sind filterbar.** `beans list --where key=value` filtert auf zusätzliche Schlüssel.

**R-07 — GraphQL zeigt dieselben beans wie die CLI.** Schema und Resolver tragen die zusätzlichen Schlüssel mit.

**R-08 — Die Fähigkeit ist erkennbar.** `beans version` macht sichtbar, ob das Binary zusätzliche Schlüssel erhält — der Datenverlust wandert sonst unbemerkt in die Versionsstände.

**R-09 — `beans list --sort order` sortiert nach dem fractional index.** Der bereits deklarierte, aber nicht angebotene Sortierschlüssel wird angeboten.

**R-10 — `beans order <id>` setzt die Position relativ zu den Geschwistern.** `--after <id>`, `--before <id>`, `--first`, `--last`; ein Umsortieren schreibt genau **eine** bean-Datei.

**R-11 — `beans create --order <wert>` nimmt einen expliziten Index.** Für den Import einer bestehenden Reihenfolge.

**R-12 — `order` gilt je Elter, und der Hilfetext sagt es.** Geschwister unter demselben `parent` bilden eine Reihenfolge; zwei Kinder verschiedener Eltern stehen in keiner gemeinsamen. PO-Entscheidung vom 2026-08-10.

## Reihenfolge

Paket 1 zuerst und allein — es ändert das Datenformat, und alles andere setzt darauf auf. Paket 2 ist davon unabhängig und kann parallel laufen. Paket 3a und 3c setzen Paket 1 voraus; 3b ist unabhängig.

## Referenzen

````
pkg/bean/bean.go:136   type Bean struct
pkg/bean/bean.go:154   Order string  — deklariert, von keinem Kommando geschrieben
pkg/bean/bean.go:170   type frontMatter struct        — Lesepfad, verwirft Unbekanntes
pkg/bean/bean.go:185   func Parse
pkg/bean/bean.go:212   type renderFrontMatter struct  — Schreibpfad, verwirft Unbekanntes
pkg/bean/bean.go:227   func (b *Bean) Render
go.mod:9               github.com/adrg/frontmatter v0.2.0
internal/commands/     create.go · update.go · list.go · check.go
gqlgen.yml             GraphQL-Schema

~/dev/sproutling/docs/roadmap-release-planning/DESIGN.md   das auslösende Modell
~/dev/sproutling/docs/roadmap-release-planning/plan.yaml   die Datei, die entstehen soll
````
