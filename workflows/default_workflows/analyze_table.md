---
id: analyze_table
name: Analyze Table / Fields
description: Analyze a FileMaker table's fields, validation rules, auto-enter calculations, and cross-schema impact. Output is text feedback and impact report only — no FMXML generation.
allowed_tools:
  - get_active_context
  - find_table
  - find_field_references_in_scripts
  - find_field_references_in_layouts
  - find_field_references_in_calculations
  - find_field_references_in_relationships
  - search_index
fallback_tools:
  - read_file
  - search_file
---

# Analyze Table / Fields Workflow

## Purpose

Analyze a FileMaker table's structure — fields, validation, auto-enter calculations, and impact across scripts, layouts, calculations, and relationships. Output is written analysis and recommendations only. This workflow does not generate FMXML or DDR changes, as FileMaker does not support importing field definitions via XML paste.

## Procedure

### 1. Resolve the Table — Silent Tool Call First

**Do NOT ask any questions yet. Resolve the table target before saying anything.**

**Step 1a — Get active context:**
Call `get_active_context` to get the active database name.

**Step 1b — Search for the table:**
Call `find_table` with the name from the user's message and the active database:
```json
{ "table_name": "<name from user request>", "database": "<active_database_from_context>" }
```

**Step 1c — Evaluate the result and ask ONE combined confirmation message:**

If multiple tables match (across files or similar names):
```text
I found multiple tables matching "[name]":

• [Table Name A] — [Database A] ([N] fields)
• [Table Name B] — [Database B] ([N] fields)

Which one did you mean? And what would you like me to look at?
- Field definitions (types, validation, auto-enter)
- Missing or redundant fields
- Impact: which scripts, layouts, or calculations use these fields
- Naming conventions
- Other (describe)
```

If a **single match** is found:
```text
I'll analyze: [Table Name] in [Database] ([N] fields)

What would you like me to focus on?
- Field definitions (types, validation, auto-enter)
- Missing or redundant fields
- Impact: which scripts, layouts, or calculations use these fields
- Naming conventions
- Other (describe)
```

**Wait for the developer's response before proceeding.**

### 2. Read the Table Definition and Run Impact Checks

After confirmation, read the table's field index from `.kibuild_index` using `find_table` results or `search_index`.

Run the impact checks relevant to the developer's focus:
- `find_field_references_in_scripts` — scripts that read/write these fields
- `find_field_references_in_layouts` — layouts that display these fields
- `find_field_references_in_calculations` — calculations (auto-enter, validation, custom functions) that reference them
- `find_field_references_in_relationships` — relationships predicated on these fields

### 3. Analyze and Report

**⚠ Risk classification:** Before reporting, classify each finding:
- **LOW** — cosmetic or naming issue, no cross-schema impact
- **MEDIUM** — affects scripts or layouts but change is localised
- **HIGH** — rename or removal would break calculations, relationships, or multiple layouts/scripts — enumerate every affected location

**Report format:**

```text
Table Analysis: [Table Name] — [Database]
Fields: [N total]

── Field Definitions ────────────────────────────────
[For each field: Name | Type | Validation | Auto-enter | Notes]

── Issues Found ─────────────────────────────────────
[HIGH] Field "[x]" has no validation but is used in [N] scripts as a required parameter.
[MEDIUM] Field "[y]" is referenced in [N] layouts but has no auto-enter default — may produce empty values.
[LOW] Field naming convention inconsistency: "[z]" vs "[ZZ]" style.

── Impact Summary ───────────────────────────────────
Scripts using these fields: [N] — [list key scripts]
Layouts displaying these fields: [N] — [list key layouts]
Calculations referencing these fields: [N]
Relationships predicated on these fields: [N]

── Recommendations ──────────────────────────────────
[Prioritised, plain text]
1. [High] ...
2. [Medium] ...
3. [Low] ...

── Important: How to Apply Changes ─────────────────
FileMaker does not support importing field definitions via XML paste.
Any field changes (rename, type, validation) must be applied manually
in FileMaker's Manage Database dialog.

If a field is renamed, every reference listed above must also be updated.
```

### 4. Offer Next Steps

```text
Next steps:
- If you want to rename a field, say which one — I'll produce a full impact report listing every reference that needs updating.
- For script-level changes, use the refactor or add-to-script workflow.
- For layout-level changes, use the analyze-layout workflow.
```

Do not write to the Outbox. Do not generate any FMXML.
