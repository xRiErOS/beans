#!/usr/bin/env bash
# test-zsh-widget.sh — proves extras/zsh/beans-pick.zsh's widget contract
# end-to-end under a real zsh, without depending on an interactive picker
# or the real `beans` binary (beans-fcj0).
#
# A stub `beans` on PATH records its argv and echoes a fixed ID. `zle` is
# stubbed as a no-op before the widget is sourced, since there is no real
# ZLE outside an interactive zsh line editor.
#
# Two cases exercise the branch beans-eeej added to beans-pick-widget: a
# non-empty $BUFFER forwards --line/--cursor to `beans pick` (this also
# regression-covers beans-eeej's Go-side AC5 fix), an empty one calls it
# flag-less. Both also prove the widget's splice contract: LBUFFER gains
# "<id> ", RBUFFER is untouched -- so a mutation of `LBUFFER+=` to `BUFFER=`
# fails this script, and dropping --line/--cursor from the forwarding call
# fails it too.
set -euo pipefail

repo="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
widget="$repo/extras/zsh/beans-pick.zsh"

# A missing interpreter must not turn this guard into a silent pass on the
# path that gates merges: locally a skip is a convenience, on a runner it is
# the same no-op class the guard exists to prevent.
require() {
	local tool="$1" why="$2"
	command -v "$tool" >/dev/null 2>&1 && return 0
	if [[ -n "${CI:-}" ]]; then
		echo "FAIL: $tool missing on CI ($why) -- the widget guard cannot be skipped here" >&2
		exit 1
	fi
	echo "$tool not installed ($why), skipping beans-pick-widget test"
	exit 0
}

require zsh "the widget under test is zsh"
require python3 "needed for the pty the widget's /dev/tty redirect requires"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

fixed_id="beans-stub1"
argv_log="$tmp/argv.log"
bin_dir="$tmp/bin"
mkdir -p "$bin_dir"

cat > "$bin_dir/beans" <<STUB
#!/usr/bin/env bash
printf '%s\n' "\$*" >> "$argv_log"
echo "$fixed_id"
STUB
chmod +x "$bin_dir/beans"

export PATH="$bin_dir:$PATH"

failures=0

# run_case sources the widget in a fresh, non-interactive zsh under a pty
# (the widget's own command substitution redirects from /dev/tty, so there
# has to be a controlling terminal), seeds BUFFER/CURSOR/LBUFFER/RBUFFER,
# invokes beans-pick-widget, and writes the resulting LBUFFER/RBUFFER to
# result_file so this script can assert on them without parsing terminal
# noise.
#
# The pty comes from python3's stdlib pty.spawn rather than script(1): the
# BSD/macOS and util-linux dialects of script(1) take their arguments in
# incompatible orders, and picking the wrong one degrades to a silent no-op
# on the very platform (CI) that cannot be rehearsed locally. python3 is not
# a declared tool in mise.toml, but it ships with both macOS and the CI
# runner image and behaves identically on both; the require() guard above
# turns its absence into a hard CI failure rather than a skipped guard.
run_case() {
	local name="$1" buffer="$2" cursor="$3" lbuffer="$4" rbuffer="$5"
	local case_script="$tmp/$name.zsh"
	local result_file="$tmp/$name.result"

	{
		echo "zle() { :; }"
		printf 'BUFFER=%q\n' "$buffer"
		printf 'CURSOR=%s\n' "$cursor"
		printf 'LBUFFER=%q\n' "$lbuffer"
		printf 'RBUFFER=%q\n' "$rbuffer"
		printf 'source %q\n' "$widget"
		echo "beans-pick-widget"
		printf '{ printf "LBUFFER=%%s\\n" "$LBUFFER"; printf "RBUFFER=%%s\\n" "$RBUFFER"; } > %q\n' "$result_file"
	} > "$case_script"

	local status=0
	python3 -c 'import pty,sys; sys.exit(pty.spawn(sys.argv[1:]))' \
		zsh -f "$case_script" >/dev/null 2>&1 || status=$?
	if [[ ! -s "$result_file" ]]; then
		echo "FAIL: $name case produced no result (pty exit $status) -- the widget never ran" >&2
		failures=$((failures + 1))
	fi
}

assert_eq() {
	local label="$1" got="$2" want="$3"
	if [[ "$got" != "$want" ]]; then
		echo "FAIL: $label = $(printf '%q' "$got"), want $(printf '%q' "$want")" >&2
		failures=$((failures + 1))
	fi
}

assert_contains() {
	local label="$1" haystack="$2" needle="$3"
	if [[ "$haystack" != *"$needle"* ]]; then
		echo "FAIL: $label does not contain $(printf '%q' "$needle"); got: $haystack" >&2
		failures=$((failures + 1))
	fi
}

assert_not_contains() {
	local label="$1" haystack="$2" needle="$3"
	if [[ "$haystack" == *"$needle"* ]]; then
		echo "FAIL: $label unexpectedly contains $(printf '%q' "$needle"); got: $haystack" >&2
		failures=$((failures + 1))
	fi
}

# Case 1: non-empty buffer with a non-empty prefix AND a non-empty suffix
# around the cursor -- forwards --line/--cursor.
prefix="beans list "
suffix="extra"
buffer="$prefix$suffix"
cursor="${#prefix}"

run_case "nonempty" "$buffer" "$cursor" "$prefix" "$suffix"
lbuffer_got="$(sed -n 's/^LBUFFER=//p' "$tmp/nonempty.result" 2>/dev/null || true)"
rbuffer_got="$(sed -n 's/^RBUFFER=//p' "$tmp/nonempty.result" 2>/dev/null || true)"
argv_got="$(cat "$argv_log" 2>/dev/null || true)"

assert_eq "non-empty-buffer LBUFFER" "$lbuffer_got" "${prefix}${fixed_id} "
assert_eq "non-empty-buffer RBUFFER" "$rbuffer_got" "$suffix"
assert_contains "non-empty-buffer argv" "$argv_got" "--line $buffer --cursor $cursor"

# Case 2: empty buffer -- calls flag-less `beans pick`, no --line forwarded.
: > "$argv_log"
run_case "empty" "" "0" "" ""
argv_got="$(cat "$argv_log" 2>/dev/null || true)"
assert_contains "empty-buffer argv" "$argv_got" "pick"
assert_not_contains "empty-buffer argv" "$argv_got" "--line"

if [[ "$failures" -gt 0 ]]; then
	echo "$failures assertion(s) failed" >&2
	exit 1
fi

echo "beans-pick-widget: all assertions passed"
