---
id: create_layout
name: Create FileMaker Layout
description: Design or generate a FileMaker layout proposal using schema context and existing layout conventions.
allowed_tools:
  - route_request
  - get_context_freshness
  - find_layout
  - find_table
  - find_field
  - get_layout_context
  - get_table_context
  - read_artifact
  - validate_layout_references
  - write_outbox_artifact
fallback_tools:
  - list_dir
  - read_file
  - search_file
required_outputs:
  - layout_plan.md
  - layout.json
  - context_snapshot.json
  - validation.json
---

# Create Layout Workflow

## Purpose

Create a FileMaker layout proposal using project schema, target table occurrence, existing layout conventions, and validation.

## Procedure

### 1. Resolve Target

Identify:

- module
- database
- base table
- table occurrence
- layout purpose
- target device or size, if known

### 2. Check Freshness

Stop if layout/table/field context is stale and the task requires exact references.

### 3. Study Existing Layouts

Find similar layouts in the same module/database.

Inspect:

- naming pattern
- theme/style pattern
- portal usage
- button scripts
- field grouping
- navigation controls
- privilege-dependent elements

### 4. Select Fields and Interactions

Use `field_index.json` and table context to choose fields.

Do not invent fields.

Define:

- header fields
- primary body fields
- portals
- actions/buttons
- script triggers
- validation messages

### 5. Generate Layout Proposal

Until direct layout write tools are available, generate:

```text
layout_plan.md
layout.json
```

**IMPORTANT**: Do NOT generate XML for Layouts, as layout XML is not supported for clipboard operations in this framework. Only generate the plan and JSON representation.

If FileMaker/Claris direct layout creation tools become available, this workflow can call those tools after preview/approval.

### 6. Validate References

Validate fields, table occurrences, and scripts used by buttons/triggers.

### 7. Write Versioned Outbox Artifact

Write to:

```text
outbox/layouts/{layout_slug}/vNNN_{timestamp}/
```

## Final Response

```text
Created layout proposal: {layout_name}

Purpose:
{short summary}

Validation:
{passed/failed}

Version:
{artifact version path}
```

