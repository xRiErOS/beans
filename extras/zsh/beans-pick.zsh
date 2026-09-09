# beans-pick.zsh — a ZLE widget that binds Alt+i (`^[i`) to `beans pick`
# and splices the selected bean ID into the current line at the cursor.
#
# The widget never executes the assembled line and never reads it back to
# classify it (R-14, R-15): it only calls `beans pick`, reads its single
# stdout ID, and inserts that opaque string via LBUFFER — leaving cursor
# position and any text after it untouched.
#
# Usage: source this file from .zshrc, alongside `beans completion zsh`:
#
#   source /path/to/beans-pick.zsh
#
# The key binding is configurable: set BEANS_PICK_KEYBIND to a bindkey
# keyspec before sourcing this file to use something other than the
# default `^[i`. `^I` (Tab) is never offered here — it is left untouched
# for whatever it is already bound to (e.g. fzf-tab-complete).

beans-pick-widget() {
	local id
	zle -I
	id=$(beans pick </dev/tty) || { zle reset-prompt; return; }
	zle reset-prompt
	[[ -z $id ]] && return
	LBUFFER+="$id "
}
zle -N beans-pick-widget

: ${BEANS_PICK_KEYBIND:='^[i'}
bindkey "$BEANS_PICK_KEYBIND" beans-pick-widget
