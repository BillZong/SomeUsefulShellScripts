---
name: git-maintainer
description: Read-only repository maintenance analysis using the repo's existing MCP tools
---

# git-maintainer

## Purpose / Scope

Use this skill to inspect repository health with read-only MCP tools before any human decides whether cleanup or follow-up action is needed.

This first slice is intentionally narrow. It is for:

- repository line-count analysis
- large-object inspection
- sub-repository status inspection
- analysis-only cleanup suggestions

It is not for automated cleanup or repository mutation.

## Preconditions

Before using this skill, confirm the local MCP server for this repo is available and exposes these tool ids:

- `git_count_line`
- `git_find_large_files`
- `git_status_subdirs`

If one or more of those tools are unavailable, stop immediately and report that the environment prerequisites are not met.

If deeper filesystem triage is needed after the primary Git inspection, `du_directory` may be used as a secondary diagnostic only. It is not part of the default workflow.

## Allowed MCP Tools

Primary tools:

- `git_count_line`
- `git_find_large_files`
- `git_status_subdirs`

Secondary-only tool:

- `du_directory`
  - Use only after the primary Git-oriented tools show a repository hygiene or size anomaly that needs local directory-level follow-up.

## Forbidden Actions

This skill must not perform or recommend default execution of:

- `git clean`
- branch deletion
- history rewrite
- GitHub release actions
- k8s operations

This skill must stay read-only. It may analyze, summarize, and suggest human follow-up, but it must not mutate repository state.

## Workflow Steps

1. Confirm the required MCP tool ids are available.
2. Clarify the inspection target:
   - single repository
   - a directory containing multiple repositories
   - a date range if line-count analysis is requested
3. Run the smallest relevant primary tool set:
   - use `git_count_line` for churn or author/date-based line analysis
   - use `git_find_large_files` for blob-size inspection
   - use `git_status_subdirs` for nested repository status inspection
4. Summarize findings in maintenance language:
   - what looks normal
   - what looks unusual
   - what deserves manual follow-up
5. If the Git-oriented tools reveal a size or layout anomaly that still needs local confirmation, use `du_directory` as a secondary diagnostic only.
6. When branch cleanup is discussed, keep it analysis-only:
   - identify candidate branches or suspicious states
   - suggest manual review steps
   - do not prescribe default destructive commands

## Output Contract

Return a concise maintenance report with these parts:

- `Scope`
  - what repository or directory was inspected
- `Signals`
  - notable read-only findings from the MCP tools
- `Suggested Manual Follow-up`
  - human review items only
- `Blocked Preconditions`
  - include this section only when required MCP tools are missing

The report should distinguish clearly between evidence from tools and suggested next steps.

## Escalation Rules

Escalate instead of continuing when:

- a required MCP tool id is missing
- the request shifts from analysis into destructive cleanup
- the user asks for branch deletion, history rewrite, or other repository mutation
- the request expands into GitHub release or k8s workflow territory

When escalating, say what was inspected, what remains unknown, and which manual decision or separate workflow is required next.
