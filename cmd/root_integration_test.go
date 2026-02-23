package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunCLI_CompletionZsh(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runCLI([]string{"completion", "zsh"}, &stdout, &stderr)

	assert.Equal(t, 0, code)
	assert.Contains(t, stdout.String(), "#compdef pubcli")
	assert.Empty(t, stderr.String())
}

func TestRunCLI_HelpStores(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runCLI([]string{"help", "stores"}, &stdout, &stderr)

	assert.Equal(t, 0, code)
	assert.Contains(t, stdout.String(), "pubcli stores [flags]")
	assert.Empty(t, stderr.String())
}

func TestRunCLI_TolerantRewriteWithoutNetworkCall(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runCLI([]string{"stores", "-zip", "33101", "--help"}, &stdout, &stderr)

	assert.Equal(t, 0, code)
	assert.Contains(t, stdout.String(), "pubcli stores [flags]")
	assert.Contains(t, stderr.String(), "interpreted `-zip` as `--zip`")
}

func TestRunCLI_DoubleDashBoundary(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runCLI([]string{"stores", "--", "zip", "33101", "--help"}, &stdout, &stderr)

	assert.Equal(t, 0, code)
	assert.Contains(t, stdout.String(), "pubcli stores [flags]")
	assert.False(t, strings.Contains(stderr.String(), "interpreted `zip` as `--zip`"))
}
