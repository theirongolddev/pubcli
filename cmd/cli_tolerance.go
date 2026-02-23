package cmd

import (
	"fmt"
	"strings"
)

type flagSpec struct {
	name          string
	requiresValue bool
}

var knownFlags = map[string]flagSpec{
	"store":      {name: "store", requiresValue: true},
	"zip":        {name: "zip", requiresValue: true},
	"json":       {name: "json", requiresValue: false},
	"category":   {name: "category", requiresValue: true},
	"department": {name: "department", requiresValue: true},
	"bogo":       {name: "bogo", requiresValue: false},
	"query":      {name: "query", requiresValue: true},
	"limit":      {name: "limit", requiresValue: true},
	"help":       {name: "help", requiresValue: false},
}

var knownCommands = []string{
	"categories",
	"stores",
	"completion",
	"help",
}

var flagAliases = map[string]string{
	"zipcode":      "zip",
	"postal-code":  "zip",
	"store-number": "store",
	"storeno":      "store",
	"dept":         "department",
	"search":       "query",
	"max":          "limit",
}

func normalizeCLIArgs(args []string) ([]string, []string) {
	out := make([]string, 0, len(args))
	notes := make([]string, 0, 2)
	commandChosen := false
	activeCommand := ""
	nestedCommandAllowed := false
	nestedCommandChosen := false
	allowBareFlagRewrite := true
	expectingValue := false
	afterDoubleDash := false

	for i, tok := range args {
		if afterDoubleDash {
			out = append(out, tok)
			continue
		}

		if expectingValue {
			out = append(out, tok)
			expectingValue = false
			continue
		}

		if tok == "--" {
			out = append(out, tok)
			afterDoubleDash = true
			continue
		}

		canBeCommand := !commandChosen || (nestedCommandAllowed && !nestedCommandChosen)
		normalized, note, isFlag, needsValue, isCommand := normalizeToken(tok, canBeCommand, allowBareFlagRewrite)
		if note != "" {
			notes = append(notes, note)
		}
		out = append(out, normalized)

		if isCommand {
			if !commandChosen {
				commandChosen = true
				activeCommand = normalized
				allowBareFlagRewrite = bareFlagRewriteAllowed(activeCommand)
				nestedCommandAllowed = allowsNestedCommandArg(activeCommand)
				continue
			}
			if nestedCommandAllowed && !nestedCommandChosen {
				nestedCommandChosen = true
			}
		}
		if isFlag && needsValue && !strings.Contains(normalized, "=") && i < len(args)-1 {
			expectingValue = true
		}
	}

	return out, notes
}

func normalizeToken(tok string, canBeCommand bool, allowBareFlagRewrite bool) (normalized, note string, isFlag, needsValue, isCommand bool) {
	if tok == "--" {
		return tok, "", false, false, false
	}

	if strings.HasPrefix(tok, "--") {
		flagName, rest := splitFlag(strings.TrimPrefix(tok, "--"))
		canonical, ok := resolveFlagName(flagName)
		if ok {
			newTok := "--" + canonical + rest
			if newTok != tok {
				return newTok, fmt.Sprintf("interpreted `%s` as `%s`; use `%s` next time.", tok, newTok, newTok), true, knownFlags[canonical].requiresValue, false
			}
			return newTok, "", true, knownFlags[canonical].requiresValue, false
		}
		return tok, "", true, false, false
	}

	if strings.HasPrefix(tok, "-") && len(tok) > 2 {
		flagName, rest := splitFlag(strings.TrimPrefix(tok, "-"))
		canonical, ok := resolveFlagName(flagName)
		if ok {
			newTok := "--" + canonical + rest
			return newTok, fmt.Sprintf("interpreted `%s` as `%s`; use `%s` next time.", tok, newTok, newTok), true, knownFlags[canonical].requiresValue, false
		}
		return tok, "", true, false, false
	}

	if strings.Contains(tok, "=") && !strings.HasPrefix(tok, "-") {
		flagName, rest := splitFlag(tok)
		canonical, ok := resolveFlagName(flagName)
		if ok {
			newTok := "--" + canonical + rest
			return newTok, fmt.Sprintf("interpreted `%s` as `%s`; use `%s` next time.", tok, newTok, newTok), true, knownFlags[canonical].requiresValue, false
		}
	}

	if canBeCommand && !strings.HasPrefix(tok, "-") {
		if corrected, ok := resolveCommand(tok); ok {
			if corrected != tok {
				return corrected, fmt.Sprintf("interpreted command `%s` as `%s`; use `%s` next time.", tok, corrected, corrected), false, false, true
			}
			return tok, "", false, false, true
		}
	}

	if allowBareFlagRewrite && !strings.HasPrefix(tok, "-") {
		canonical, ok := resolveFlagName(tok)
		if ok {
			newTok := "--" + canonical
			return newTok, fmt.Sprintf("interpreted `%s` as `%s`; use `%s` next time.", tok, newTok, newTok), true, knownFlags[canonical].requiresValue, false
		}
	}

	return tok, "", false, false, false
}

func bareFlagRewriteAllowed(command string) bool {
	// Some commands (for example `stores` and `categories`) are flag-only, so
	// rewriting bare tokens like `zip` -> `--zip` is helpful there.
	switch command {
	case "stores", "categories":
		return true
	default:
		return false
	}
}

func allowsNestedCommandArg(command string) bool {
	// These commands accept another command token as a positional argument.
	switch command {
	case "help", "completion":
		return true
	default:
		return false
	}
}

func resolveFlagName(raw string) (string, bool) {
	name := strings.ToLower(strings.TrimSpace(raw))
	name = strings.ReplaceAll(name, "_", "-")

	if canonical, ok := flagAliases[name]; ok {
		return canonical, true
	}
	if _, ok := knownFlags[name]; ok {
		return name, true
	}

	if suggestion, ok := closestMatch(name, mapKeys(knownFlags), 2); ok {
		return suggestion, true
	}
	return "", false
}

func resolveCommand(raw string) (string, bool) {
	name := strings.ToLower(strings.TrimSpace(raw))
	for _, cmd := range knownCommands {
		if name == cmd {
			return cmd, true
		}
	}
	if suggestion, ok := closestMatch(name, knownCommands, 2); ok {
		return suggestion, true
	}
	return "", false
}

func explainCLIError(err error) string {
	return formatCLIErrorText(classifyCLIError(err))
}

func splitFlag(value string) (string, string) {
	parts := strings.SplitN(value, "=", 2)
	if len(parts) == 2 {
		return parts[0], "=" + parts[1]
	}
	return value, ""
}

func extractUnknownValue(msg, marker string) string {
	idx := strings.Index(msg, marker)
	if idx == -1 {
		return ""
	}

	remaining := strings.TrimSpace(msg[idx+len(marker):])
	remaining = strings.TrimPrefix(remaining, ":")
	remaining = strings.TrimSpace(remaining)

	if strings.HasPrefix(remaining, "\"") {
		remaining = strings.TrimPrefix(remaining, "\"")
		end := strings.Index(remaining, "\"")
		if end >= 0 {
			return remaining[:end]
		}
	}

	if strings.HasPrefix(remaining, "`") {
		remaining = strings.TrimPrefix(remaining, "`")
		end := strings.Index(remaining, "`")
		if end >= 0 {
			return remaining[:end]
		}
	}

	if fields := strings.Fields(remaining); len(fields) > 0 {
		return strings.Trim(fields[0], "\"`")
	}
	return ""
}

func mapKeys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}

func closestMatch(target string, candidates []string, maxDistance int) (string, bool) {
	best := ""
	bestDist := maxDistance + 1

	for _, candidate := range candidates {
		d := levenshtein(target, candidate)
		if d < bestDist {
			bestDist = d
			best = candidate
		}
	}

	if bestDist <= maxDistance {
		return best, true
	}
	return "", false
}

func levenshtein(a, b string) int {
	if a == b {
		return 0
	}
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)

	for j := range prev {
		prev[j] = j
	}

	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 0
			if a[i-1] != b[j-1] {
				cost = 1
			}
			del := prev[j] + 1
			ins := curr[j-1] + 1
			sub := prev[j-1] + cost
			curr[j] = minInt(del, ins, sub)
		}
		prev, curr = curr, prev
	}
	return prev[len(b)]
}

func minInt(vals ...int) int {
	best := vals[0]
	for _, v := range vals[1:] {
		if v < best {
			best = v
		}
	}
	return best
}
