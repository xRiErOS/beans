# Lessons Learned — beans-src Supervisor-Sessions

Append-only. Je Eintrag: **Symptom (Fundort) · Fix · Forward-Guard**. Ein Eintrag ohne Forward-Guard ist unvollständig.

## Session 2026-07-17/18 — roadmap Feature-Nesting-Fix (Epic beans-en7i, PR hmans/beans#207)

### LL-01 · Backend-Builds brauchen dist-Stub
- **Symptom (Fundort):** `command go test ./internal/commands/` failt beim Setup — `internal/web/embed.go: pattern dist/*: no matching files found`. Kein Package kompiliert, das `internal/web` transitiv importiert, solange das Frontend nicht gebaut ist.
- **Fix:** gitignored Stub `internal/web/dist/index.html` anlegen (`mkdir -p internal/web/dist && printf '%s' '<!doctype html>' > internal/web/dist/index.html`). Baseline-Worktrees (origin/main) brauchen ihn separat.
- **Forward-Guard:** In Preflight/Dispatch-Prompts verankern: jeder Backend-only-Bau/Test setzt den dist-Stub voraus; für jeden neuen (Temp-)Worktree neu anlegen.

### LL-02 · `go` ist shell-shadowed
- **Symptom (Fundort):** `go build` löst git-pull-Rauschen aus — eine Shell-Funktion überschreibt `go`.
- **Fix:** überall `command go` (build/test/vet/fmt).
- **Forward-Guard:** CLAUDE.md + jeder Agent-Dispatch: „IMMER `command go`". Bereits als bekannte devd-Falle dokumentiert — gilt repo-übergreifend.

### LL-03 · `beans create`-Prefix kommt aus cwd-`.beans.yml`, nicht aus `--beans-path`
- **Symptom (Fundort):** bean bekam Fremd-Prefix `bt-` statt `beans-`, weil cwd (nach Bash-Reset) im beans-tui-Repo lag und `--beans-path` nur das Datendir setzt — der Prefix stammt aus der cwd-gefundenen `.beans.yml`.
- **Fix:** für `create` immer `cd <ziel-repo>` ODER `--config <repo>/.beans.yml`. `--beans-path` allein genügt nur für read/update-by-id.
- **Forward-Guard:** §6.9 (Prefix sofort nach create prüfen). Regel: `create` nie aus fremdem cwd nur mit `--beans-path`.

### LL-04 · Reviewer-Verdict muss genau ein Token sein
- **Symptom (Fundort):** Reviewer T3 schrieb erste Zeile `APPROVED`, revidierte am Ende auf `CHANGES_REQUIRED` — maschinelle Auswertung der ersten Zeile wäre falsch gewesen.
- **Fix:** Substanz gewertet, nicht die erste Zeile.
- **Forward-Guard:** Reviewer-Prompt: „Zeile 1 = GENAU EIN Token, vorab entscheiden, kein Rückzug." Ab T4 angewendet, hielt.

### LL-05 · gofmt-Gate nie auf vorbestehend-unclean Dateien
- **Symptom (Fundort):** Plan-gofmt-Gate umfasste `roadmap_test.go`, das schon auf origin/main gofmt-unclean war → Gate immer rot, ohne eigenes Verschulden; Versuchung, Upstream-Code umzuformatieren (unerwünschte PR-Churn).
- **Fix:** Gate auf die geänderte Datei scopen (`roadmap.go`); vorbestehende Upstream-Formatierung nicht anfassen.
- **Forward-Guard:** gofmt-Gates auf geänderte Dateien/Hunks scopen. Vor Plan: `git show origin/main:<file> | gofmt -l` prüfen.

### LL-06 · Plan-Kommandos gegen echtes Repo verifizieren
- **Symptom (Fundort):** Plan nutzte `go build -o ... .` (kein root main.go → `./cmd/beans`) und `beans init -y` (kein `-y`-Flag; init schreibt ins cwd).
- **Fix:** ERRATA durch Implementer, korrigiert; PRELUDE-2 ins T6-bean.
- **Forward-Guard:** Planner MUSS Build-/Init-/CLI-Kommandos gegen das echte Repo (mise.toml, cmd/, --help) prüfen, nicht annehmen.

### LL-07 · Mutations-Probe deckt Coverage-Lücken auf grünem Suite auf
- **Symptom (Fundort):** T3 Milestone-Inclusion-Zeile `|| len(group.Features) > 0` war load-bearing, aber ungetestet — Zeile zurückdrehen ließ KEINEN Test failen (alle Feature-Tests hatten immer ein Epic-Geschwister).
- **Fix:** CHANGES_REQUIRED-Runde, Test `milestone-direct-feature` ergänzt (failt nachweislich ohne die Zeile).
- **Forward-Guard:** Reviewer prüft jede Guard-/Inclusion-Zeile per Mutation: „Zeile brechen → mindestens ein Test MUSS failen." Grüne Suite ≠ getestete Zeile.

### LL-08 · Real-Repo-Smoke fängt, was Unit-Tests strukturell nicht sehen
- **Symptom (Fundort):** B01 — der Fix ließ ein kinderloses offenes Feature komplett aus `roadmap` verschwinden. Kein Unit-Test deckte „Feature ohne Kinder" ab; erst der T6-Smoke gegen das echte beans-tui-Repo (Diff gegen origin/main-Baseline) zeigte `bt-6oyy` als verschwunden.
- **Fix:** B01-Fix (kinderloses Feature → flache Leaf-Zeile) + 5 Tests, eigener Task-Zyklus.
- **Forward-Guard:** Abschluss-Task jedes Epos MUSS einen Real-Repo-Smoke gegen echte Daten laufen (fix-Binary vs. Baseline-Diff), nicht nur Fixtures. Ironie-Check bei „Sichtbarkeits"-Fixes: führt der Fix einen NEUEN Verschwinde-Pfad ein?

### LL-09 · Go-Template-Whitespace ist additiv + kontextblind
- **Symptom (Fundort):** Kosmetik-Einzeiler (führende Leerzeile in `featureGroup`) fixte den Leaf→`####`-Kramp, erzeugte aber im Epic→Feature-Fall 3 Leerzeilen (stapelte mit der eingebauten Epic-Header-Leerzeile). Ein geteilter Template-Block „weiß" seinen Vorkontext nicht.
- **Fix:** revertiert; Kosmetik vom PO als bekannte Einschränkung akzeptiert (byte-identisch-Garantie für feature-lose Ausgabe verbietet sicheren Einzeiler).
- **Forward-Guard:** Template-Whitespace-Änderungen brauchen render-verifizierten Golden-Snapshot + byte-identisch-Diff der unveränderten Pfade — nie ein Blind-Einzeiler. (Verwandt: E8/NBSP-Wordwrap-Falle in beans-tui.)

## Session 2026-07-23 — roadmap TTY-Output (Epic beans-1ec3)

### LL-10 · Ein Forward-Guard, den niemand verdrahtet, ist keiner
- **Symptom (Fundort):** Die `go`-Shell-Shadowing-Falle stand seit dem 2026-07-17-Lauf als **LL-02** in genau dieser Datei, mit dem Forward-Guard „CLAUDE.md + jeder Agent-Dispatch". Beim Start des Folge-Epos war sie in `CLAUDE.md` **nicht** verankert — die Falle wurde erneut entdeckt, diesmal vom `ce-specs-reviewer` in T1, und musste als D21 neu ins Epic-bean geschrieben werden. Kostenpunkt bei Nichtentdeckung: ein Agent meldet grüne Tests mit Exit 0, die **nie ausgeführt** wurden.
- **Fix:** Als D21 ins Epic-bean `beans-1ec3` (gilt für alle Kinder) und als Prelude in jedes Folge-bean; jeder Dispatch-Prompt nennt sie explizit.
- **Forward-Guard:** Ein LL-Eintrag, dessen Guard auf eine **Datei** zeigt, ist erst erledigt, wenn die Zeile **in dieser Datei steht**. Beim Session-Start jedes Folge-Epos: die Forward-Guards der letzten Session gegen ihren Zielort prüfen, nicht gegen die Erinnerung. Guard-Ort für dieses Repo ist `docs/SSTD.md` § Nicht-Ableitbarkeiten (`CLAUDE.md` ist Upstream-Datei — Ergänzungen dort vergrößern das Fork-Delta und brauchen PO-Freigabe).

### LL-11 · `awk` misst Bytes, nicht Zeichen
- **Symptom (Fundort):** Akzeptanzkriterien SC-201/SC-202 gaben wörtlich `awk '{print length($0)}' | sort -rn | head -1` als Breitenprüfung vor. `/usr/bin/awk` ist auf dieser Maschine nicht multibyte-aware (trotz UTF-8-Locale) und meldete bei der Glyphen-Ausgabe (`■ ▸ ▪`) **240 statt 80** — das Kriterium wäre scheinbar hart verletzt gewesen, obwohl die Ausgabe zeichengenau korrekt war.
- **Fix:** Zeichengenau nachgemessen mit `wc -m` und Rune-Zählung in `command python3`; beide bestätigten exakt 80 bzw. 110. Als D22 ins Epic-bean verankert.
- **Forward-Guard:** Breitenprüfungen **nie** mit `awk length()`, immer `wc -m` oder Rune-Zählung. Allgemeiner (deckt D21 und D22 gemeinsam ab): **bevor ein Kommando-Output als Beweis zitiert wird, verifizieren, dass das Kommando misst, was es messen soll.** Ein im bean wörtlich vorgegebener Prüfbefehl ist eine Absichtserklärung, kein Freibrief — Planner können Kommandos vorgeben, die auf der Zielmaschine etwas anderes tun.

### LL-12 · Der Plan war intern inkonsistent — Quelltext ≠ eigene Zielausgabe
- **Symptom (Fundort):** `PLAN.md` Task 2 Step 1 gab einen Prototyp-Quelltext vor, der ausschließlich über Milestones iterierte (kein `kids['']`). Step 3 desselben Abschnitts zeigte eine Zielausgabe **mit** `No Milestone`-Sektion (D18). Der vorgegebene Code konnte die vom selben Plan geforderte Ausgabe nicht erzeugen. An echten Daten: **277 von 278** Nicht-Milestone-beans wären kommentarlos aus der Ausgabe gefallen. Der Plan war zweimal `ce-plan-reviewer`-grün und PO-freigegeben.
- **Fix:** Implementer ergänzte die Schleife (plus Ausschluss des Milestone-beans selbst, sonst Doppel-Render), Reviewer bestätigte Lücke und Korrektheit unabhängig. Als Nachtrag ins Epic-bean: maßgebliche Referenz ist die **Datei** `render-prototype.py`, nicht der Quelltext-Block im Plan.
- **Forward-Guard:** Plan-Review muss vorgegebenen Beispiel-Quelltext **gegen die im selben Plan gezeigte Zielausgabe** prüfen — Code-Block und Ziel-Block sind zwei Behauptungen, die auseinanderlaufen können. Wo ein Plan sowohl Quelltext als auch eine ausführbare Referenzdatei kennt, gilt **die Datei**; der Plan sagt das explizit, statt es offenzulassen.

### LL-13 · Operationalisierung invalidiert die eigenen Plan-Annahmen
- **Symptom (Fundort):** Der Plan verankerte als D13 den Fakt „`main == fork/main == origin/main`, 0 divergent" und leitete daraus `git merge --ff-only` als EARS-3/SC-101/SC-102 ab. Bis zum Realisierungs-Start hatten die Operationalisierungs-Commits des Epos selbst (`.beans/`-Pflege) `main` um vier Commits vorgerückt — `--ff-only` war unmöglich, das Task-bean hätte den Implementer korrekt zum Abbruch gezwungen. Beim Verankern des PO-Entscheids stieg der Zähler nochmals, was eine zweite ERRATA nötig machte.
- **Fix:** PO-Entscheid D20 (Rebase statt Merge, Intent von EARS-3 = lineare Historie bleibt gewahrt); SC-101/102 revidiert, SC-102b (kein Merge-Commit, Commits inhaltlich unverändert obenauf) neu.
- **Forward-Guard:** Akzeptanzkriterien **nie** gegen absolute Commit-Zahlen oder einen Divergenz-Zählerstand formulieren — sie veralten durch die bean-Pflege des eigenen Epos. Stattdessen gegen **Invarianten** prüfen: „kein Merge-Commit im integrierten Bereich", „`patch-id` der Commits unverändert", „Zielsymbol im Code vorhanden". Wo eine Zahl unvermeidlich ist, zählt der Task sie zur Laufzeit frisch statt sie aus dem bean zu zitieren.

### LL-14 · Der Work-State der Vorsession lag komplett untracked
- **Symptom (Fundort):** Beim Vorflug-Check lagen **16** `.beans/*.md`-Dateien des ti53-Strangs untracked im Working Tree — weder auf `main` noch auf dem `fix/`-Branch committet, inklusive des Epic-beans `beans-ti53` und aller Task-beans. Ein `git clean` oder ein verunglückter Branch-Wechsel hätte die gesamte Nachvollziehbarkeit der Vorsession gelöscht.
- **Fix:** In einem eigenen Verankerungs-Commit versioniert (`e53ff16`), nachdem T1 durch war (vorher hätte es den Commit-Zähler der T1-Kriterien verschoben — siehe LL-13).
- **Forward-Guard:** Der Vorflug-Check jeder Realisierungs-Session prüft `git status --porcelain .beans/` und versorgt untrackte beans, **bevor** der erste Dispatch läuft. Zusätzlich am Session-Ende: `git status` muss clean sein — ein bean, das nur im Working Tree existiert, ist kein Work-State, sondern ein Zufall.

### LL-15 · Ein Testfall neben der Grenze beweist die Grenze nicht
- **Symptom (Fundort):** T3 (`beans-ejoz`) ging mit grüner Suite in den Review — 15 Testeinheiten, alle `PASS`, Literale unabhängig nachgerechnet. Die Mutations-Probe fand trotzdem **zwei** ungetestete load-bearing Zeilen:
  - **Grenzwert daneben statt drauf:** Die D17-Bedingung `prefixW >= roadmapTitleCol` (17) war nur durch einen Testfall mit Präfixlänge **26** abgedeckt. Mutation `>=`→`>` liess die komplette Suite grün — der Fall traf den Zweig, aber nie seine Grenze.
  - **Zufällig gleiche Margin:** Ein Testfall `"Prüfung ändern"` bei Breite 8 sollte Rune-Zählung (D16) absichern. `"Prüfung"` ist 7 Runen **und** 8 Bytes — beide ≤ 8, also liefern `len()` und `utf8.RuneCountInString` dasselbe Ergebnis. Die Mutation `RuneCountInString`→`len()` in `roadmapWrapTitle` blieb grün. Dass die *globale* Mutation die Suite brach, lag allein an einer anderen, korrekt getesteten Fundstelle in `roadmapLine` — der Test *wirkte* aussagekräftig, war es aber nicht.
- **Fix:** CHANGES_REQUIRED, eine Fix-Runde ohne Logikänderung. Zwei Testfälle **an** der Grenze ergänzt: Präfix mit exakt 17 Runen (mit Setup-Selbstprüfung per `t.Fatalf`, falls die Länge nicht stimmt), und `roadmapWrapTitle("ab é", 4)` — Rune-Summe 4, Byte-Summe 5, damit trennen die beiden Zählweisen nachweisbar. Beide Mutationen jetzt rot, vom Reviewer unabhängig reproduziert.
- **Forward-Guard:** Testfälle für Grenzwerte **auf** die Grenze legen (`==`), nicht bequem daneben. Bei Multibyte-Tests die Margin so wählen, dass Byte- und Rune-Zählung **unterschiedliche** Ergebnisse liefern — sonst testet der Fall die Zählweise nicht, egal wie „umlautig" er aussieht. Implementer führt die Mutations-Probe **selbst** vor dem Abschluss durch und zitiert die Rot-Ausgabe; der Reviewer reproduziert sie. Ein Test, der die zugehörige Mutation nicht rot macht, zählt nicht als Abdeckung. (Verschärft LL-07 um den Grenzwert- und den Margin-Fall.)

### LL-16 · Ein Parameter, der nur in eine Richtung mutiert wird, ist halb getestet
- **Symptom (Fundort):** T5 (`beans-zb00`) reichte `links` durch `roadmapOutput` an `renderRoadmapMarkdown` weiter. Der Implementer hatte die Zeile mutiert — auf `false` — und sein Test wurde rot, also galt sie als abgedeckt. Der Reviewer mutierte auf **`true`**: die komplette Suite blieb grün. Alle vier Testaufrufe übergaben `links=true`, der `false`-Pfad lief nie. Beobachtet war damit nur, dass der Wert *irgendwie* ankommt — nicht, dass er **durchgereicht** statt konstant gesetzt wird. Praktische Folge einer solchen Regression: das reale Flag `--no-links` würde im gepipten Pfad stillschweigend ignoriert.
- **Fix:** Test auf table-driven mit **beiden** Werten umgebaut; beide Mutationsrichtungen brechen jetzt spiegelbildlich je genau einen Subtest. Der Implementer ergänzte ungefragt eine **Fixture-Invarianz-Assertion**, die einen vakuosen Pass ausschliesst (falls die Fixture gar keine verlinkbaren beans enthielte, wären beide Zweige identisch und der Test wertlos). Der Reviewer prüfte per Sonden-Test, dass die Zweige real divergieren und der Guard nicht tautologisch erfüllt ist.
- **Forward-Guard:** Bei booleschen Parametern **beide** Richtungen mutieren (`→true` **und** `→false`). Eine Mutation, die zufällig mit dem einzigen getesteten Wert kollidiert, beweist nichts. Table-driven über den Wertebereich statt eines Einzelfalls. Wo zwei Zweige gegeneinander geprüft werden, zusätzlich sicherstellen, dass sie sich am Testdatensatz überhaupt **unterscheiden können** — sonst ist der Vergleich leer.

### LL-17 · Im bean zitierte Prüfkommandos sind Absichtserklärungen, keine Messgeräte
- **Symptom (Fundort):** Drei Akzeptanzkriterien dieses Epos gaben Kommandos wörtlich vor, die auf der Zielmaschine **etwas anderes messen als gemeint**: `mise test` (hängt an `test:e2e`, Playwright-Browser fehlt → codeunabhängiges Rot), `awk '{print length($0)}'` (misst Bytes → 240 statt 80 bei Glyphen-Ausgabe), und überall `go test` (Shell-Funktion verdeckt den Compiler → **Exit 0 ohne Testlauf**). Ein Agent, der sie buchstabengetreu absetzt, meldet entweder einen Fehlbefund oder — gefährlicher — einen grünen Haken auf einer Messung, die nie stattfand.
- **Fix:** Vor dem Dispatch des betroffenen Tasks als Prelude ins bean geschrieben, welche Kriterien wörtlich untauglich sind und welches Ersatzkommando die **Absicht** trifft. Jede Ersetzung wurde vom Implementer als Deviation dokumentiert und vom Reviewer gegen die Absicht (nicht den Buchstaben) bewertet. Der Implementer führte das awk-Ergebnis zusätzlich als Falsifikationsbeleg mit, statt es zu verschweigen.
- **Forward-Guard:** Der Supervisor prüft die Prüfkommandos eines beans **vor** dem Dispatch gegen die bekannten Umgebungsfallen der Zielmaschine (hier: D19/D21/D22, verankert in `docs/SSTD.md`) und schreibt Abweichungen als Prelude hinein — nicht der Implementer beim Stolpern. Generell: **bevor ein Kommando-Output als Beweis zitiert wird, verifizieren, dass das Kommando misst, was es messen soll.** Planner dürfen Kommandos vorgeben; sie haften nicht dafür, dass diese auf jeder Maschine funktionieren.

### LL-18 · Prüfsummen driften, Relationen nicht
- **Symptom (Fundort):** Zwei Reviewer meldeten für denselben Byte-Identitäts-Nachweis **andere** SHA-256-Werte und Zeilenzahlen als der jeweilige Implementer (89 → 88 → 82 Zeilen). Ursache war nicht der Code, sondern das `.beans/`-Verzeichnis: der Supervisor hatte zwischen den Läufen beans committet, also veränderte sich der **Input** des Renderers. Ein weniger sorgfältiger Reviewer hätte hier einen Phantom-Blocker gemeldet und eine Fix-Runde ausgelöst.
- **Fix:** Beide Reviewer ordneten es selbst korrekt ein: die Garantie lautet „**before == after auf identischem Input**", nicht „Prüfsumme gleich der im bean notierten". Beide reproduzierten den Vergleich eigenständig mit frisch gebauten Binaries.
- **Forward-Guard:** Byte-Identitäts-Kriterien als **Relation zwischen zwei gleichzeitig erzeugten Ausgaben** formulieren, nie als absoluten Hash-Wert im bean — sobald die Eingabedaten mitlaufen (hier: das eigene `.beans/`), veraltet jeder notierte Hash. Verwandt mit LL-13 (keine absoluten Commit-Zahlen in Kriterien): **Kriterien gegen Invarianten formulieren, nicht gegen Momentaufnahmen.**

### LL-19 · Was der Agent nicht messen kann, gehört als offener Punkt ans Gate — nicht als Haken
- **Symptom (Fundort):** R01 des Epos (die Glyphen `■ ▸ ▪` sind East-Asian-Ambiguous; rendert ein Terminal sie doppelt breit, verschieben sich alle Spalten um 1) ist in tmux und in einem pty **nicht** abschliessend prüfbar — dort rendern sie einspaltig. Der Abschluss-Task hätte den Haken bequem setzen können; die Ausgabe „sah richtig aus".
- **Fix:** Im Prelude ausdrücklich angewiesen, zu prüfen was messbar ist (tmux 80 Spalten, pty) und den Rest **als offenen PO-Verifikationspunkt zu melden**, statt eine unbelegte Behauptung über das echte Terminal aufzustellen. Implementer und Reviewer haben das unabhängig so berichtet.
- **Forward-Guard:** Beim Abschluss-Task jedes Epos aktiv fragen: *Welche Akzeptanz kann ein Agent strukturell nicht erbringen?* (Fremde Terminal-Emulatoren, echte Hardware, visuelle Wahrnehmung, Fremdsysteme.) Diese Punkte gehören als benannte Liste ans PO-Gate — eine ehrliche Lücke ist wertvoller als ein unbelegter Haken. Umgekehrt: ein Epos, dessen DoD nur aus agentisch prüfbaren Punkten besteht, hat seine Nutzerwirkung wahrscheinlich nicht erfasst.

### LL-20 · „Folgenlos" ist eine Behauptung über die Konsumenten, kein Freibrief
- **Symptom (Fundort):** T06 (`beans-z1we`, Prefix-Rebrand) schreibt nach dem atomaren `stageAndSwap` die neue Prefix in `.beans.yml` — aber `config.Save()` ist selbst nicht-atomar (`os.WriteFile`). Der Implementer notierte als Deviation, ein Teilausfall hier sei „folgenlos, hinterlässt nur einen veralteten `.beans.yml`-Prefix". Der Reviewer verfolgte die **Konsumenten** von `config.Beans.Prefix`: er treibt projekt-weite Short-ID-Normalisierung (`core.go:321,344,697,834`) **und** die Prefix-Vergabe neuer Beans (`core.go:490-499`). Ein Teilausfall bricht nach Reload alle Short-ID-Lookups und lässt neue Beans den alten Prefix bekommen — exakt der Mixed-Prefix-Zustand, den der eigene B04-Guard beim nächsten Rebrand verweigert. „Folgenlos" war das Gegenteil der Wahrheit.
- **Fix:** Als D02 ans PO-Gate verankert (SC eng lassen + Doku-Fix vs. Hardening-Task), die falsche Deviation-Notiz im bean korrigiert. Nicht-blockierend gegen die eng geschriebene T06-Spec, aber der PO kennt jetzt die reale Tragweite.
- **Forward-Guard:** Wer einen Failure-Mode als „harmlos/folgenlos" abtut, muss die **Konsumenten** des betroffenen Zustands nennen, nicht nur den Zustand selbst. Ein `grep` auf jeden Leser des Feldes kostet Minuten und widerlegt oder bestätigt die Behauptung. Reviewer: „harmlos"-Claims sind ein Prüf-Trigger, kein Passierschein.

### LL-21 · Ein Guard in der Planungsphase schützt nicht den Mutationspunkt (TOCTOU)
- **Symptom (Fundort):** T07 verdrahtete die Rename-Guards (Server läuft? Worktree aktiv?) in `PlanRebrand` — also in der Dry-Run-Berechnung. Zwischen Plan und dem echten `ApplyRename` (Staging+Swap) klafft ein Zeitfenster; sobald T08 einen interaktiven `y/N`-Confirm einführt, kann der User bestätigen, während dazwischen ein `beans serve` startet. Der Guard hätte gegriffen — beim Planen, nicht beim Zerstören.
- **Fix:** Als Prelude in T08 getragen. Der T08-Implementer rief die Guards unmittelbar vor dem realen Apply erneut auf (`applyRenameWithGuards` re-berechnet den Plan frisch, was die Guards als ersten Akt neu triggert **und** Plan-Staleness schließt). Der Reviewer simulierte das Fenster real (Port während des Prompts belegt) und bestätigte die Refusal + Nicht-Mutation.
- **Forward-Guard:** Ein Check gehört an den **Punkt der irreversiblen Aktion**, nicht an die Vorberechnung. Wo Plan und Apply zeitlich getrennt sind (Confirm-Prompt, Queue, Netzwerk-Roundtrip), Precondition-Checks am Apply wiederholen. „Vor dem Plan geprüft" ist bei zerstörenden Operationen kein Schutz.

### LL-22 · Eine Fixture, die still zu Zero-Value parst, macht Cascade-Tests vakuos grün
- **Symptom (Fundort):** Die PLAN-Codeblöcke (Task 2/5/8) gaben bean-Test-Fixtures mit dem `# id`-Kommentar **vor** dem öffnenden `---` vor. `pkg/bean.Parse` liefert dafür still Zero-Value-Frontmatter **ohne Fehler** (Title/Status/Parent alle `""`). Ein Cascade-Test, der auf `child.Parent == "tp-zzzz"` prüft, aber gegen eine leer-geparste Fixture läuft, kann tautologisch grün werden — er beweist nichts über die Ref-Umschreibung. Trat über drei Tasks wiederholt auf (Plan-Vorlage vererbte den Fehler).
- **Fix:** Ab T05 nutzten die Implementer die kanonische Form `---\n# id\n<yaml>\n---` (= tatsächlicher `Render()`-Output) und asserteten auf geparsten Feldern; T06 reparierte zusätzlich die geerbten T02-Fixtures. Reviewer wiesen die Nicht-Tautologie per Mutationstest nach (Rewrite-Zweig ausschalten → Test **muss** failen).
- **Forward-Guard:** Test-Fixtures immer gegen den echten Serialisierer (`Render()`/round-trip) formen, nie gegen ein handgeschriebenes Format aus dem Plan. Wo ein Parser bei Fehl-Input **still** einen Null-Wert liefert statt zu failen, ist jeder Test, der auf einem solchen Feld gründet, verdächtig — per Mutation (erwartetes Feld absichtlich brechen) prüfen, dass er überhaupt failen **kann**.

### LL-23 · Ein ungelöster Cross-Layer-Regelkonflikt driftet über die Tasks
- **Symptom (Fundort):** `Co-Authored-By`: lean-stack/CLAUDE.md fordert den Trailer explizit, der tools-weite git-Hook E2 verbietet ihn. Der Konflikt war beim Start ungeklärt. Ergebnis: T01–T06 committeten **mit** Trailer, T07/T08/T10 **ohne** (spätere Implementer folgten dem durchgesetzten Hook als sicherem Default). Die git-Historie eines einzigen Epos ist damit in sich inkonsistent.
- **Fix:** Als Q01 ans PO-Gate verankert; ab T08 im Dispatch der sichere Default vorgegeben (Trailer weg, Hook E2 ist die durchgesetzte Autorität). Nicht-blockierend, aber die Inkonsistenz ist bereits im Artefakt.
- **Forward-Guard:** Einen erkannten Regelkonflikt zwischen zwei CLAUDE.md-Schichten **vor dem ersten Task** als D ans Gate geben und im Dispatch sofort einen einheitlichen Default festlegen — nicht jeden Implementer selbst entscheiden lassen. Ein Konflikt, der über die Tasks „ausgesessen" wird, produziert genau so viele Varianten wie Tasks. Durchgesetzte Autorität (Hook) schlägt deklarative (CLAUDE.md-Prosa) als Default, bis der PO entscheidet.
