---
id: analyze_layout
name: Analyze Layout
description: Analyze an existing FileMaker layout for structural issues, script trigger problems, field placement, portal configuration, and improvement opportunities. Output is text feedback only — no FMXML generation.
allowed_tools:
  - get_active_context
  - find_layout
  - read_layout
  - find_layout_references_to_scripts
  - find_layout_references_to_tables
  - find_layout_references_to_valuelists
  - find_script_references_in_layouts
  - find_field_references_in_layouts
fallback_tools:
  - search_index
  - read_file
  - search_file
---

# Analyze Layout Workflow

## Purpose

Analyze an existing FileMaker layout and provide clear, actionable text feedback. This workflow produces written analysis and improvement recommendations only — it does not generate FMXML or write Outbox artifacts.

## Procedure

### 1. Resolve the Layout — Silent Tool Call First

**Do NOT ask any questions yet. Resolve the layout target before saying anything.**

**Step 1a — Get active context:**
Call `get_active_context` to get the active database name.

**Step 1b — Search for the layout:**
Call `find_layout` with the name from the user's message and the active database:
```json
{ "layout_name": "<name from user request>", "database": "<active_database_from_context>" }
```

**Step 1c — Evaluate the result and ask ONE combined confirmation message:**

If `find_layout` returns multiple matches (across files or similar names):
```text
I found multiple layouts matching "[name]":

• [Layout Name A] — [Database A]
• [Layout Name B] — [Database B]
• [Layout Name C] — [Database A]

Which one did you mean? And what aspect would you like me to look at?
- Script triggers (OnObjectEnter, OnObjectSave, etc.)
- Field placement and missing fields
- Portal configuration and filtering
- Value list assignments
- Tab order
- General structure and issues
- Other (describe)
```

If a **single match** is found:
```text
I'll analyze: [Layout Name] in [Database] (based on table: [table occurrence])

What would you like me to focus on?
- Script triggers (OnObjectEnter, OnObjectSave, etc.)
- Field placement and missing fields
- Portal configuration and filtering
- Value list assignments
- Tab order
- General structure and issues
- Other (describe)
```

**Wait for the developer's response before proceeding.**

### 2. Read the Layout and Run Impact Checks

After confirmation:

1. Call `read_layout` to get the full layout definition — fields, portals, script triggers, and layout objects.
2. Run reference checks relevant to the focus areas confirmed by the developer:
   - `find_layout_references_to_scripts` — which scripts are triggered from this layout
   - `find_layout_references_to_tables` — which table occurrences are used
   - `find_layout_references_to_valuelists` — which value lists are attached to fields
   - `find_field_references_in_layouts` — cross-check field usage

### 3. Analyze and Report

Produce a structured text report covering the focus areas the developer selected. Do not generate FMXML.

**Report format:**

```text
Layout Analysis: [Layout Name] — [Database]
Based on table occurrence: [table occurrence name]

── Script Triggers ──────────────────────────────────
[List each trigger: event → script name → assessment: OK / Missing / Wrong script]

── Fields ───────────────────────────────────────────
[List fields present, flag any that are missing from the table, duplicated, or misaligned]

── Portals ──────────────────────────────────────────
[For each portal: related table, filter predicate, sort, issue if any]

── Value Lists ──────────────────────────────────────
[Field → value list name → source type (static/dynamic) → potential issue]

── Issues Found ─────────────────────────────────────
[Numbered list of concrete problems]
1. [Issue]: [explanation] → [recommended fix]
2. ...

── Recommendations ──────────────────────────────────
[Prioritized list of improvements in plain text]
1. [High] ...
2. [Medium] ...
3. [Low] ...
```

### 4. Offer Next Steps

End the response with:
```text
Next steps:
- If you'd like me to look at a specific script trigger in detail, say which one.
- If you want a deeper field-by-field review, let me know.
- Script changes (if any are needed) can be handled via the refactor or add-to-script workflow.
```

Do not write to the Outbox. Do not generate any FMXML.
