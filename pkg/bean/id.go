package bean

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	gonanoid "github.com/matoous/go-nanoid/v2"
)

const idAlphabet = "0123456789abcdefghijklmnopqrstuvwxyz"

// blockedIDWords contains offensive words that could appear in generated IDs
// using the [a-z0-9] alphabet. Only includes words 3-4 chars long since
// default IDs are 4 characters.
var blockedIDWords = []string{
	"ass", "cum", "fag", "fu0k", "fuck", "fuk",
	"gay", "god",
	"ho3", "hoe",
	"jiz",
	"kkk", "kum",
	"naz", "nig",
	"poo",
	"rap", "rim",
	"s3x", "sex", "sh1t", "shit", "slut",
	"tit", "twat",
	"wop",
	// Specific 4-char words
	"anus", "cock", "coon", "crap", "damn", "dick", "dumb",
	"dyke", "gook", "homo", "jerk", "kike", "knob", "lmao",
	"muff", "nazi", "nob", "oral", "piss", "poop",
	"porn", "pube", "puss", "rape", "scum", "slag",
	"slob", "smeg", "spic", "suck", "turd", "wank",
}

// containsBlockedWord checks if the given ID contains any blocked substring.
func containsBlockedWord(id string) bool {
	for _, word := range blockedIDWords {
		if strings.Contains(id, word) {
			return true
		}
	}
	return false
}

// NewID generates a new NanoID for a bean with an optional prefix and configurable length.
// It regenerates if the ID contains an offensive word.
func NewID(prefix string, length int) (string, error) {
	for {
		id, err := gonanoid.Generate(idAlphabet, length)
		if err != nil {
			return "", fmt.Errorf("generating nanoid: %w", err)
		}
		if !containsBlockedWord(id) {
			return prefix + id, nil
		}
	}
}

// ParseFilename extracts the ID and optional slug from a bean filename.
// Supports multiple formats for backward compatibility:
//   - New format: "f7g--user-registration.md" -> ("f7g", "user-registration")
//   - Dot format: "f7g.user-registration.md" -> ("f7g", "user-registration")
//   - Legacy format: "f7g-user-registration.md" -> ("f7g", "user-registration")
//   - ID only: "f7g.md" -> ("f7g", "")
func ParseFilename(name string) (id, slug string) {
	// Remove .md extension
	name = strings.TrimSuffix(name, ".md")

	// Try new format first (double-dash separator): id--slug
	if idx := strings.Index(name, "--"); idx > 0 {
		return name[:idx], name[idx+2:]
	}

	// Try dot format: id.slug
	if idx := strings.Index(name, "."); idx > 0 {
		return name[:idx], name[idx+1:]
	}

	// Fall back to original legacy format (single dash separator): id-slug
	parts := strings.SplitN(name, "-", 2)
	id = parts[0]
	if len(parts) > 1 {
		slug = parts[1]
	}
	return id, slug
}

// BuildFilename constructs a filename from ID and optional slug.
// Uses double-dash separator: id--slug.md
func BuildFilename(id, slug string) string {
	if slug == "" {
		return id + ".md"
	}
	return id + "--" + slug + ".md"
}

// Slugify converts a title to a URL-friendly slug.
func Slugify(title string) string {
	// Convert to lowercase
	s := strings.ToLower(title)

	// Replace spaces and underscores with dashes
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "_", "-")

	// Remove non-alphanumeric characters (except dashes)
	var result strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' {
			result.WriteRune(r)
		}
	}
	s = result.String()

	// Collapse multiple dashes
	re := regexp.MustCompile(`-+`)
	s = re.ReplaceAllString(s, "-")

	// Trim dashes from ends
	s = strings.Trim(s, "-")

	// Truncate to reasonable length
	if len(s) > 50 {
		s = s[:50]
		// Don't end with a dash
		s = strings.TrimRight(s, "-")
	}

	return s
}

// IDSuffix returns id with prefix removed. If id does not start with
// prefix (or prefix is empty), id is returned unchanged.
func IDSuffix(id, prefix string) string {
	if prefix == "" {
		return id
	}
	return strings.TrimPrefix(id, prefix)
}

// RebrandID rewrites id from oldPrefix to newPrefix, preserving the suffix.
func RebrandID(id, oldPrefix, newPrefix string) string {
	return newPrefix + IDSuffix(id, oldPrefix)
}
