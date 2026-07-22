package cli

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSkillsGet_UnknownSkill_NotFound(t *testing.T) {
	configPath, _ := tempCLIConfig(t)

	stdout, stderr, code := runCLI(t, "--config", configPath, "skills", "get", "nosuch")

	assert.Equal(t, ExitAppError, code)
	assert.Empty(t, stdout)

	var env describeErrEnvelope
	require.NoError(t, json.Unmarshal([]byte(stderr), &env), "stderr: %s", stderr)
	assert.Equal(t, CodeNotFound, env.Code)
	assert.Equal(t, "skills", env.Resource)
	assert.Equal(t, "get", env.Action)
	assert.Contains(t, env.Hint, "xdb skills")
}

func TestSkillsGet_MissingArg_InvalidArgument(t *testing.T) {
	configPath, _ := tempCLIConfig(t)

	stdout, stderr, code := runCLI(t, "--config", configPath, "skills", "get")

	assert.Equal(t, ExitInvalidArgs, code)
	assert.Empty(t, stdout)

	var env describeErrEnvelope
	require.NoError(t, json.Unmarshal([]byte(stderr), &env), "stderr: %s", stderr)
	assert.Equal(t, CodeInvalidArgument, env.Code)
	assert.Equal(t, "skills", env.Resource)
	assert.Equal(t, "get", env.Action)
}

func TestSkillsGet_KnownSkill_PrintsContent(t *testing.T) {
	configPath, _ := tempCLIConfig(t)

	stdout, _, code := runCLI(t, "--config", configPath, "skills", "get", "getting-started")

	assert.Equal(t, ExitOK, code)
	assert.NotEmpty(t, stdout)
}
