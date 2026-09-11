package evals

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestShippedTasksLoad keeps every task under evals/tasks valid.
func TestShippedTasksLoad(t *testing.T) {
	tasks, err := LoadTasks("tasks")
	require.NoError(t, err)
	require.NotEmpty(t, tasks)

	for _, task := range tasks {
		assert.NotEmpty(t, task.Description, task.Name)
		assert.NotEmpty(t, task.Rubric, task.Name)

		for _, phase := range task.Phases {
			assert.NotEmpty(t, phase.Checks, "%s phase %s has no checks", task.Name, phase.Name)
		}
	}
}
