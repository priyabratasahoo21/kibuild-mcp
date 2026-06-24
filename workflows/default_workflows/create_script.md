---
id: create_script
name: Create FileMaker Script
description: Create a new FileMaker script using project context, valid fmxmlsnippet XML, validation, and versioned outbox output.
allowed_tools:
  - route_request
  - get_context_freshness
  - search_objects
  - find_script
  - find_layout
  - find_table
  - find_field
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
---

# Create Script Workflow

## Purpose

Create a new FileMaker script for the active project using current context, existing conventions, and valid FileMaker clipboard XML.

## Procedure

### 1. Route and Resolve

Identify the likely module, database, and objects involved.

If the request mentions an existing script by name, first call `get_active_context` to get the active database name, then **call `find_script` with the database parameter:**

```json
{ "script_name": "<name>", "database": "<active_database_from_context>" }
```

Always pass `database` — without it `find_script` may match a script in the wrong file.
If the tool returns `AMBIGUOUS:` (multiple equal-score matches), list all candidates and ask the user which database they mean.

The response gives you:
- `txt_path` — sanitized steps → **use for analysis and convention study**
- `xml_path` — raw XML → **read only when generating output XML**
- Sibling scripts — **study these for naming and pattern conventions**

If no specific script is mentioned, use `search_file` to identify the module, then `find_script` to pull related scripts.

Do NOT call `list_dir` to browse scripts — use `find_script`.

### 2. Check Freshness

Check freshness for the selected database/module.

If context is stale or missing, stop and recommend export/index refresh before generation.

### 3. Gather Exact Context

Read compact indexes first:

```text
script_index.json
layout_index.json
table_index.json
field_index.json
dependency_graph.json
```

If the request mentions an existing script, layout, table, or field, resolve it through the indexes and read the exact artifact.

### 4. Study Project Conventions

Search for similar scripts and read one or two exact examples.

Look for:

- naming style
- parameter format
- JSON return format
- error handling
- transaction usage
- layout navigation pattern
- comment style

### 5. Read XML Rules

Call `read_xml_guide` before writing FileMaker XML.

The output must use:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<fmxmlsnippet type="FMObjectList">
  ...
</fmxmlsnippet>
```

No `<Script>` wrapper.

### 6. Design Plain Step List

Draft the script as human-readable FileMaker steps first.

Include:

- comment header
- `Set Error Capture [ On ]`
- parameter parsing when needed
- `test_mode`, `dry_run`, and `trace_id` parsing for non-trivial business scripts
- context preservation when navigation is used
- main logic
- error checks after critical operations
- structured `Exit Script` result

### 7. Generate JSON

Serialize the step list into a valid JSON array of step objects inside a ````json ```` block, conforming to the `fm_xml_serializer` skill.

Hard rules:
- You MUST output a JSON array of step objects, wrapped in a ````json ```` block.
- NEVER output raw XML. The Go sidecar compiles the JSON array to native clipboard XML automatically.
- Calculations must be raw strings without enclosing XML tags.
- Comments are step objects with `"stepName": "Comment"` or `"stepName": "# (comment)"`.

### 8. Validate

Before saving, review your JSON array for:
- All `If`, `Else If`, `Exit Loop If`, and `Set Field` steps have a non-empty `Calculation` parameter.
- All `Set Variable` steps have both `Name` and `Calculation` parameters.
- Step names match the canonical catalog names (use exact casing where possible).

Do NOT call `validate_fmxmlsnippet` on the JSON — it expects compiled XML. Validation runs automatically when `write_outbox_artifact` compiles and saves the artifact. Any compile or validation error is returned immediately so you can fix and retry.

### 9. Write Versioned Outbox Artifact

Call `write_outbox_artifact` with `files` containing ONLY `script.json` (the JSON array payload). Do NOT provide `script.txt` or `script.xml` — the system compiles `.json` → `.xml` and generates `.txt` automatically from the compiled output.

Do NOT manually create output directories — `write_outbox_artifact` handles versioning and folder creation.

If the tool returns a compile error, fix the reported step name or parameter and retry once. Do not fall back to `raw_xml` unless the step genuinely has no template.

## Final Response

Use this format:

```text
Created script: {script_name}

What it does:
{short summary}

Validation:
{passed/failed plus key warnings}

Version:
{artifact version path}
```
