---
id: implementation_plan
name: Implementation Plan
description: Create a developer-reviewed implementation plan before generating, testing, or executing FileMaker changes.
allowed_tools:
  - route_request
  - get_context_freshness
  - get_module_context
  - get_database_context
  - search_objects
  - find_script
  - find_layout
  - find_table
  - find_field
  - get_dependencies
  - get_impact_report
  - read_artifact
  - write_outbox_artifact
fallback_tools:
  - list_dir
  - read_file
  - search_file
required_outputs:
  - implementation_plan.md
  - context_snapshot.json
  - impact_report.json
---

# Implementation Plan Workflow

## Purpose

Create a reviewed implementation plan before KiBuild generates, refactors, tests, or executes FileMaker script/layout changes.

This workflow is the safety gate between "idea" and "action."

## When to Use

Use before:

- generating a non-trivial script
- refactoring an existing script
- creating multiple related scripts
- touching scripts with cross-module dependencies
- using FileMaker-backed test data
- executing any generated script in FileMaker
- creating layouts once direct layout tools become available

Small read-only analysis tasks do not need this workflow.

## Procedure

### 1. Route the Request

Identify:

- module
- database
- target scripts/layouts/tables/fields
- whether this is create, modify, test, or execute

Use indexes and graph tools before raw XML.

### 2. Check Freshness

Report freshness for every target database.

If context is stale, plan should include:

```text
Required first step: export/index refresh
```

### 3. Gather Impact

Use dependency tools to identify:

- scripts called
- scripts that call target scripts
- layouts used
- tables/fields touched
- cross-module dependencies
- do-not-modify notes

### 4. Define Proposed Change

Describe:

- goal
- user-facing behavior
- generated/modified scripts
- generated/modified layouts
- expected inputs and outputs
- expected side effects
- testing approach

### 5. Define Script Suite

For generated multi-script work, propose the script suite:

```text
Public entry script
Helper scripts
Repository/data scripts
Utility scripts
```

Include whether helpers should be called by name.

### 6. Define Test Plan

Include:

- static validation
- offline simulation
- test cases
- whether `test_mode` is required
- whether `dry_run` is required
- whether FileMaker-backed test data is needed
- exact tables/fields requested for fixture data

### 7. Define Risks and Rollback

Report:

- stale context risk
- data mutation risk
- cross-module risk
- naming ambiguity
- FileMaker version/API limitations
- rollback strategy

### 8. Write Plan Artifact

Write versioned plan artifact:

```text
outbox/docs/implementation_plan_{scope_slug}/vNNN_{timestamp}/implementation_plan.md
outbox/docs/implementation_plan_{scope_slug}/vNNN_{timestamp}/context_snapshot.json
outbox/docs/implementation_plan_{scope_slug}/vNNN_{timestamp}/impact_report.json
```

### 9. Preview and Wait

Preview the plan.

Do not generate, test, or execute script changes until the developer approves the plan.

## Final Response

```text
Implementation plan ready: {scope}

Proposed work:
{short summary}

Needs approval before:
- generating scripts
- pulling FileMaker data
- running FileMaker-backed tests
- applying changes

Plan:
{artifact version path}
```

