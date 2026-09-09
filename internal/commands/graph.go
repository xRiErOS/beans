package commands

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/xRiErOS/beans/internal/output"
	"github.com/xRiErOS/beans/internal/ui"
	"github.com/xRiErOS/beans/pkg/bean"
	"github.com/spf13/cobra"
)

// Relation kinds a graph edge can carry. "parent" points from the parent
// bean to its child; "blocks" points from the blocking bean to the bean it
// blocks. Both directions in the front matter (Blocking and BlockedBy)
// collapse to the same "blocks" kind -- see buildGraphEdges.
const (
	relParent = "parent"
	relBlocks = "blocks"
)

// graphEdge is one relationship between two beans. It is also the
// deduplication key: Blocking and BlockedBy are not mirrored on disk (see
// buildGraphEdges), so the same logical edge can be declared from either
// end and must collapse to one entry here.
type graphEdge struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Relation string `json:"relation"`
}

// graphNode is one bean as it appears in --format json output.
type graphNode struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
	Type   string `json:"type,omitempty"`
}

// graphModel is the --format json payload: every node in scope, and every
// edge between two nodes in scope.
type graphModel struct {
	Nodes []graphNode `json:"nodes"`
	Edges []graphEdge `json:"edges"`
}

// buildGraphEdges derives every relationship edge from the given beans.
//
// Blocking and BlockedBy are not mirrored on disk and not validated for
// symmetry: the mutation resolvers write only the bean being mutated
// (pkg/beangraph/mutations.go), and beancore only unions the two directions
// transiently for GraphQL reads (pkg/beangraph/bean_fields.go). So
// A.blocking=[B] and B.blocked_by=[A] may both be present for one logical
// edge, or only one of them may be -- both must produce the same edge here.
//
// A link to a bean that is not in the given set, or a link from a bean to
// itself, is skipped rather than reported: that is beans check's job
// (pkg/beancore.CheckAllLinks reports these as BrokenLink and SelfLink), and
// a second, differently worded report here would be a rival policy.
func buildGraphEdges(all []*bean.Bean) []graphEdge {
	byID := make(map[string]*bean.Bean, len(all))
	for _, b := range all {
		byID[b.ID] = b
	}

	seen := make(map[graphEdge]bool)
	var edges []graphEdge
	add := func(from, to, relation string) {
		if from == to {
			return
		}
		if byID[from] == nil || byID[to] == nil {
			return
		}
		e := graphEdge{From: from, To: to, Relation: relation}
		if seen[e] {
			return
		}
		seen[e] = true
		edges = append(edges, e)
	}

	for _, b := range all {
		if b.Parent != "" {
			add(b.Parent, b.ID, relParent)
		}
		for _, target := range b.Blocking {
			add(b.ID, target, relBlocks)
		}
		for _, blocker := range b.BlockedBy {
			add(blocker, b.ID, relBlocks)
		}
	}

	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		if edges[i].Relation != edges[j].Relation {
			return edges[i].Relation < edges[j].Relation
		}
		return edges[i].To < edges[j].To
	})
	return edges
}

// filterGraphEdges keeps only edges whose Relation is in kinds. A nil or
// empty kinds leaves edges unchanged.
func filterGraphEdges(edges []graphEdge, kinds []string) []graphEdge {
	if len(kinds) == 0 {
		return edges
	}
	keep := make(map[string]bool, len(kinds))
	for _, k := range kinds {
		keep[k] = true
	}
	var out []graphEdge
	for _, e := range edges {
		if keep[e.Relation] {
			out = append(out, e)
		}
	}
	return out
}

// scopeGraph limits edges to the neighbourhood of rootID: every bean
// reachable within depth hops over the undirected view of edges (From->To
// and To->From both traversable), plus every edge with at least one
// endpoint closer than depth. depth == 0 means the whole connected
// component. The returned map always contains rootID, even when it has no
// edges, so an unlinked bean still yields a one-node scope.
func scopeGraph(edges []graphEdge, rootID string, depth int) (map[string]int, []graphEdge) {
	adj := make(map[string][]graphEdge)
	for _, e := range edges {
		adj[e.From] = append(adj[e.From], e)
		adj[e.To] = append(adj[e.To], e)
	}

	dist := map[string]int{rootID: 0}
	queue := []string{rootID}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		d := dist[id]
		if depth > 0 && d >= depth {
			continue
		}
		for _, e := range adj[id] {
			other := e.To
			if other == id {
				other = e.From
			}
			if _, ok := dist[other]; ok {
				continue
			}
			dist[other] = d + 1
			queue = append(queue, other)
		}
	}

	var kept []graphEdge
	for _, e := range edges {
		distFrom, okFrom := dist[e.From]
		distTo, okTo := dist[e.To]
		if !okFrom || !okTo {
			continue
		}
		if depth == 0 {
			kept = append(kept, e)
			continue
		}
		if distFrom < depth || distTo < depth {
			kept = append(kept, e)
		}
	}
	return dist, kept
}

var (
	graphFormat   string
	graphRelation []string
	graphDepth    int
)

var graphCmd = &cobra.Command{
	Use:   "graph [id]",
	Short: "Print the bean relationship graph",
	Long: `Prints the parent and blocking relationships between beans.

The default output is Graphviz DOT, which pipes into ` + "`dot -Tpng`" + ` for an
image; --format ascii prints a plain edge list for reading in a terminal,
--format mermaid a Mermaid flowchart that pastes into a Markdown document,
and --format json the same graph as data. Nodes are coloured by the status colour
from the configuration.

Naming a bean scopes the output to that bean's neighbourhood: --depth 1 (the
default) shows only relationships the bean itself takes part in, a higher
depth widens it by that many hops, and --depth 0 walks the whole connected
component.

Links pointing at a bean that does not exist, and self-references, are left
out here; ` + "`beans check`" + ` is what reports them.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if graphFormat != "dot" && graphFormat != "ascii" && graphFormat != "json" && graphFormat != "mermaid" {
			return fmt.Errorf("invalid --format %q (must be dot, ascii, mermaid or json)", graphFormat)
		}
		jsonMode := graphFormat == "json"

		if graphDepth < 0 {
			return cmdError(jsonMode, output.ErrValidation, "--depth must be >= 0, got %d", graphDepth)
		}
		if cmd.Flags().Changed("depth") && len(args) == 0 {
			return cmdError(jsonMode, output.ErrValidation, "--depth requires a bean id")
		}
		for _, r := range graphRelation {
			if r != relParent && r != relBlocks {
				return cmdError(jsonMode, output.ErrValidation, "invalid --relation %q (must be %s or %s)", r, relParent, relBlocks)
			}
		}

		all := core.All()
		bean.SortByStatusPriorityAndType(all, cfg.StatusNames(), cfg.PriorityNames(), cfg.TypeNames())

		edges := filterGraphEdges(buildGraphEdges(all), graphRelation)

		var rootID string
		if len(args) == 1 {
			resolved, err := core.Get(args[0])
			if err != nil {
				return cmdError(jsonMode, output.ErrNotFound, "bean not found: %s", args[0])
			}
			rootID = resolved.ID

			var nodeIDs map[string]int
			nodeIDs, edges = scopeGraph(edges, rootID, graphDepth)
			filtered := make([]*bean.Bean, 0, len(nodeIDs))
			for _, b := range all {
				if _, ok := nodeIDs[b.ID]; ok {
					filtered = append(filtered, b)
				}
			}
			all = filtered
		}

		switch graphFormat {
		case "ascii":
			return renderGraphASCII(cmd, edges, rootID)
		case "mermaid":
			return renderGraphMermaid(cmd, all, edges)
		case "json":
			return renderGraphJSON(cmd, all, edges)
		default:
			return renderGraphDot(cmd, all, edges)
		}
	},
}

// dotQuote escapes a string for use inside a DOT double-quoted identifier or
// attribute value.
func dotQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\r\n", `\n`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\n`)
	return s
}

func renderGraphDot(cmd *cobra.Command, beans []*bean.Bean, edges []graphEdge) error {
	w := cmd.OutOrStdout()
	fmt.Fprintln(w, "digraph beans {")
	fmt.Fprintln(w, "  rankdir=LR;")
	fmt.Fprintln(w, `  node [shape=box style=filled fontname="Helvetica"];`)
	for _, b := range beans {
		label := dotQuote(b.ID) + `\n` + dotQuote(ui.Truncate(b.Title, 40))
		attrs := fmt.Sprintf(`label="%s"`, label)
		if sc := cfg.GetStatus(b.Status); sc != nil && sc.Color != "" {
			color := string(ui.ResolveColor(sc.Color))
			if strings.HasPrefix(color, "#") {
				attrs += fmt.Sprintf(`, fillcolor="%s"`, color)
			}
		}
		fmt.Fprintf(w, "  \"%s\" [%s];\n", dotQuote(b.ID), attrs)
	}
	for _, e := range edges {
		fmt.Fprintf(w, "  \"%s\" -> \"%s\" [label=\"%s\"];\n", dotQuote(e.From), dotQuote(e.To), dotQuote(e.Relation))
	}
	fmt.Fprintln(w, "}")
	return nil
}

// mermaidHandle turns a bean id into a Mermaid node handle. Bean ids carry
// a hyphen, and Mermaid reads `beans-a --> beans-b` as an id followed by a
// malformed arrow, so anything outside [A-Za-z0-9_] becomes an underscore.
// The real id stays in the label, which is where a reader looks for it.
func mermaidHandle(id string) string {
	var sb strings.Builder
	sb.Grow(len(id) + 1)
	sb.WriteByte('n')
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			sb.WriteRune(r)
		default:
			sb.WriteByte('_')
		}
	}
	return sb.String()
}

// mermaidLabel escapes a string for a quoted Mermaid node label. A double
// quote would close the label and a bracket would close the node, either of
// which yields a diagram that does not render at all, so both become HTML
// entities -- Mermaid resolves them back when it draws the label. Angle
// brackets go the same way because Mermaid draws labels as HTML, so markup
// in a title would otherwise be rendered as markup. The <br/> the caller
// joins the label with is inserted after this escaping, never before.
// Newlines become <br/> for the same reason they become \n in DOT.
func mermaidLabel(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	s = strings.ReplaceAll(s, "[", "&#91;")
	s = strings.ReplaceAll(s, "]", "&#93;")
	s = strings.ReplaceAll(s, "(", "&#40;")
	s = strings.ReplaceAll(s, ")", "&#41;")
	s = strings.ReplaceAll(s, "{", "&#123;")
	s = strings.ReplaceAll(s, "}", "&#125;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\r\n", "<br/>")
	s = strings.ReplaceAll(s, "\n", "<br/>")
	s = strings.ReplaceAll(s, "\r", "<br/>")
	return s
}

// renderGraphMermaid prints the graph as a Mermaid flowchart, which pastes
// straight into a Markdown document. It keeps --format dot's shape: LR
// direction, id and truncated title in the label, and the status colour
// from the configuration -- as a classDef per status, since Mermaid has no
// per-node fill attribute.
func renderGraphMermaid(cmd *cobra.Command, beans []*bean.Bean, edges []graphEdge) error {
	w := cmd.OutOrStdout()
	fmt.Fprintln(w, "flowchart LR")

	classes := make(map[string]string)
	var classOrder []string
	for _, b := range beans {
		fmt.Fprintf(w, "  %s[\"%s<br/>%s\"]\n",
			mermaidHandle(b.ID), mermaidLabel(b.ID), mermaidLabel(ui.Truncate(b.Title, 40)))

		sc := cfg.GetStatus(b.Status)
		if sc == nil || sc.Color == "" {
			continue
		}
		colour := string(ui.ResolveColor(sc.Color))
		if !strings.HasPrefix(colour, "#") {
			continue
		}
		name := "s" + mermaidHandle(b.Status)
		if _, ok := classes[name]; !ok {
			classes[name] = colour
			classOrder = append(classOrder, name)
		}
		fmt.Fprintf(w, "  class %s %s;\n", mermaidHandle(b.ID), name)
	}

	for _, e := range edges {
		fmt.Fprintf(w, "  %s -->|%s| %s\n",
			mermaidHandle(e.From), mermaidLabel(e.Relation), mermaidHandle(e.To))
	}

	for _, name := range classOrder {
		fmt.Fprintf(w, "  classDef %s fill:%s,stroke:#333;\n", name, classes[name])
	}
	return nil
}

func renderGraphASCII(cmd *cobra.Command, edges []graphEdge, rootID string) error {
	w := cmd.OutOrStdout()
	if len(edges) == 0 {
		if rootID != "" {
			fmt.Fprintf(w, "no relationships for %s\n", rootID)
		} else {
			fmt.Fprintln(w, "no relationships")
		}
		return nil
	}

	widest := 0
	for _, e := range edges {
		if w := len([]rune(e.From)); w > widest {
			widest = w
		}
	}
	for _, e := range edges {
		fmt.Fprintf(w, "%s ──%s──> %s\n", ui.PadRight(e.From, widest), e.Relation, e.To)
	}
	return nil
}

func renderGraphJSON(cmd *cobra.Command, beans []*bean.Bean, edges []graphEdge) error {
	model := graphModel{Edges: edges}
	for _, b := range beans {
		model.Nodes = append(model.Nodes, graphNode{ID: b.ID, Title: b.Title, Status: b.Status, Type: b.Type})
	}
	if model.Edges == nil {
		model.Edges = []graphEdge{}
	}
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(model)
}

func RegisterGraphCmd(root *cobra.Command) {
	// Flags are bound once: these are package-level vars and pflag panics on
	// a second definition, while tests register into a throwaway root.
	if graphCmd.Flags().Lookup("format") == nil {
		graphCmd.Flags().StringVar(&graphFormat, "format", "dot", `Output format: "dot", "ascii", "mermaid" or "json"`)
		graphCmd.Flags().StringArrayVar(&graphRelation, "relation", nil, `Only this relation kind: "parent" or "blocks" (can be repeated)`)
		graphCmd.Flags().IntVar(&graphDepth, "depth", 1, "Hops from the named bean; 0 walks the whole connected component (requires a bean id)")
	}
	graphCmd.ValidArgsFunction = completionUpTo(1)
	root.AddCommand(graphCmd)
}
