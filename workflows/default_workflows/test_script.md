---
id: test_script
name: Test FileMaker Script
description: Validate and test a generated or existing FileMaker script using static checks, offline simulation, and optional FileMaker-backed tests.
allowed_tools:
  - route_request
  - get_context_freshness
  - find_script
  - get_dependencies
  - read_artifact
  - validate_fmxmlsnippet
  - validate_script_contract
  - generate_test_cases
  - simulate_script
  - request_test_data_access
  - pull_test_fixture
  - run_filemaker_script_test
  - compare_test_result
  - write_outbox_artifact
fallback_tools:
  - list_dir
  - read_file
  - search_file
required_outputs:
  - contract.json
  - tests/cases/*.json
  - tests/results/static_validation.json
  - tests/results/offline_simulation.json
---

# Test Script Workflow

## Purpose

Test a FileMaker script before the developer applies or trusts it.

This workflow supports three levels:

```text
static validation
offline simulation
FileMaker-backed test with user confirmation
```

## Procedure

### 1. Resolve Script or Artifact

Find the target script from:

- active preview
- outbox artifact version
- script name
- active FileMaker context

If ambiguous, ask the developer to choose.

### 2. Load Contract

Read `contract.json` if present.

If missing, generate a draft contract from:

- script name
- script step list
- user request
- dependency graph
- referenced tables/layouts/scripts

### 3. Run Static Validation

Validate:

- XML structure
- parameter handling
- exit paths
- referenced scripts/layouts/fields
- contract consistency
- risky operations

Write:

```text
tests/results/static_validation.json
```

### 4. Generate Test Cases

Generate at least:

- one success case
- one missing/invalid parameter case
- one missing record/reference case
- one expected FileMaker error path, if relevant

Write cases under:

```text
tests/cases/
```

### 5. Offline Simulation

Use fixtures when available.

If fixtures are missing, generate synthetic fixtures from schema context.

Run simulation for:

- JSON input validation
- branch logic
- expected status/result shape
- dry-run planned changes
- `test_mode` behavior
- trace output shape and expected trace checkpoints

Write:

```text
tests/results/offline_simulation.json
```

### 6. Ask Before FileMaker-Backed Tests

If the script needs real FileMaker data, explain the required tables, fields, row limits, and storage/anonymization choice.

Do not pull data without developer confirmation.

### 7. Pull Fixture or Run Test

After confirmation:

- pull selected fixture data, or
- run a safe FileMaker-backed test, or
- run dry-run mode against selected records
- pass `test_mode=true`, `dry_run=true` when safe, and a unique `trace_id`
- compare returned trace checkpoints with expected test case checkpoints

Write:

```text
tests/results/filemaker_backed_test.json
```

### 8. Report Results

Show:

- passed/failed checks
- expected vs actual result
- data access used
- warnings
- next recommended fix

Preview the result report.

## Final Response

```text
Tested script: {script_name}

Results:
- Static validation: {passed/failed}
- Offline simulation: {passed/failed}
- FileMaker-backed test: {not run/passed/failed}

Important:
{warnings or required fixes}

Report:
{artifact version path}
```
