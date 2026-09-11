// Command xdb-eval runs the agent task evaluations under tasks/.
//
// Usage:
//
//	xdb-eval [-tasks DIR] [-task NAME[,NAME]] [-binary PATH] [-model MODEL] [-results DIR] [-rubric]
//
// The exit code is 0 when every task passes. It is 1 when a task fails. It
// is 2 when the harness could not run.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

func main() {
	os.Exit(run())
}

func run() int {
	var (
		tasksDir   = flag.String("tasks", "tasks", "directory of task directories")
		filter     = flag.String("task", "", "comma-separated task names to run (default all)")
		binary     = flag.String("binary", "bin/xdb", "xdb binary to copy into each sandbox")
		model      = flag.String("model", "", "override the subject model for every task")
		resultsDir = flag.String("results", "", "results directory (default results/<timestamp>)")
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
		*resultsDir = filepath.Join("results", time.Now().Format("20060102-150405"))
	}

	opts := Options{Binary: binPath, Open: OpenRouterSubject, Model: *model}
	if *rubric {
		opts.Judge = OpenRouterJudge{}
	}

	results, err := runAll(ctx, tasks, opts, *resultsDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)

		return 2
	}

	if err := WriteSummary(*resultsDir, results); err != nil {
		fmt.Fprintln(os.Stderr, err)

		return 2
	}

	fmt.Print(Table(results))
	fmt.Fprintf(os.Stderr, "results: %s\n", *resultsDir)

	for _, r := range results {
		if r.Status != StatusPass {
			return 1
		}
	}

	return 0
}

func selectTasks(dir, filter string) ([]*Task, error) {
	tasks, err := LoadTasks(dir)
	if err != nil {
		return nil, err
	}

	if filter == "" {
		return tasks, nil
	}

	names := strings.Split(filter, ",")
	selected := make([]*Task, 0, len(names))

	for _, t := range tasks {
		if slices.Contains(names, t.Name) {
			selected = append(selected, t)
		}
	}

	if len(selected) == 0 {
		return nil, fmt.Errorf("no task matched %q in %s", filter, dir)
	}

	return selected, nil
}

func runAll(ctx context.Context, tasks []*Task, opts Options, resultsDir string) ([]*TaskResult, error) {
	results := make([]*TaskResult, 0, len(tasks))

	for _, task := range tasks {
		runOpts := opts
		runOpts.ResultsDir = filepath.Join(resultsDir, task.Name)

		fmt.Fprintf(os.Stderr, "==> %s\n", task.Name)

		result, err := RunTask(ctx, runOpts, task)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", task.Name, err)
		}

		fmt.Fprintf(os.Stderr, "    %s %s\n", result.Status, result.Reason)
		results = append(results, result)
	}

	return results, nil
}
