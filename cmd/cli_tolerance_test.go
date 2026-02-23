package cmd

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeCLIArgs_RewritesCommonFlagSyntax(t *testing.T) {
	args, notes := normalizeCLIArgs([]string{"-zip", "33101", "json"})

	assert.Equal(t, []string{"--zip", "33101", "--json"}, args)
	assert.NotEmpty(t, notes)
}

func TestNormalizeCLIArgs_RewritesTypoFlag(t *testing.T) {
	args, notes := normalizeCLIArgs([]string{"--ziip", "33101"})

	assert.Equal(t, []string{"--zip", "33101"}, args)
	assert.NotEmpty(t, notes)
}

func TestNormalizeCLIArgs_RewritesCommandTypo(t *testing.T) {
	args, notes := normalizeCLIArgs([]string{"categoriess", "--zip", "33101"})

	assert.Equal(t, []string{"categories", "--zip", "33101"}, args)
	assert.NotEmpty(t, notes)
}

func TestNormalizeCLIArgs_DoesNotRewriteCompletionPositionalArgs(t *testing.T) {
	args, notes := normalizeCLIArgs([]string{"completion", "zsh"})

	assert.Equal(t, []string{"completion", "zsh"}, args)
	assert.Empty(t, notes)
}

func TestNormalizeCLIArgs_DoesNotRewriteHelpCommandArgAsFlag(t *testing.T) {
	args, notes := normalizeCLIArgs([]string{"help", "stores"})

	assert.Equal(t, []string{"help", "stores"}, args)
	assert.Empty(t, notes)
}

func TestNormalizeCLIArgs_RespectsDoubleDashBoundary(t *testing.T) {
	args, notes := normalizeCLIArgs([]string{"stores", "--", "zip", "33101"})

	assert.Equal(t, []string{"stores", "--", "zip", "33101"}, args)
	assert.Empty(t, notes)
}

func TestNormalizeCLIArgs_LeavesKnownShorthandUntouched(t *testing.T) {
	args, notes := normalizeCLIArgs([]string{"-z", "33101", "-n", "5"})

	assert.Equal(t, []string{"-z", "33101", "-n", "5"}, args)
	assert.Empty(t, notes)
}

func TestExplainCLIError_UnknownFlagIncludesSuggestionAndExamples(t *testing.T) {
	msg := explainCLIError(errors.New("unknown flag: --ziip"))

	assert.Contains(t, msg, "Try `--zip`.")
	assert.Contains(t, msg, "pubcli --zip 33101")
	assert.Contains(t, msg, "pubcli --store 1425 --bogo")
}

func TestExplainCLIError_UnknownCommandIncludesSuggestionAndExamples(t *testing.T) {
	msg := explainCLIError(errors.New("unknown command \"stors\" for \"pubcli\""))

	assert.Contains(t, msg, "Did you mean `stores`?")
	assert.Contains(t, msg, "pubcli stores --zip 33101")
	assert.Contains(t, msg, "pubcli categories --zip 33101")
}
