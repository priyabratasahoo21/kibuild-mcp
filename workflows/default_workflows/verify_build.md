---
id: verify_build
name: Verify Build
description: Re-export/explode FileMaker XML and compare expected build steps against the actual FileMaker solution state.
allowed_tools:
  - get_context_freshness
  - export_schema
  - get_context_status
  - verify_build_step
  - verify_build_phase
  - verify_build_run
  - write_verification_report
fallback_tools:
  - list_dir
  - read_file
  - search_file
required_outputs:
  - verification_report.json
  - verification_report.md
---

# Verify Build Workflow

## Purpose

Verify that developer-followed build steps are actually present in the current FileMaker solution.

The source of truth is the latest FileMaker XML export, not the UI checkbox state.

## Procedure

### 1. Select Scope

Verify one of:

```text
selected step
current phase
full build run
```

### 2. Export and Explode

Prompt the developer to export/refresh schema if needed.

After export:

```text
explode XML
re-index context
load latest indexes
```

### 3. Compare Expected vs Actual

For each expected step, verify against indexes or exact XML.

Check:

- tables
- fields
- relationships
- calculations
- layouts
- layout objects
- WebViewer signatures
- scripts
- generated helper scripts
- test mode/trace support

### 4. Classify Result

Each step becomes:

```text
verified
missing
different
needs_review
failed
skipped
```

### 5. Write Report

Write:

```text
verification_report.json
verification_report.md
```

## Final Response

```text
Verification complete.

Verified:
{count}

Missing:
{short list}

Different:
{short list}

Next:
{recommended fix}
```

