package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"gopkg.in/yaml.v3"
)

// ErrInvalidTask is the error for a task.yaml that fails validation.
var ErrInvalidTask = errors.New("[xdb/evals] invalid task")

const (
	// DefaultModel is the subject model when task.yaml sets none.
	DefaultModel = "anthropic/claude-sonnet-4.6"
	// DefaultMaxTurns is the limit on agent turns per phase when task.yaml sets none.
	DefaultMaxTurns = 60
	// TaskFileName is the file the loader reads from each task directory.
	TaskFileName = "task.yaml"
)

var taskNameRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// Task is one evaluation. It holds the subject configuration, the ordered
// phases, and an optional judge rubric. The loader reads it from a task.yaml.
type Task struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Tags        []string `yaml:"tags"`
	Namespace   string   `yaml:"namespace"`
	Model       string   `yaml:"model"`
	MaxTurns    int      `yaml:"max_turns"`
	Budget      Budget   `yaml:"budget"`
	Phases      []Phase  `yaml:"phases"`
	Rubric      string   `yaml:"rubric"`

	// Dir is the directory that holds the task.yaml and the fixtures.
	Dir string `yaml:"-"`
}

// Budget holds optional friction limits. A zero value is no limit. If a run
// exceeds a limit, the task fails.
type Budget struct {
	FailedCommands int     `yaml:"failed_commands"`
	BlindFailures  int     `yaml:"blind_failures"`
	DiscoveryCalls int     `yaml:"discovery_calls"`
	CostUSD        float64 `yaml:"cost_usd"`
}

// Phase is one turn of the task conversation. It has a prompt for the
// subject and the checks that run after the subject finishes.
type Phase struct {
	Name   string  `yaml:"name"`
	Prompt string  `yaml:"prompt"`
	Checks []Check `yaml:"checks"`
}

// Check is one graded assertion. Exactly one of Run, Answer, or AnswerRegex
// is set. A Run check runs a command in the sandbox and passes on exit code
// 0. An Answer check compares the ANSWER line of the subject's final reply.
type Check struct {
	Name string `yaml:"name"`
	// Run is a shell command. Exit code 0 passes.
	Run         string  `yaml:"run"`
	Answer      *string `yaml:"answer"`
	AnswerRegex string  `yaml:"answer_regex"`
}

// LoadTask reads and validates the task.yaml in dir.
func LoadTask(dir string) (*Task, error) {
	path := filepath.Join(dir, TaskFileName)

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("[xdb/evals] open task: %w", err)
	}
	defer func() { _ = f.Close() }()

	dec := yaml.NewDecoder(f)
	dec.KnownFields(true)

	var task Task
	if err := dec.Decode(&task); err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrInvalidTask, path, err)
	}

	task.Dir = dir
	task.applyDefaults()

	if err := task.validate(); err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrInvalidTask, path, err)
	}

	return &task, nil
}

// LoadTasks loads every task directory under root, sorted by name. It skips
// a directory without a task.yaml. The task name must match the directory
// name.
func LoadTasks(root string) ([]*Task, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("[xdb/evals] read tasks dir: %w", err)
	}

	tasks := make([]*Task, 0, len(entries))

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dir := filepath.Join(root, entry.Name())
		if _, err := os.Stat(filepath.Join(dir, TaskFileName)); err != nil {
			continue
		}

		task, err := LoadTask(dir)
		if err != nil {
			return nil, err
		}

		if task.Name != entry.Name() {
			return nil, fmt.Errorf(
				"%w: %s: name %q does not match directory %q",
				ErrInvalidTask,
				dir,
				task.Name,
				entry.Name(),
			)
		}

		tasks = append(tasks, task)
	}

	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].Name < tasks[j].Name
	})

	return tasks, nil
}

func (t *Task) applyDefaults() {
	if t.Model == "" {
		t.Model = DefaultModel
	}

	if t.MaxTurns == 0 {
		t.MaxTurns = DefaultMaxTurns
	}
}

func (t *Task) validate() error {
	if t.Name == "" {
		return errors.New("name is required")
	}

	if !taskNameRe.MatchString(t.Name) {
		return fmt.Errorf("name must match %s, got %q", taskNameRe, t.Name)
	}

	if t.Namespace == "" {
		return errors.New("namespace is required")
	}

	if len(t.Phases) == 0 {
		return errors.New("at least one phase is required")
	}

	seen := make(map[string]bool, len(t.Phases))

	for i := range t.Phases {
		phase := &t.Phases[i]

		if phase.Name == "" {
			return fmt.Errorf("phase %d: name is required", i+1)
		}

		if seen[phase.Name] {
			return fmt.Errorf("phase %q: duplicate name", phase.Name)
		}

		seen[phase.Name] = true

		if phase.Prompt == "" {
			return fmt.Errorf("phase %q: prompt is required", phase.Name)
		}

		if err := phase.validateChecks(); err != nil {
			return fmt.Errorf("phase %q %w", phase.Name, err)
		}
	}

	return nil
}

func (p *Phase) validateChecks() error {
	for i := range p.Checks {
		check := &p.Checks[i]

		if check.Name == "" {
			return fmt.Errorf("check %d: name is required", i+1)
		}

		if err := check.validate(); err != nil {
			return fmt.Errorf("check %q: %w", check.Name, err)
		}
	}

	return nil
}

func (c *Check) validate() error {
	kinds := 0

	if c.Run != "" {
		kinds++
	}

	if c.Answer != nil {
		kinds++
	}

	if c.AnswerRegex != "" {
		kinds++
	}

	if kinds != 1 {
		return errors.New("exactly one of run, answer, answer_regex must be set")
	}

	if c.AnswerRegex != "" {
		if _, err := regexp.Compile(c.AnswerRegex); err != nil {
			return fmt.Errorf("answer_regex: %w", err)
		}
	}

	return nil
}
