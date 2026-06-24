---
id: analyze_script
name: Analyze FileMaker Script
description: Explain an existing FileMaker script, identify risks, and recommend improvements without generating replacement XML unless requested.
allowed_tools:
  - route_request
  - get_context_freshness
  - find_script
  - get_dependencies
  - get_called_by
  - get_impact_report
  - read_artifact
fallback_tools:
  - list_dir
  - read_file
  - search_file
required_outputs:
  - analysis_report
---

# Analyze Script Workflow

## Purpose

Analyze a FileMaker script for purpose, flow, dependencies, complexity, error handling, and risk.

## Procedure

### 1. Resolve Script

Find the exact script from active context, user text, or search.

If ambiguous, ask the developer to choose.

### 2. Check Freshness

Report whether the analysis uses fresh or stale context.

### 3. Read Exact Script

Read the exact XML/readable artifact before analysis.

### 4. Gather Dependencies

Use dependency tools to identify:

- scripts called by this script
- scripts that call this script
- layouts used
- fields/tables touched
- cross-module dependencies

### 5. Produce Analysis

Report:

- purpose
- step summary
- inputs and outputs
- key dependencies
- error handling quality
- transaction safety
- hardcoded values
- missing validation
- deprecated or risky patterns
- verdict

### 6. Preview Requests — Return Outbox Links, Not Text Dumps

If the user asks to **show**, **preview**, **display**, or **see** a script that was recently generated or already exists in the Outbox:

1. Use `search_file` on the Outbox directory to find the latest version path for the script (look for `*_latest.xml` or the highest-numbered `Version N` folder).
2. Return the outbox file links in this format — **do NOT copy-paste the script content as a numbered text list**:

```
Here is the latest version of **{ScriptName}**:

- [📄 {ScriptName}_Version N.txt](file:///path/to/Outbox/scripts/{slug}/Version N/{ScriptName}_Version N.txt)
- [📋 {ScriptName}_Version N.xml](file:///path/to/Outbox/scripts/{slug}/Version N/{ScriptName}_Version N.xml)
```

Returning `file:///` links activates the XMLCodeBlock component in the UI, which shows TXT/FMXML tabs and the Copy FM Steps button. Pasting the script as plain text bypasses this entirely — **never do this for preview requests**.

### 7. Use `write_outbox_artifact` for Structured Reports Only

Use `write_outbox_artifact` only when producing a new analysis report artifact, not for showing already-generated scripts.

## Final Response

```text
Script: {script_name}

Purpose:
{one sentence}

Flow:
{short summary}

Risks:
{highest signal issues}

Verdict:
{production-ready / needs improvement / high risk}
```

