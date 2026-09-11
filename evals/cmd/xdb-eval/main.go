// Command xdb-eval runs the agent task evaluations under evals/tasks.
//
// Usage:
//
//	xdb-eval [-tasks DIR] [-task NAME[,NAME]] [-binary PATH] [-model MODEL] [-repeat N] [-results DIR] [-rubric]
//
// The exit code is 0 when every task passes every run. It is 1 when a task
// fails. It is 2 when the harness could not run.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/xdb-dev/xdb/evals"
)

func main() {
	os.Exit(run())
}

func run() int {
	var (
		tasksDir   = flag.String("tasks", "evals/tasks", "directory of task directories")
		filter     = flag.String("task", "", "comma-separated task names to run (default all)")
		binary     = flag.String("binary", "bin/xdb", "xdb binary to copy into each sandbox")
		model      = flag.String("model", "", "override the subject model for every task")
		repeat     = flag.Int("repeat", 1, "runs per task")
		resultsDir = flag.String("results", "", "results directory (default evals/results/<timestamp>)")
		rubric     = flag.Bool("rubric", false, "score each run against its rubric with an LLM judge")
	)

	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	tasks, err := selectTasks(*tasksDir, *filter)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)

		return 2
	}

	binPath, err := filepath.Abs(*binary)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)

		return 2
	}

	if *resultsDir == "" {
		*resultsDir = filepath.Join("evals", "results", time.Now().Format("20060102-150405"))
	}

	opts := evals.Options{Binary: binPath, Runner: evals.ClaudeRunner{}, Model: *model}
	if *rubric {
		opts.Judge = evals.ClaudeJudge{}
	}

	results, err := runAll(ctx, tasks, opts, *repeat, *resultsDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)

		return 2
	}

	if err := evals.WriteSummary(*resultsDir, results); err != nil {
		fmt.Fprintln(os.Stderr, err)

		return 2
	}

	summaries := evals.Summarize(results)
	fmt.Print(evals.Markdown(summaries))
	fmt.Fprintf(os.Stderr, "results: %s\n", *resultsDir)

	for _, s := range summaries {
		if s.Passed != s.Runs {
			return 1
		}
	}

	return 0
}

func selectTasks(dir, filter string) ([]*evals.Task, error) {
	tasks, err := evals.LoadTasks(dir)
	if err != nil {
		return nil, err
	}

	if filter == "" {
		return tasks, nil
	}

	want := map[string]bool{}
	for _, name := range strings.Split(filter, ",") {
		want[strings.TrimSpace(name)] = true
	}

	var selected []*evals.Task

	for _, t := range tasks {
		if want[t.Name] {
			selected = append(selected, t)
			delete(want, t.Name)
		}
	}

	if len(want) > 0 {
		missing := make([]string, 0, len(want))
		for name := range want {
			missing = append(missing, name)
		}

		return nil, fmt.Errorf("unknown tasks: %s", strings.Join(missing, ", "))
	}

	return selected, nil
}

func runAll(ctx context.Context, tasks []*evals.Task, opts evals.Options, repeat int, resultsDir string) ([]*evals.TaskResult, error) {
	var results []*evals.TaskResult

	for _, task := range tasks {
		for i := 1; i <= repeat; i++ {
			runOpts := opts
			runOpts.ResultsDir = filepath.Join(resultsDir, task.Name)

			if repeat > 1 {
				runOpts.ResultsDir = filepath.Join(resultsDir, fmt.Sprintf("%s-%d", task.Name, i))
			}

			fmt.Fprintf(os.Stderr, "==> %s (%d/%d)\n", task.Name, i, repeat)

			result, err := evals.RunTask(ctx, runOpts, task)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", task.Name, err)
			}

			fmt.Fprintf(os.Stderr, "    %s %s\n", result.Status, result.Reason)
			results = append(results, result)
		}
	}

	return results, nil
}
