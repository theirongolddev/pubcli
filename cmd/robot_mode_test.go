package cmd

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShouldAutoJSON(t *testing.T) {
	assert.True(t, shouldAutoJSON([]string{"stores", "--zip", "33101"}, false))
	assert.False(t, shouldAutoJSON([]string{"stores", "--zip", "33101", "--json"}, false))
	assert.False(t, shouldAutoJSON([]string{"completion", "zsh"}, false))
	assert.False(t, shouldAutoJSON([]string{"--help"}, false))
	assert.False(t, shouldAutoJSON([]string{"stores", "--zip", "33101"}, true))
}

func TestFirstCommand_SkipsFlagValues(t *testing.T) {
	cmd := firstCommand([]string{"--zip", "33101", "stores"})
	assert.Equal(t, "stores", cmd)
}

func TestPrintQuickStart_JSON(t *testing.T) {
	var buf bytes.Buffer
	err := printQuickStart(&buf, true)
	require.NoError(t, err)

	var payload quickStartJSON
	err = json.Unmarshal(buf.Bytes(), &payload)
	require.NoError(t, err)

	assert.Equal(t, "pubcli", payload.Name)
	assert.NotEmpty(t, payload.Usage)
	assert.Len(t, payload.Examples, 3)
}

func TestPrintCLIErrorJSON(t *testing.T) {
	var buf bytes.Buffer
	err := printCLIErrorJSON(&buf, classifyCLIError(invalidArgsError("bad flag", "pubcli --zip 33101")))
	require.NoError(t, err)

	var payload map[string]any
	err = json.Unmarshal(buf.Bytes(), &payload)
	require.NoError(t, err)

	errorObject, ok := payload["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "INVALID_ARGS", errorObject["code"])
	assert.Equal(t, "bad flag", errorObject["message"])
}
