package expansion

import (
	"fmt"
	"regexp"
	"strings"
)

// variablePattern matches {{name}} and {{name:argument}} placeholders. The
// argument may contain any characters except a closing brace pair.
var variablePattern = regexp.MustCompile(`\{\{\s*([a-zA-Z_][a-zA-Z0-9_]*)(?::([^}]*))?\s*\}\}`)

// Expand renders every variable in tmpl using ctx. It returns the first error
// encountered, but continues rendering so callers see a best-effort result.
func Expand(tmpl string, ctx Context) (string, error) {
	var firstErr error
	out := variablePattern.ReplaceAllStringFunc(tmpl, func(m string) string {
		groups := variablePattern.FindStringSubmatch(m)
		name := groups[1]
		arg := groups[2]
		val, err := resolve(name, arg, ctx)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			return m
		}
		return val
	})
	return out, firstErr
}

// ValidateTemplate reports unknown variables in tmpl without rendering it. It is
// used by configuration validation so mistakes are caught before activation.
func ValidateTemplate(tmpl string) error {
	var errs []string
	for _, groups := range variablePattern.FindAllStringSubmatch(tmpl, -1) {
		name := groups[1]
		if !knownVariables[name] {
			errs = append(errs, fmt.Sprintf("unknown template variable %q", name))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

// HasVariables reports whether tmpl contains at least one variable.
func HasVariables(tmpl string) bool {
	return variablePattern.MatchString(tmpl)
}
