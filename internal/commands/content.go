package commands

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/hmans/beans/pkg/bean"
	"github.com/hmans/beans/internal/output"
)

// resolveContent returns content from a direct value or file flag.
// If value is "-", reads from stdin.
func resolveContent(value, file string) (string, error) {
	if value != "" && file != "" {
		return "", fmt.Errorf("cannot use both --body and --body-file")
	}

	if value == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("reading stdin: %w", err)
		}
		return string(data), nil
	}

	if value != "" {
		return bean.UnescapeBody(value), nil
	}

	if file != "" {
		data, err := os.ReadFile(file)
		if err != nil {
			return "", fmt.Errorf("reading file: %w", err)
		}
		return string(data), nil
	}

	return "", nil
}

// applyTags adds tags to a bean, returning an error if any tag is invalid.
func applyTags(b *bean.Bean, tags []string) error {
	for _, tag := range tags {
		if err := b.AddTag(tag); err != nil {
			return err
		}
	}
	return nil
}

// formatCycle formats a cycle path for display.
func formatCycle(path []string) string {
	return strings.Join(path, " → ")
}

// cmdError returns an appropriate error for JSON or text mode.
// Note: Use %v instead of %w for error arguments - wrapping is not preserved in JSON mode.
func cmdError(jsonMode bool, code string, format string, args ...any) error {
	if jsonMode {
		return output.Error(code, fmt.Sprintf(format, args...))
	}
	return fmt.Errorf(format, args...)
}

// mergeTags combines existing tags with additions and removals.
func mergeTags(existing, add, remove []string) []string {
	tags := make(map[string]bool)
	for _, t := range existing {
		tags[t] = true
	}
	for _, t := range add {
		tags[t] = true
	}
	for _, t := range remove {
		delete(tags, t)
	}
	result := make([]string, 0, len(tags))
	for t := range tags {
		result = append(result, t)
	}
	return result
}

// applyBodyReplace replaces exactly one occurrence of old with new.
// Returns an error if old is not found or found multiple times.
func applyBodyReplace(body, old, new string) (string, error) {
	return bean.ReplaceOnce(body, old, new)
}

// applyBodyAppend appends text to the body with a newline separator.
func applyBodyAppend(body, text string) string {
	return bean.AppendWithSeparator(body, text)
}

// parseSetPair splits a --set key=value argument on the first "=". Only the
// first "=" is significant, so values may contain their own "=" (e.g. URLs).
func parseSetPair(arg string) (key, value string, err error) {
	idx := strings.Index(arg, "=")
	if idx < 0 {
		return "", "", fmt.Errorf("invalid --set value %q: expected key=value", arg)
	}
	return arg[:idx], arg[idx+1:], nil
}

// checkReservedKey returns an error naming the native flag for key if key is
// a field of the known bean schema, nil otherwise. Known keys without a
// direct write flag (created_at, updated_at, order) get a "managed field"
// error instead of a fabricated flag name.
func checkReservedKey(key string) error {
	flag, known := bean.ReservedKeyFlag(key)
	if !known {
		return nil
	}
	if flag == "" {
		return fmt.Errorf("%q is a managed field and cannot be set directly", key)
	}
	return fmt.Errorf("%q is a reserved field; use %s instead", key, flag)
}

// validateExtraKeys checks every --set and --unset argument up front: --set
// entries must carry "=" (a usage error otherwise) and neither --set nor
// --unset may name a reserved schema field.
func validateExtraKeys(sets []string, unsets []string) error {
	for _, s := range sets {
		key, _, err := parseSetPair(s)
		if err != nil {
			return err
		}
		if err := checkReservedKey(key); err != nil {
			return err
		}
	}
	for _, key := range unsets {
		if err := checkReservedKey(key); err != nil {
			return err
		}
	}
	return nil
}

// applyExtraOps applies --set and --unset operations to a bean's Extra map.
// --set operations are applied first, then --unset removes keys -- a key
// named by both --set and --unset in the same invocation ends up unset. This
// keeps --unset predictable as the more destructive, last-word operation.
// Callers must run validateExtraKeys first; a malformed --set here returns
// an error but any already-applied --set writes are not rolled back.
func applyExtraOps(b *bean.Bean, sets []string, unsets []string) error {
	for _, s := range sets {
		key, value, err := parseSetPair(s)
		if err != nil {
			return err
		}
		if b.Extra == nil {
			b.Extra = make(map[string]any)
		}
		b.Extra[key] = value
	}
	for _, key := range unsets {
		delete(b.Extra, key)
	}
	return nil
}

// resolveAppendContent handles --append value, supporting stdin with "-".
func resolveAppendContent(value string) (string, error) {
	if value == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("reading stdin: %w", err)
		}
		return strings.TrimRight(string(data), "\n"), nil
	}
	return bean.UnescapeBody(value), nil
}
