// Package evals runs agent task evaluations against the xdb CLI.
//
// An eval gives a headless agent a real-world task in a sandbox, in phases.
// The harness builds the sandbox and drives each phase. After each phase, it
// runs checks against the store through the CLI and grades the answers of
// the agent. It also reads the trajectory for friction and
// progressive-disclosure metrics. A failed eval is a product finding. The
// harness never patches the product.
//
// See docs/plans/2026-09-11-evals-framework.md for the design.
package evals
