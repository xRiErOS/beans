#!/usr/bin/env bash
# Interaktive tmux-Bühne für die CLI-Oberfläche von beans (beans-ejxf).
#
# Baut das aktuelle Binary, legt einen Wegwerf-Store mit realistischem
# Status-Mix an und startet eine zsh mit geladener Vervollständigung.
#
# Das `beans` in der Sitzung ist ein Wrapper mit fest verdrahtetem
# --beans-path: aus KEINEM Arbeitsverzeichnis kann es den echten Store
# treffen. Das ist der Punkt der Bühne — hier wird Fehlbedienung geübt.
set -euo pipefail

# Beide Werte kommen vom Aufrufer. Kein Default für $root: der Pfad steht
# hinter einem `rm -rf`, und ein zweiter Default hier wäre eine zweite
# Quelle für denselben Wert (justfile-Kanon, Regel 35). Das justfile führt
# ihn als Variable und übergibt ihn beiden Rezepten.
session="${1:?Sitzungsname fehlt}"
root="${2:?Wurzel des Wegwerf-Verzeichnisses fehlt}"
repo="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

command -v tmux >/dev/null || { echo "tmux fehlt" >&2; exit 1; }

rm -rf "$root"
mkdir -p "$root/bin" "$root/zdot"

echo "==> Binary bauen"
go build -o "$root/bin/beans-real" "$repo/cmd/beans"

# HOME erst JETZT umlenken, nach dem Build. `beans-6y60` persistiert den
# Suchindex nach `~/.beans/index/<hash der Store-Wurzel>`
# (`pkg/beancore/search_index.go`, über `os.UserHomeDir()`) — also AUSSERHALB
# von $root. Ohne Umlenkung hinterlässt jeder Bühnenlauf ein Verzeichnis im
# echten `~/.beans`, das `just demo-stop` nicht wegräumen kann, weil es nur
# $root löscht. Auf der alten Basis war das unsichtbar, weil es dort noch
# keine Persistenz gab.
#
# Die Reihenfolge ist nicht kosmetisch: vor dem Build gesetzt, macht die
# Umlenkung `$root/home/go` zum GOPATH, und Go legt dort 443 MB Modulcache mit
# Nur-Lese-Rechten ab — `rm -rf $root` scheitert dann mit `Permission denied`,
# und `demo-stop` bricht seine Zusage. Nach dem Build enthält `$root/home`
# nur noch `.beans/`.
export HOME="$root/home"
mkdir -p "$HOME"

# Ab hier IM Demo-Verzeichnis arbeiten und `init` ohne --beans-path laufen
# lassen: nur dann schreibt es eine eigene .beans.yml und der Store wird
# selbstständig. Mit --beans-path aus dem Repo heraus käme der Store aus dem
# Flag, die KONFIGURATION aber weiter aus der .beans.yml des Repos — die
# Beans hießen dann `beans-…` und lägen auf `draft` statt `todo`. Dieselbe
# Art Auseinanderentwicklung zwischen Store- und Config-Auflösung hat
# beans-9jtq für den `__complete`-Pfad synchronisiert (--beans-path UND
# --config folgen dort jetzt derselben Präzedenz wie im regulären Pfad).
cd "$root"
store="$root/.beans"

echo "==> Wegwerf-Store anlegen"
"$root/bin/beans-real" init >/dev/null

# Gar kein ID-Präfix. Mit einem Präfix — auch einem kurzen — teilen sich
# ALLE IDs einen gemeinsamen Anfang, und das erste TAB ergänzt nur diesen,
# statt die Kandidatenliste zu zeigen. Korrektes zsh-Verhalten, aber es
# kostet auf der Bühne einen Tastendruck vor dem, was sie vorführen soll.
# `sed -i ''` ist BSD-Syntax und bricht unter GNU sed; das Skript liegt im
# veröffentlichten Fork und muss auf Linux/CI laufen.
tmp="$(mktemp)"
sed 's/^    prefix: .*/    prefix: ""/' "$root/.beans.yml" >"$tmp"
mv "$tmp" "$root/.beans.yml"

bake() { "$root/bin/beans-real" "$@"; }
new_id() { bake create "$1" "${@:2}" | awk '{print $2}'; }

# Offene Beans: was ein Nutzer per TAB tatsächlich sucht.
bake create "Login-Maske zeigt keinen Fehler" --type bug --priority high >/dev/null
bake create "Suchfeld in der Kopfzeile" --type feature >/dev/null
bake create "Abhaengigkeiten aktualisieren" --type task >/dev/null
bake create "Export als CSV" --type feature --priority low >/dev/null
bake start "$(new_id 'Umbau der Seitenleiste' --type feature)" >/dev/null

# Altlast, bewusst in der Mehrheit: ohne sie ist beans-sfle (TAB-Liste
# besteht am echten Store zu 91 % aus Abgeschlossenem) auf der Bühne
# unbeobachtbar. Ein Fixture, das nur den guten Fall zeigt, prüft nichts.
for i in $(seq 1 24); do
	bake complete "$(new_id "Erledigte Altlast $i" --type task)" >/dev/null
done
for i in $(seq 1 6); do
	bake scrap "$(new_id "Verworfene Idee $i" --type feature)" --reason "Bühnen-Fixture" >/dev/null
done

# Wrapper: bindet den Store ans Kommando, nicht ans Verzeichnis.
cat >"$root/bin/beans" <<EOF
#!/bin/sh
exec "$root/bin/beans-real" --beans-path "$store" "\$@"
EOF
chmod +x "$root/bin/beans"

"$root/bin/beans-real" completion zsh >"$root/completion.zsh"

# Die Completion ist auf den Kommandonamen `beans` registriert, deshalb
# muss der Wrapper über PATH gefunden werden — ein Aufruf über den vollen
# Pfad umgeht compdef und TAB ergänzt dann Dateinamen.
cat >"$root/zdot/.zshrc" <<EOF
export PATH="$root/bin:\$PATH"
export HOME="$root/home"
autoload -Uz compinit && compinit -u
source "$root/completion.zsh"
cd "$root"
PS1='beans-demo%% '
cat <<'BANNER'

  beans-Demo — Wegwerf-Store, der echte Store ist nicht erreichbar.

    beans list                      Bestand ansehen
    beans show <TAB>                IDs mit Titel und Status
    beans complete <id> <TAB>       Kandidaten auch auf Position 2
    beans list junk                 Fehlbedienung: Exit 1
    beans archive <id>              Fehlbedienung: bricht ab, archiviert nichts

BANNER
EOF

tmux kill-session -t "$session" 2>/dev/null || true
tmux new-session -d -s "$session" -x 140 -y 40 "ZDOTDIR=$root/zdot zsh -i"

# Readiness messen, nicht annehmen: die Sitzung ist erst brauchbar, wenn
# compinit durch ist und der Prompt steht.
ready=0
for _ in $(seq 1 40); do
	if tmux capture-pane -p -t "$session" 2>/dev/null | grep -q 'beans-demo%'; then
		ready=1
		break
	fi
	sleep 0.25
done
[ "$ready" = 1 ] || echo "WARNUNG: Prompt der Sitzung nicht beobachtet" >&2

# Bestand direkt aus den Store-Dateien zählen, nicht über ein list-Flag:
# so kann die Zeile nicht mit dem Verb-Vokabular auseinanderlaufen.
printf '\n==> Bestand der Buehne\n'
sed -n 's/^status: //p' "$store"/*.md | sort | uniq -c | sort -rn | sed 's/^/    /'

cat <<EOF

==> Buehne laeuft: $session

    tmux attach -t $session      anhaengen  (loesen: Ctrl-b d)
    just demo-stop               beenden und Store entfernen
EOF
