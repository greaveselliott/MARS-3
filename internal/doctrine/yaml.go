/*
FactoryDocSync:
docs:
- docs/features/F-001-doctrine-foundation.md
- docs/design-docs/mars-provenance.md
- docs/code-documentation-map.md
*/

package doctrine

import (
	"bufio"
	"strconv"
	"strings"
)

type yamlContext struct {
	indent int
	key    string
}

// yamlScalars reads the deliberately small, data-only YAML subset used by the
// harness manifests. It does not attempt to be a general YAML implementation.
func yamlScalars(data []byte) map[string]string {
	values := make(map[string]string)
	var stack []yamlContext
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		raw := strings.TrimRight(scanner.Text(), " \t\r")
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "-") {
			continue
		}
		colon := strings.Index(trimmed, ":")
		if colon < 1 {
			continue
		}
		indent := len(raw) - len(strings.TrimLeft(raw, " "))
		key := strings.TrimSpace(trimmed[:colon])
		value := trimYAMLScalar(trimmed[colon+1:])
		for len(stack) > 0 && stack[len(stack)-1].indent >= indent {
			stack = stack[:len(stack)-1]
		}
		parts := make([]string, 0, len(stack)+1)
		for _, context := range stack {
			parts = append(parts, context.key)
		}
		parts = append(parts, key)
		values[strings.Join(parts, ".")] = value
		if value == "" {
			stack = append(stack, yamlContext{indent: indent, key: key})
		}
	}
	return values
}

func trimYAMLScalar(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
			if value[0] == '"' {
				if decoded, err := strconv.Unquote(value); err == nil {
					return decoded
				}
			}
			return value[1 : len(value)-1]
		}
	}
	if index := strings.Index(value, " #"); index >= 0 {
		value = strings.TrimSpace(value[:index])
	}
	return value
}

func scalar(values map[string]string, names ...string) string {
	for _, name := range names {
		if value, ok := values[name]; ok {
			return value
		}
	}
	return ""
}

func normalizeKey(value string) string {
	value = strings.ToLower(value)
	value = strings.ReplaceAll(value, "-", "")
	value = strings.ReplaceAll(value, "_", "")
	value = strings.ReplaceAll(value, " ", "")
	value = strings.ReplaceAll(value, "\t", "")
	return value
}
