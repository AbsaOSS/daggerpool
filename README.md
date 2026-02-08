# daggerpool
DAG-driven worker pool orchestrator for Go: execute job graphs with concurrency limits, timeouts, retries, and structured results (frontiers/subtrees) for safe state reconciliation.

# Overview

This repository provides a Go library for building concurrent, deterministic workflows on top of a worker pool with explicit state handling.
The library is composed of a small set of focused packages:
 - `pkg/workerpool` Core execution engine for running jobs with bounded concurrency and controlled lifecycle.
 - `pkg/store` In-memory state storage used to track job execution, results, and transitions.
 - `pkg/readiness` Utilities for probing and waiting on readiness conditions, built around `context.Context` for safe cancellation and timeouts.

The project is designed as a library, not a framework — it provides composable building blocks that can be embedded into controllers, orchestrators, or higher-level systems.

# Usage



# Contributing

## Pull Requests
When opening a Pull Request, please:
 - Keep PRs focused and minimal
 - Include tests where applicable
 - Add or update documentation/comments if behavior changes
 - Describe why the change is needed, not just what was changed
 - CI pipelines must pass before the PR can be reviewed.

## Design & Behavior Changes
 - For non-trivial changes (e.g. scheduling semantics, DAG behavior, readiness logic):
 - Prefer opening an issue first to discuss the approach
 - Be explicit about expected behavior and edge cases
 - Consider backward compatibility

