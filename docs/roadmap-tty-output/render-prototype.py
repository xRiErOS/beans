#!/usr/bin/env python3
"""Verbindliche Layout-Referenz für `beans roadmap` im TTY (Variante β, D13).

Aufruf:  python3 render-prototype.py [breite]
         Ohne Argument: echte Terminalbreite, Cap 110, Floor 80.
         Erwartet den Pfad zum Demo-.beans-Verzeichnis in /tmp/roadmap_bp.txt.
"""
import sys, json, subprocess, textwrap, shutil

BP = open('/tmp/roadmap_bp.txt').read().strip()
if len(sys.argv) > 1:
    W = int(sys.argv[1])
else:
    W = max(80, min(shutil.get_terminal_size((110, 24)).columns, 110))

raw = subprocess.run(['beans', 'list', '--beans-path', BP, '--json', '--full'],
                     capture_output=True, text=True).stdout
beans = json.loads(raw)
kids = {}
for b in beans:
    kids.setdefault(b.get('parent') or '', []).append(b)

ARCHIVE = {'completed', 'scrapped'}
def open_(b): return b['status'] not in ARCHIVE

TITLE_COL = 17                              # Variante β (D13), war 15
PRIO_W, STAT_W, ID_W = 8, 11, 4
RIGHT_W = PRIO_W + 2 + STAT_W + 2 + ID_W    # = 27
TITLE_W = W - TITLE_COL - 2 - RIGHT_W       # = W - 46

def sid(b): return b['id'].split('-')[-1]

def right_block(b, show_prio):
    prio = (b.get('priority') or '') if show_prio else ''
    if prio == 'normal':                    # D10: Default nicht zeigen
        prio = ''
    stat = b.get('status') or ''
    return f"{prio:>{PRIO_W}}  {stat:<{STAT_W}}  {sid(b):<{ID_W}}"

def emit(prefix, b, show_prio):
    """Eine Bean-Zeile. Titel wrappt mit Hanging-Indent auf TITLE_COL (D07)."""
    lines = textwrap.wrap(b['title'], TITLE_W) or ['']
    if len(prefix) >= TITLE_COL:            # D17: Präfix zu lang -> 1 Space
        first = prefix + ' ' + lines[0]
    else:
        first = f"{prefix:<{TITLE_COL}}{lines[0]}"
    rb = right_block(b, show_prio)
    pad = max(2, W - RIGHT_W - len(first))
    print(first + ' ' * pad + rb)
    for cont in lines[1:]:
        print(' ' * TITLE_COL + cont)

def leaf_prefix(indent, b):
    return ' ' * indent + '- ' + b['type']

def emit_feature_group(f, indent):
    """Feature-Ast + seine Leafs. D15: Feature-Zeile zeigt Priority."""
    emit(' ' * indent + '▪ Feature', f, show_prio=True)
    for it in [c for c in kids.get(f['id'], []) if open_(c)]:
        emit(leaf_prefix(indent + 2, it), it, show_prio=True)

def emit_epic_group(e, indent):
    items = [c for c in kids.get(e['id'], []) if open_(c) and c['type'] != 'feature']
    feats = [c for c in kids.get(e['id'], []) if open_(c) and c['type'] == 'feature']
    if not items and not feats:
        return
    emit(' ' * indent + '▸ Epic', e, show_prio=False)
    for it in items:                                    # Leafs zuerst ...
        emit(leaf_prefix(indent + 2, it), it, show_prio=True)
    for f in feats:                                     # ... dann Feature-Äste
        emit_feature_group(f, indent + 2)

print("Roadmap")
print("═" * W)

for m in [b for b in beans if b['type'] == 'milestone']:
    print()
    emit("■ Milestone", m, show_prio=False)
    mkids = kids.get(m['id'], [])
    for e in [c for c in mkids if c['type'] == 'epic']:
        emit_epic_group(e, 2)
    for f in [c for c in mkids if c['type'] == 'feature' and open_(c)]:
        emit_feature_group(f, 2)
    for it in [c for c in mkids
               if c['type'] not in ('epic', 'feature') and open_(c)]:   # D12
        emit(leaf_prefix(2, it), it, show_prio=True)

# D18: Orphans (kein Milestone-Parent) unter "No Milestone" — nackte Zeile
# Spalte 0, Leerzeile davor. Lücke im PLAN.md-Step-1-Quelltext: dort fehlte
# diese Schleife, obwohl der Ziel-Format-Block (Step 3) sie zeigt und D12/D18
# sie fordern. Ohne sie würden alle Beans ohne Milestone-Vorfahren (der
# überwiegende Teil der echten Repo-Daten) kommentarlos verschwinden. Siehe
# Deviation-Notiz im bean-Abschluss.
orphans = kids.get('', [])
oepics = [c for c in orphans if c['type'] == 'epic']
ofeats = [c for c in orphans if c['type'] == 'feature' and open_(c)]
oleaves = [c for c in orphans
           if c['type'] not in ('epic', 'feature', 'milestone') and open_(c)]
if oepics or ofeats or oleaves:
    print()
    print("No Milestone")
    print()
    for e in oepics:
        emit_epic_group(e, 2)
    for f in ofeats:
        emit_feature_group(f, 2)
    for it in oleaves:
        emit(leaf_prefix(2, it), it, show_prio=True)
