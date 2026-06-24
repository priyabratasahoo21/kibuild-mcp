---
id: refactor_script
name: Refactor FileMaker Script
description: Improve an existing FileMaker script while preserving intent and producing a versioned, validated replacement.
allowed_tools:
  - route_request
  - get_context_freshness
  - find_script
  - get_dependencies
  - get_called_by
  - get_impact_report
  - read_artifact
  - read_xml_guide
  - validate_fmxmlsnippet
  - write_outbox_artifact
fallback_tools:
  - list_dir
  - read_file
  - search_file
required_outputs:
  - script.json
  - diff.json
---

# Refactor Script Workflow

## Purpose

Refactor an existing FileMaker script for reliability, readability, error handling, or maintainability without changing intended behavior unless explicitly requested.

## Procedure

### 1. Resolve the Script — Silent Tool Call First

**Do NOT ask any questions yet. Resolve the script target before saying anything.**

**Step 1a — Get active context:**
Call `get_active_context` to get the active database name.

**Step 1b — Search for the script:**
Call `find_script` with the name from the user's message and the active database:
```json
{ "script_name": "<name from user request>", "database": "<active_database_from_context>" }
```

**Step 1c — Evaluate the result and ask ONE combined confirmation message:**

If `find_script` returns `AMBIGUOUS:` (multiple equal-score matches across files or within the same file):
```text
I found multiple scripts matching "[name]":

• [Script Name A] — [Database A] (folder: [folder])
• [Script Name B] — [Database B] (folder: [folder])
• [Script Name C] — [Database A] (folder: [folder])

Which one did you mean? And what would you like improved?
- Error handling (Set Error Capture, LastError checks)
- Parameter validation
- Transaction safety (Open/Commit/Revert Transaction)
- Comment headers and documentation
- Reduce redundancy / unnecessary steps
- Performance (fewer layout hops, fewer SQL calls)
- Structured JSON exit result
- Other (describe)
```

If `find_script` returns a **single match**, confirm it and ask what to improve in one message:
```text
I'll be working on: [Script Name] in [Database] (folder: [folder path])

What would you like improved?
- Error handling (Set Error Capture, LastError checks)
- Parameter validation
- Transaction safety (Open/Commit/Revert Transaction)
- Comment headers and documentation
- Reduce redundancy / unnecessary steps
- Performance (fewer layout hops, fewer SQL calls)
- Structured JSON exit result
- Other (describe)
```

**Wait for the developer's confirmation before proceeding to Step 2.**

Always pass `database` to `find_script` — never omit it. Without it, results may span the wrong file.

### 2. Check Freshness and Impact

Check context freshness.

Run impact checks:

```text
get_dependencies(script)
get_called_by(script)
get_impact_report(script)
```

If the script is high risk or cross-module, report that before generating a new version.

### 3. Read Exact Source

**Only execute this step after the developer has confirmed the target script and improvement intent.**

Read the exact script XML and any closely related scripts it calls or depends on.

Do not refactor from an index summary alone.

### 4. Explain Current Behavior

Before writing, summarize:

- purpose
- main steps
- inputs and outputs
- dependencies
- current risks

### 5. Design Changes

Refactor only what the request requires.

Common improvements:

- add `Set Error Capture [ On ]`
- add parameter validation
- add real Comment steps
- add error checks after critical operations
- add structured JSON result
- reduce duplicate blocks
- preserve/restore original context
- add transaction handling where appropriate
### 6. Generate Replacement JSON

Generate a replacement JSON array of step objects inside a ````json ```` block, conforming to the `fm_xml_serializer` skill.
- You MUST output a JSON array of step objects, wrapped in a ````json ```` block.
- NEVER output raw XML. The Go sidecar compiles the JSON array to native clipboard XML automatically.
- Calculations must be raw strings without enclosing XML tags.
- Comments are step objects with `"stepName": "Comment"` or `"stepName": "# (comment)"`.

Do not return the same script unchanged. If no change is justified, state that and stop.

### 7. Validate and Diff

Validate your JSON step array using `fm_xml_validator` standard rules.

Create `diff.json` comparing original vs proposed step list:

```text
added
changed
removed
risk_notes
```

### 8. Write Versioned Outbox Artifact

Call `write_outbox_artifact` with:
- `script.json` — the full replacement JSON array payload
- `diff.json` — comparison of original vs proposed steps (added, changed, removed, risk_notes)

Do NOT provide `script.txt` — it is generated automatically from the compiled XML.
Do NOT manually create output directories — `write_outbox_artifact` handles versioning and folder creation.

If the tool returns a compile error, fix the reported step and retry once before escalating to the developer.

## Final Response

```text
Refactored script: {script_name}

Changed:
{short step-level delta}

Impact:
{dependencies/callers/risk}

Validation:
{passed/failed}

Version:
{artifact version path}
```
