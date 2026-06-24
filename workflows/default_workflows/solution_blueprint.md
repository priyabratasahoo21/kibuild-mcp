---
id: solution_blueprint
name: Solution Blueprint
description: Interview the developer and create a greenfield or brownfield FileMaker solution blueprint before build execution.
allowed_tools:
  - route_request
  - get_context_freshness
  - get_module_context
  - get_database_context
  - search_objects
  - get_dependencies
  - get_impact_report
  - write_outbox_artifact
fallback_tools:
  - list_dir
  - read_file
  - search_file
required_outputs:
  - solution_blueprint.md
  - data_model.json
  - build_steps.md
  - script_plan.md
  - layout_plan.md
  - test_plan.md
---

# Solution Blueprint Workflow

## Purpose

Create a developer-reviewed blueprint before building or changing a FileMaker solution.

This workflow supports:

```text
greenfield apps
existing single-file apps
existing multi-file apps
large 30-50 file solutions
```

## Procedure

### 1. Determine Mode

Classify the request:

```text
greenfield
brownfield_single_file
brownfield_multi_file
brownfield_large_solution
```

### 2. Interview

Ask only the missing questions needed to produce a useful blueprint.

Core questions:

- What problem does this solve?
- Who uses it?
- What modules are needed?
- What are the core tables/entities?
- What workflows must be supported?
- What reports/dashboards matter?
- Is this desktop, mobile, WebDirect, or mixed?
- What should be generated first?

For existing apps, also ask:

- Which FileMaker files are in scope?
- Are there legacy areas to avoid?
- Which module/request is the priority?
- Can KiBuild export/index the schema?

### 3. Create Domain Model

Define:

- modules
- tables/entities
- key fields
- relationships
- calculations
- layouts
- scripts
- tests

### 4. Define Build Phases

Create phased steps:

```text
Phase 1: Schema
Phase 2: Relationships
Phase 3: Layouts
Phase 4: Scripts
Phase 5: Tests
Phase 6: Documentation
```

Each phase should be checklist-ready.

### 5. Define Copy/Paste Artifacts

Identify which outputs are:

```text
pasteable now
manual build steps
future direct-tool steps
requires developer confirmation
```

### 6. Define Validation and Tests

For scripts:

- contract
- test mode
- dry run
- trace expectations
- static validation
- optional FileMaker-backed test

For layouts:

- field references
- button script references
- WebViewer HTML
- layout object checklist

### 7. Write Blueprint Artifact

Write:

```text
outbox/docs/solution_blueprint_{slug}/vNNN_{timestamp}/solution_blueprint.md
outbox/docs/solution_blueprint_{slug}/vNNN_{timestamp}/data_model.json
outbox/docs/solution_blueprint_{slug}/vNNN_{timestamp}/build_steps.md
outbox/docs/solution_blueprint_{slug}/vNNN_{timestamp}/script_plan.md
outbox/docs/solution_blueprint_{slug}/vNNN_{timestamp}/layout_plan.md
outbox/docs/solution_blueprint_{slug}/vNNN_{timestamp}/test_plan.md
```

### 8. Preview and Wait

Preview the blueprint.

Do not proceed to generating scripts/layouts or requesting FileMaker data until the developer approves the blueprint.

## Final Response

```text
Solution blueprint ready: {solution_name}

Mode:
{greenfield/brownfield}

Build phases:
{short list}

Needs approval before:
- generating artifacts
- pulling FileMaker data
- running tests

Blueprint:
{artifact version path}
```

