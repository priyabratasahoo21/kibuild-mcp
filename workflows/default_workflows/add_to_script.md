---
id: add_to_script
name: Add to FileMaker Script
description: Append or insert steps into an existing FileMaker script using proper FM conventions — comment header, error handling, and signature block.
allowed_tools:
  - route_request
  - get_context_freshness
  - find_script
  - get_dependencies
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
  - prompt.md
  - workflow.md
  - context_snapshot.json
  - validation.json
  - diff.json
---

# Add to Script Workflow

## Purpose

Insert or append new logic into an existing FileMaker script without breaking existing behavior.

## Procedure

### 1. Confirm Intent — Ask First

**Do NOT generate any code yet.**

Ask the developer exactly what should be added:

```text
What would you like to add to this script?
- New logic block (e.g. "add error check after Commit Records")
- New parameter handling (e.g. "add support for dry_run mode")
- New comment section (e.g. "add a section header before the navigation block")
- Other (describe)

Where should it be inserted?
- At the top
- Before a specific step (name it)
- After a specific step (name it)
- At the bottom
```

Wait for the developer's response before proceeding.

### 2. Resolve the Script

**Call `find_script` immediately:**

```json
{ "script_name": "<name from user request>", "database": "<db if known>" }
```

The response gives you:
- `txt_path` — sanitized steps → **use to read current script content**
- `xml_path` — raw XML → **read only when generating the final replacement XML**
- Sibling scripts — understand what else lives in this module

If multiple candidates return, list them and ask the developer to confirm.

Do NOT use `list_dir` or `search_file` to find scripts.

### 3. Check Impact

```text
get_dependencies(script)
get_called_by(script)
```

If the script is high-risk or cross-module, warn before modifying.

### 4. Design the Addition

Plan the exact steps to add following **Pro Scriptwriter** standards:

- New logic blocks start with a `# ── Section ──` comment step
- All new calculations use `$localVars` not global vars unless explicitly required
- New error checks use `Get ( LastError )` immediately after critical operations
- Preserve all existing steps — do not remove or reorder them

### 5. Generate FM Script Template

Every addition must follow the ideal FileMaker script template:

```text
# ── [Section Name] ──
# Added by: [script context]
# Purpose: [one sentence]

[new steps here]
```

### 6. Generate Replacement JSON

Produce a full replacement JSON array of step objects of the entire script with the new steps inserted, conforming to the `fm_xml_serializer` skill.
- You MUST output a JSON array of step objects, wrapped in a ````json ```` block.
- NEVER output raw XML. The Go sidecar compiles the JSON array to native clipboard XML automatically.
- Calculations must be raw strings without enclosing XML tags.
- Comments are step objects with `"stepName": "Comment"` or `"stepName": "# (comment)"`.

### 7. Validate

Validate your JSON step array using `fm_xml_validator` standard rules.

Create `diff.json` showing only the added steps.

### 8. Write Versioned Outbox Artifact

Call `write_outbox_artifact` with:
- `script.json` — full updated JSON array payload (the entire script, not just the added steps)
- `diff.json` — only the added steps delta

Do NOT provide `script.txt` — it is generated automatically from the compiled XML.

If the tool returns a compile error, fix the reported step and retry. Do not fall back to `raw_xml` unless the step has no template.

## Final Response

```text
Added to script: {script_name}

Added:
{summary of what was inserted and where}

Validation:
{passed/failed}

Version:
{artifact version path}
```
