package bean

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/adrg/frontmatter"
	"gopkg.in/yaml.v3"
)

// tagPattern matches valid tags: lowercase letters, numbers, and hyphens.
// Must start with a letter, can contain hyphens but not consecutively or at the end.
var tagPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)

// ValidateTag checks if a tag is valid (lowercase, URL-safe, single word).
func ValidateTag(tag string) error {
	if tag == "" {
		return fmt.Errorf("tag cannot be empty")
	}
	if !tagPattern.MatchString(tag) {
		return fmt.Errorf("invalid tag %q: must be lowercase, start with a letter, and contain only letters, numbers, and hyphens", tag)
	}
	return nil
}

// NormalizeTag converts a tag to its canonical form (lowercase).
func NormalizeTag(tag string) string {
	return strings.ToLower(strings.TrimSpace(tag))
}

// HasTag returns true if the bean has the specified tag.
func (b *Bean) HasTag(tag string) bool {
	normalized := NormalizeTag(tag)
	for _, t := range b.Tags {
		if t == normalized {
			return true
		}
	}
	return false
}

// AddTag adds a tag to the bean if it doesn't already exist.
// Returns an error if the tag is invalid.
func (b *Bean) AddTag(tag string) error {
	normalized := NormalizeTag(tag)
	if err := ValidateTag(normalized); err != nil {
		return err
	}
	if !b.HasTag(normalized) {
		b.Tags = append(b.Tags, normalized)
	}
	return nil
}

// RemoveTag removes a tag from the bean.
func (b *Bean) RemoveTag(tag string) {
	normalized := NormalizeTag(tag)
	result := make([]string, 0, len(b.Tags))
	for _, t := range b.Tags {
		if t != normalized {
			result = append(result, t)
		}
	}
	b.Tags = result
}

// HasParent returns true if the bean has a parent.
func (b *Bean) HasParent() bool {
	return b.Parent != ""
}

// IsBlocking returns true if this bean is blocking the given bean ID.
func (b *Bean) IsBlocking(id string) bool {
	for _, target := range b.Blocking {
		if target == id {
			return true
		}
	}
	return false
}

// AddBlocking adds a bean ID to the blocking list if not already present.
func (b *Bean) AddBlocking(id string) {
	if !b.IsBlocking(id) {
		b.Blocking = append(b.Blocking, id)
	}
}

// RemoveBlocking removes a bean ID from the blocking list.
func (b *Bean) RemoveBlocking(id string) {
	result := make([]string, 0, len(b.Blocking))
	for _, target := range b.Blocking {
		if target != id {
			result = append(result, target)
		}
	}
	b.Blocking = result
}

// IsBlockedBy returns true if this bean is blocked by the given bean ID.
func (b *Bean) IsBlockedBy(id string) bool {
	for _, blocker := range b.BlockedBy {
		if blocker == id {
			return true
		}
	}
	return false
}

// AddBlockedBy adds a bean ID to the blocked-by list if not already present.
func (b *Bean) AddBlockedBy(id string) {
	if !b.IsBlockedBy(id) {
		b.BlockedBy = append(b.BlockedBy, id)
	}
}

// RemoveBlockedBy removes a bean ID from the blocked-by list.
func (b *Bean) RemoveBlockedBy(id string) {
	result := make([]string, 0, len(b.BlockedBy))
	for _, blocker := range b.BlockedBy {
		if blocker != id {
			result = append(result, blocker)
		}
	}
	b.BlockedBy = result
}

// Bean represents an issue stored as a markdown file with front matter.
type Bean struct {
	// ID is the unique NanoID identifier (from filename).
	ID string `yaml:"-" json:"id"`
	// Slug is the optional human-readable part of the filename.
	Slug string `yaml:"-" json:"slug,omitempty"`
	// Path is the relative path from .beans/ root (e.g., "epic-auth/abc123-login.md").
	Path string `yaml:"-" json:"path"`

	// Front matter fields
	Title     string     `yaml:"title" json:"title"`
	Status    string     `yaml:"status" json:"status"`
	Type      string     `yaml:"type,omitempty" json:"type,omitempty"`
	Priority  string     `yaml:"priority,omitempty" json:"priority,omitempty"`
	Tags      []string   `yaml:"tags,omitempty" json:"tags,omitempty"`
	CreatedAt *time.Time `yaml:"created_at,omitempty" json:"created_at,omitempty"`
	UpdatedAt *time.Time `yaml:"updated_at,omitempty" json:"updated_at,omitempty"`

	// Order is a fractional index string for manual sorting.
	Order string `yaml:"order,omitempty" json:"order,omitempty"`

	// Body is the markdown content after the front matter.
	Body string `yaml:"-" json:"body,omitempty"`

	// Parent is the optional parent bean ID (milestone, epic, or feature).
	Parent string `yaml:"parent,omitempty" json:"parent,omitempty"`

	// Blocking is a list of bean IDs that this bean is blocking.
	Blocking []string `yaml:"blocking,omitempty" json:"blocking,omitempty"`

	// BlockedBy is a list of bean IDs that are blocking this bean.
	BlockedBy []string `yaml:"blocked_by,omitempty" json:"blocked_by,omitempty"`

	// Extra holds front matter keys not covered by any typed field above.
	Extra map[string]any `yaml:"-" json:"extra,omitempty"`
}

// frontMatter is the subset of Bean that gets serialized to YAML front matter.
type frontMatter struct {
	Title     string     `yaml:"title"`
	Status    string     `yaml:"status"`
	Type      string     `yaml:"type,omitempty"`
	Priority  string     `yaml:"priority,omitempty"`
	Tags      []string   `yaml:"tags,omitempty"`
	CreatedAt *time.Time `yaml:"created_at,omitempty"`
	UpdatedAt *time.Time `yaml:"updated_at,omitempty"`
	Order     string     `yaml:"order,omitempty"`
	Parent    string     `yaml:"parent,omitempty"`
	Blocking  []string   `yaml:"blocking,omitempty"`
	BlockedBy []string   `yaml:"blocked_by,omitempty"`
}

// knownFrontMatterKeys are the YAML keys owned by a typed field on frontMatter.
var knownFrontMatterKeys = map[string]bool{
	"title":      true,
	"status":     true,
	"type":       true,
	"priority":   true,
	"tags":       true,
	"created_at": true,
	"updated_at": true,
	"order":      true,
	"parent":     true,
	"blocking":   true,
	"blocked_by": true,
}

// reservedKeyFlags maps a known front matter key to the native CLI flag that
// writes it. Keys with no direct write flag (created_at, updated_at, order)
// map to "" and get a generic "managed field" error instead of a flag name.
var reservedKeyFlags = map[string]string{
	"title":      "--title",
	"status":     "--status",
	"type":       "--type",
	"priority":   "--priority",
	"tags":       "--tag",
	"created_at": "",
	"updated_at": "",
	"order":      "",
	"parent":     "--parent",
	"blocking":   "--blocking",
	"blocked_by": "--blocked-by",
}

// ReservedKeyFlag reports whether key is a known front matter field and, if
// so, the native CLI flag that writes it. A known key with no direct write
// flag (e.g. "order") returns known=true and flag="" -- callers should fall
// back to a "managed field" message rather than inventing a flag name.
func ReservedKeyFlag(key string) (flag string, known bool) {
	flag, known = reservedKeyFlags[key]
	return flag, known
}

// normalizeYAMLValue converts the map[interface{}]interface{} produced by the
// yaml.v2 decoder (used internally by the frontmatter library) into
// map[string]any, recursively, so Extra stays JSON-marshalable.
func normalizeYAMLValue(v any) any {
	switch typed := v.(type) {
	case map[interface{}]interface{}:
		result := make(map[string]any, len(typed))
		for k, val := range typed {
			result[fmt.Sprintf("%v", k)] = normalizeYAMLValue(val)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(typed))
		for i, val := range typed {
			result[i] = normalizeYAMLValue(val)
		}
		return result
	default:
		return v
	}
}

// Parse reads a bean from a reader (markdown with YAML front matter).
func Parse(r io.Reader) (*Bean, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading bean: %w", err)
	}

	var fm frontMatter
	body, err := frontmatter.Parse(bytes.NewReader(data), &fm)
	if err != nil {
		return nil, fmt.Errorf("parsing front matter: %w", err)
	}

	raw := map[string]any{}
	if _, err := frontmatter.Parse(bytes.NewReader(data), &raw); err != nil {
		return nil, fmt.Errorf("parsing front matter: %w", err)
	}
	var extra map[string]any
	for key, value := range raw {
		if knownFrontMatterKeys[key] {
			continue
		}
		if extra == nil {
			extra = make(map[string]any, len(raw))
		}
		extra[key] = normalizeYAMLValue(value)
	}

	// Trim trailing newline from body (POSIX files end with newline, but it's not part of content)
	bodyStr := strings.TrimSuffix(string(body), "\n")

	return &Bean{
		Title:     fm.Title,
		Status:    fm.Status,
		Type:      fm.Type,
		Priority:  fm.Priority,
		Tags:      fm.Tags,
		CreatedAt: fm.CreatedAt,
		UpdatedAt: fm.UpdatedAt,
		Order:     fm.Order,
		Body:      bodyStr,
		Parent:    fm.Parent,
		Blocking:  fm.Blocking,
		BlockedBy: fm.BlockedBy,
		Extra:     extra,
	}, nil
}

// renderFrontMatter is used for YAML output with yaml.v3 (supports custom marshalers).
type renderFrontMatter struct {
	Title     string     `yaml:"title"`
	Status    string     `yaml:"status"`
	Type      string     `yaml:"type,omitempty"`
	Priority  string     `yaml:"priority,omitempty"`
	Tags      []string   `yaml:"tags,omitempty"`
	CreatedAt *time.Time `yaml:"created_at,omitempty"`
	UpdatedAt *time.Time `yaml:"updated_at,omitempty"`
	Order     string     `yaml:"order,omitempty"`
	Parent    string     `yaml:"parent,omitempty"`
	Blocking  []string   `yaml:"blocking,omitempty"`
	BlockedBy []string   `yaml:"blocked_by,omitempty"`
}

// Render serializes the bean back to markdown with YAML front matter.
func (b *Bean) Render() ([]byte, error) {
	fm := renderFrontMatter{
		Title:     b.Title,
		Status:    b.Status,
		Type:      b.Type,
		Priority:  b.Priority,
		Tags:      b.Tags,
		CreatedAt: b.CreatedAt,
		UpdatedAt: b.UpdatedAt,
		Order:     b.Order,
		Parent:    b.Parent,
		Blocking:  b.Blocking,
		BlockedBy: b.BlockedBy,
	}

	var node yaml.Node
	if err := node.Encode(&fm); err != nil {
		return nil, fmt.Errorf("encoding front matter: %w", err)
	}

	if len(b.Extra) > 0 {
		keys := make([]string, 0, len(b.Extra))
		for k := range b.Extra {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			var keyNode, valueNode yaml.Node
			if err := keyNode.Encode(k); err != nil {
				return nil, fmt.Errorf("encoding extra key %q: %w", k, err)
			}
			if err := valueNode.Encode(b.Extra[k]); err != nil {
				return nil, fmt.Errorf("encoding extra value for key %q: %w", k, err)
			}
			node.Content = append(node.Content, &keyNode, &valueNode)
		}
	}

	fmBytes, err := yaml.Marshal(&node)
	if err != nil {
		return nil, fmt.Errorf("marshaling front matter: %w", err)
	}

	var buf bytes.Buffer
	buf.WriteString("---\n")
	if b.ID != "" {
		buf.WriteString("# ")
		buf.WriteString(b.ID)
		buf.WriteString("\n")
	}
	buf.Write(fmBytes)
	buf.WriteString("---\n")
	if b.Body != "" {
		// Only add newline separator if body doesn't already start with one
		if !strings.HasPrefix(b.Body, "\n") {
			buf.WriteString("\n")
		}
		buf.WriteString(b.Body)
		// Ensure trailing newline if body doesn't end with one
		if !strings.HasSuffix(b.Body, "\n") {
			buf.WriteString("\n")
		}
	} else {
		// Even without body, add trailing newline for POSIX compliance
		buf.WriteString("\n")
	}

	return buf.Bytes(), nil
}

// ETag returns a hash of the bean's rendered content for optimistic concurrency control.
// Uses FNV-1a 64-bit hash, producing a 16-character hex string.
// Returns "0000000000000000" if rendering fails (should never happen for valid beans).
func (b *Bean) ETag() string {
	content, err := b.Render()
	if err != nil {
		// Return a sentinel value that will never match a real ETag,
		// ensuring validation will fail rather than silently passing.
		return "0000000000000000"
	}
	h := fnv.New64a()
	h.Write(content)
	return hex.EncodeToString(h.Sum(nil))
}

// MarshalJSON implements json.Marshaler to include computed etag field.
func (b *Bean) MarshalJSON() ([]byte, error) {
	type BeanAlias Bean // Avoid infinite recursion
	return json.Marshal(&struct {
		*BeanAlias
		ETag string `json:"etag"`
	}{
		BeanAlias: (*BeanAlias)(b),
		ETag:      b.ETag(),
	})
}
