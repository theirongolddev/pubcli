package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

const (
	// ExitSuccess is returned when the command succeeds.
	ExitSuccess = 0
	// ExitNotFound is returned when the requested stores/deals are not available.
	ExitNotFound = 1
	// ExitInvalidArgs is returned when the command input is invalid.
	ExitInvalidArgs = 2
	// ExitUpstream is returned when an external dependency fails.
	ExitUpstream = 3
	// ExitInternal is returned for unexpected internal failures.
	ExitInternal = 4
)

type cliError struct {
	Code        string
	Message     string
	Suggestions []string
	ExitCode    int
}

func (e *cliError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func invalidArgsError(message string, suggestions ...string) error {
	return &cliError{
		Code:        "INVALID_ARGS",
		Message:     message,
		Suggestions: suggestions,
		ExitCode:    ExitInvalidArgs,
	}
}

func notFoundError(message string, suggestions ...string) error {
	return &cliError{
		Code:        "NOT_FOUND",
		Message:     message,
		Suggestions: suggestions,
		ExitCode:    ExitNotFound,
	}
}

func upstreamError(action string, err error) error {
	return &cliError{
		Code:        "UPSTREAM_ERROR",
		Message:     fmt.Sprintf("%s: %v", action, err),
		Suggestions: []string{"Retry in a moment."},
		ExitCode:    ExitUpstream,
	}
}

type jsonErrorPayload struct {
	Error jsonErrorBody `json:"error"`
}

type jsonErrorBody struct {
	Code        string   `json:"code"`
	Message     string   `json:"message"`
	Suggestions []string `json:"suggestions,omitempty"`
	ExitCode    int      `json:"exitCode"`
}

func printCLIErrorJSON(w io.Writer, err *cliError) error {
	if err == nil {
		return nil
	}
	payload := jsonErrorPayload{
		Error: jsonErrorBody{
			Code:        err.Code,
			Message:     err.Message,
			Suggestions: err.Suggestions,
			ExitCode:    err.ExitCode,
		},
	}
	return json.NewEncoder(w).Encode(payload)
}

func formatCLIErrorText(err *cliError) string {
	if err == nil {
		return ""
	}

	lines := []string{
		fmt.Sprintf("error[%s]: %s", strings.ToLower(err.Code), err.Message),
	}
	if len(err.Suggestions) > 0 {
		lines = append(lines, "suggestions:")
		for _, suggestion := range err.Suggestions {
			lines = append(lines, "  "+suggestion)
		}
	}
	return strings.Join(lines, "\n")
}

func classifyCLIError(err error) *cliError {
	if err == nil {
		return nil
	}

	var typed *cliError
	if errors.As(err, &typed) {
		return typed
	}

	msg := strings.TrimSpace(err.Error())
	lowerMsg := strings.ToLower(msg)

	switch {
	case strings.Contains(msg, "unknown command"):
		suggestions := []string{
			"pubcli stores --zip 33101",
			"pubcli categories --zip 33101",
		}
		if bad := extractUnknownValue(msg, "unknown command"); bad != "" {
			if suggestion, ok := closestMatch(strings.ToLower(bad), knownCommands, 2); ok {
				suggestions = append([]string{fmt.Sprintf("Did you mean `%s`?", suggestion)}, suggestions...)
			}
		}
		return &cliError{
			Code:        "INVALID_ARGS",
			Message:     msg,
			Suggestions: suggestions,
			ExitCode:    ExitInvalidArgs,
		}
	case strings.Contains(msg, "unknown flag"):
		suggestions := []string{
			"pubcli --zip 33101",
			"pubcli --store 1425 --bogo",
		}
		if bad := extractUnknownValue(msg, "unknown flag"); bad != "" {
			trimmed := strings.TrimLeft(bad, "-")
			if suggestion, ok := resolveFlagName(trimmed); ok {
				suggestions = append([]string{fmt.Sprintf("Try `--%s`.", suggestion)}, suggestions...)
			}
		}
		return &cliError{
			Code:        "INVALID_ARGS",
			Message:     msg,
			Suggestions: suggestions,
			ExitCode:    ExitInvalidArgs,
		}
	case strings.Contains(msg, "requires an argument for flag"),
		strings.Contains(msg, "flag needs an argument"),
		strings.Contains(msg, "required flag(s)"):
		return &cliError{
			Code:        "INVALID_ARGS",
			Message:     msg,
			Suggestions: []string{"pubcli --zip 33101", "pubcli --store 1425"},
			ExitCode:    ExitInvalidArgs,
		}
	case strings.Contains(lowerMsg, "no publix stores found"),
		strings.Contains(lowerMsg, "no stores found near"),
		strings.Contains(lowerMsg, "no deals found"),
		strings.Contains(lowerMsg, "no deals match"):
		return &cliError{
			Code:     "NOT_FOUND",
			Message:  msg,
			ExitCode: ExitNotFound,
		}
	case strings.Contains(lowerMsg, "unexpected status"),
		strings.Contains(lowerMsg, "executing request"),
		strings.Contains(lowerMsg, "decoding response"),
		strings.Contains(lowerMsg, "fetching deals"),
		strings.Contains(lowerMsg, "fetching savings"),
		strings.Contains(lowerMsg, "fetching stores"),
		strings.Contains(lowerMsg, "finding stores"):
		return &cliError{
			Code:        "UPSTREAM_ERROR",
			Message:     msg,
			Suggestions: []string{"Retry in a moment."},
			ExitCode:    ExitUpstream,
		}
	default:
		return &cliError{
			Code:        "INTERNAL_ERROR",
			Message:     msg,
			Suggestions: []string{"Run `pubcli --help` for usage details."},
			ExitCode:    ExitInternal,
		}
	}
}

func isTTY(w io.Writer) bool {
	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(file.Fd()))
}

func hasJSONPreference(args []string) bool {
	for _, arg := range args {
		if arg == "--json" || strings.HasPrefix(arg, "--json=") {
			return true
		}
	}
	return false
}

func hasHelpRequest(args []string) bool {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" {
			return true
		}
	}
	return false
}

func shouldAutoJSON(args []string, stdoutIsTTY bool) bool {
	if stdoutIsTTY || len(args) == 0 {
		return false
	}
	if hasJSONPreference(args) || hasHelpRequest(args) {
		return false
	}
	switch firstCommand(args) {
	case "completion", "help":
		return false
	default:
		return true
	}
}

// knownShorthands maps single-character shorthands to whether they require a value.
var knownShorthands = map[byte]bool{
	's': true, // --store
	'z': true, // --zip
	'c': true, // --category
	'd': true, // --department
	'q': true, // --query
	'n': true, // --limit
}

func firstCommand(args []string) string {
	expectingValue := false
	for _, arg := range args {
		if expectingValue {
			expectingValue = false
			continue
		}
		if arg == "--" {
			break
		}
		if !strings.HasPrefix(arg, "-") {
			return arg
		}
		if strings.HasPrefix(arg, "--") {
			name, rest := splitFlag(strings.TrimPrefix(arg, "--"))
			if spec, ok := knownFlags[name]; ok && spec.requiresValue && rest == "" {
				expectingValue = true
			}
		} else if len(arg) == 2 && arg[0] == '-' {
			// Single-char shorthand like -z, -s, -n
			if needsVal, ok := knownShorthands[arg[1]]; ok && needsVal {
				expectingValue = true
			}
		}
	}
	return ""
}

type quickStartJSON struct {
	Name     string   `json:"name"`
	Usage    string   `json:"usage"`
	Examples []string `json:"examples"`
}

func printQuickStart(w io.Writer, asJSON bool) error {
	help := quickStartJSON{
		Name:  "pubcli",
		Usage: "pubcli [flags] | [stores|categories] [flags]",
		Examples: []string{
			"pubcli --zip 33101 --limit 10",
			"pubcli stores --zip 33101",
			"pubcli categories --store 1425",
		},
	}

	if asJSON {
		return json.NewEncoder(w).Encode(help)
	}

	_, err := fmt.Fprintf(
		w,
		"%s\nusage: %s\nexamples:\n  %s\n  %s\n  %s\nflags: --zip --store --json --bogo --category --department --query --sort --limit\n",
		help.Name,
		help.Usage,
		help.Examples[0],
		help.Examples[1],
		help.Examples[2],
	)
	return err
}
