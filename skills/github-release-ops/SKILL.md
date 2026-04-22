---
name: github-release-ops
description: Guarded GitHub release-adjacent operations limited to dry-run-first cleanup of outdated Actions runs
---

# github-release-ops

## Purpose / Scope

Use this skill for a guarded first slice of GitHub release-adjacent repository operations.

This slice is intentionally narrow. It is for:

- dry-run-first inspection of outdated GitHub Actions workflow runs
- optional deletion of outdated workflow runs only after explicit destructive confirmation
- concise operator guidance about what remains manual

It is not for full release automation.

## Preconditions

Before using this skill, confirm all of the following:

- `shell/gh-delete-oudate-actions.sh` exists and is the only automation-ready command allowed in this slice
- the operator has explicit values for `owner`, `repo`, and `cutoff-epoch`
- `owner`, `repo`, and `cutoff-epoch` are passed explicitly and not inferred from git remotes, tags, or surrounding context
- the first execution is a dry run

If any required input is missing, stop and ask for the missing value instead of guessing.

## Allowed Command Surface

Only this command is allowed in this slice:

- `shell/gh-delete-oudate-actions.sh`
  - required flags: `--owner`, `--repo`, `--cutoff-epoch`
  - required first mode: `--dry-run`
  - only allowed destructive mode: `--execute --yes`

Recommended command forms:

- dry run:
  - `shell/gh-delete-oudate-actions.sh --dry-run --owner <owner> --repo <repo> --cutoff-epoch <unix-seconds>`
- destructive execution, only after explicit confirmation:
  - `shell/gh-delete-oudate-actions.sh --execute --yes --owner <owner> --repo <repo> --cutoff-epoch <unix-seconds>`

## Forbidden Actions

This skill must not perform, invoke, or recommend by default:

- `shell/gh-upload-release.sh`
- `shell/git-tag-with-logs.sh`
- release creation
- tag creation or publication
- asset upload
- broader GitHub release workflow automation
- destructive execution without explicit confirmation matching `--execute --yes`
- skipping the dry-run step
- inferring or defaulting `owner`, `repo`, or `cutoff-epoch`

Release creation, tag publication, and asset upload remain manual and blocked in this first slice.

## Workflow Steps

1. Confirm the request fits the narrow scope of outdated GitHub Actions workflow-run cleanup.
2. Confirm the operator has provided explicit `owner`, `repo`, and `cutoff-epoch` values.
3. Run only the dry-run form first:
   - `shell/gh-delete-oudate-actions.sh --dry-run --owner <owner> --repo <repo> --cutoff-epoch <unix-seconds>`
4. Summarize the dry-run findings clearly:
   - target repository
   - cutoff used
   - whether matching runs were found
   - whether destructive execution is still blocked pending explicit confirmation
5. Only if the operator explicitly confirms destructive execution with the exact destructive intent matching `--execute --yes`, run:
   - `shell/gh-delete-oudate-actions.sh --execute --yes --owner <owner> --repo <repo> --cutoff-epoch <unix-seconds>`
6. After destructive execution, report what was attempted and what the script reported.
7. If the request expands into release creation, tag publication, or asset upload, stop and state that those operations are manual and blocked in this slice.

## Output Contract

Return a concise operations report with these parts:

- `Scope`
  - which repository and cutoff were targeted
- `Inputs`
  - the explicit `owner`, `repo`, and `cutoff-epoch` values used
- `Mode`
  - `dry-run` or `execute`
- `Findings`
  - what the script reported
- `Blocked Operations`
  - include this section when release creation, tagging, or upload work was requested or discussed
- `Next Manual Step`
  - include this section when broader release work remains outside this slice

The report must distinguish between evidence from the script and blocked manual-only operations.

## Escalation Rules

Escalate instead of continuing when:

- `owner`, `repo`, or `cutoff-epoch` is missing
- the user asks to skip the dry run
- the user asks for destructive execution without explicit confirmation matching `--execute --yes`
- the user asks to invoke or recommend `shell/gh-upload-release.sh`
- the user asks to invoke or recommend `shell/git-tag-with-logs.sh`
- the request shifts into release creation, tag publication, asset upload, or broader release automation
- the command surface needed is larger than `shell/gh-delete-oudate-actions.sh`

When escalating, say what is allowed in this slice, what is blocked, and which manual step or later skill slice would be required next.
