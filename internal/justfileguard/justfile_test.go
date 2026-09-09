// Package justfileguard bewacht, dass die `just test`- und `just test-race`-Rezepte im
// repo-weiten `justfile` weiterhin `-count=1` unmittelbar vor `{{ ARGS }}` tragen. Ein
// spaeterer Edit am `justfile`, der `-count=1` verliert, bringt sonst den Go-Testcache still
// zurueck (siehe beans-mkfb).
package justfileguard

import (
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

// wantCountFlag ist die erwartete Zeichenfolge, wie sie beans-mkfb/AC-04 als Literal-Konstante
// verlangt. Sie wird NICHT aus dem justfile-Inhalt abgeleitet oder zurueckverglichen.
const wantCountFlag = "-count=1"

// repoRoot ermittelt den Repo-Root ueber `git rev-parse --show-toplevel` (AC-01), analog zum
// Referenzmuster gitRevParse in internal/gitutil/worktree.go:75-82.
func repoRoot(t *testing.T) string {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git rev-parse --show-toplevel: %v", err)
	}
	return strings.TrimSpace(string(out))
}

// recipeBody extrahiert den eingerueckten Koerper eines justfile-Rezepts anhand seines Namens.
// Das Rezept beginnt an einer Zeile, die mit "<name> " oder "<name>:" beginnt (Rezeptkopf, ggf.
// mit Parametern wie "ARGS='./...'"), gefolgt von eingerueckten Zeilen bis zur naechsten
// nicht-eingerueckten, nicht-leeren Zeile.
func recipeBody(t *testing.T, justfile, name string) string {
	t.Helper()
	lines := strings.Split(justfile, "\n")
	headerRe := regexp.MustCompile(`^` + regexp.QuoteMeta(name) + `(\s|:)`)

	var body []string
	inBody := false
	for _, line := range lines {
		if !inBody {
			if headerRe.MatchString(line) {
				inBody = true
			}
			continue
		}
		if strings.TrimSpace(line) == "" {
			break
		}
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			break
		}
		body = append(body, line)
	}

	if len(body) == 0 {
		t.Fatalf("recipe %q not found (or empty body) in justfile", name)
	}
	return strings.Join(body, "\n")
}

// requireCountBeforeArgs prueft, dass wantCountFlag unmittelbar (durch Whitespace getrennt) vor
// {{ ARGS }} bzw. {{ARGS}} im Rezeptkoerper steht (AC-02/AC-03), unabhaengig von der konkreten
// Leerzeichen-Schreibweise innerhalb der doppelten geschweiften Klammern (Risk-Eintrag im bean).
func requireCountBeforeArgs(t *testing.T, body string) {
	t.Helper()
	pattern := regexp.MustCompile(regexp.QuoteMeta(wantCountFlag) + `\s+\{\{\s*ARGS\s*\}\}`)
	if !pattern.MatchString(body) {
		t.Errorf("expected %q immediately before {{ ARGS }}/{{ARGS}} in recipe body, got:\n%s", wantCountFlag, body)
	}
}

func TestJustfileTestRecipeHasCountFlag(t *testing.T) {
	justfile := read(t, repoRoot(t)+"/justfile")
	requireCountBeforeArgs(t, recipeBody(t, justfile, "test"))
}

func TestJustfileTestRaceRecipeHasCountFlag(t *testing.T) {
	justfile := read(t, repoRoot(t)+"/justfile")
	requireCountBeforeArgs(t, recipeBody(t, justfile, "test-race"))
}

func read(t *testing.T, path string) string {
	t.Helper()
	out, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(out)
}
