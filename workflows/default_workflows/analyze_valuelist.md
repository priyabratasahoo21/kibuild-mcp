---
id: analyze_valuelist
name: Analyze Value List
description: Analyze a FileMaker value list — source type, usage across layouts and scripts, potential issues, and improvement recommendations. Output is text feedback only — no FMXML generation.
allowed_tools:
  - get_active_context
  - search_index
  - find_valuelist_references_in_calculations
  - find_layout_references_to_valuelists
  - find_script_references_to_valuelists
  - read_file
fallback_tools:
  - search_file
  - list_dir
---

# Analyze Value List Workflow

## Purpose

Analyze a FileMaker value list — whether static or dynamic, how it is used across layouts and scripts, and what improvements or corrections are recommended. Output is written analysis only. Value list XML can be generated separately if needed, but this workflow focuses on analysis and feedback.

## Procedure

### 1. Resolve the Value List — Silent Tool Call First

**Do NOT ask any questions yet. Resolve the target before saying anything.**

**Step 1a — Get active context:**
Call `get_active_context` to get the active database name.

**Step 1b — Search for the value list:**
Value lists do not have a dedicated `find_valuelist` tool. Use:
- `search_index(query: "<value list name>")` — searches the full index including value lists
- `find_layout_references_to_valuelists(database)` — find all layouts using value lists, then filter by name
- `find_valuelist_references_in_calculations(database)` — find calculation references

**Step 1c — Evaluate and ask ONE combined confirmation message:**

If multiple value lists match (same or similar names across files):
```text
I found multiple value lists matching "[name]":

• "[Value List A]" — [Database A] (type: static / dynamic)
• "[Value List B]" — [Database B] (type: static / dynamic)

Which one did you mean? And what would you like me to look at?
- Source type and values (static list review)
- Dynamic source field and display field setup
- Which layouts and fields use this value list
- Which scripts reference it
- Issues or inconsistencies
- Other (describe)
```

If a **single match** is found:
```text
I'll analyze value list: "[Value List Name]" in [Database] (type: [static/dynamic])

What would you like me to focus on?
- Source type and values
- Usage across layouts and fields
- Script references
- Issues or inconsistencies
- Other (describe)
```

**Wait for the developer's response before proceeding.**

### 2. Read the Value List Definition and Run Usage Checks

After confirmation:
1. Read the value list definition file using `read_file` (from `.kibuild_index` or schema folder).
2. Run usage checks based on the developer's focus:
   - `find_layout_references_to_valuelists` — which layouts and fields use this value list
   - `find_valuelist_references_in_calculations` — calculations that reference this list's values
   - `find_script_references_to_valuelists` — scripts that set fields controlled by this list

### 3. Analyze and Report

**Report format:**

```text
Value List Analysis: "[Value List Name]" — [Database]

── Definition ───────────────────────────────────────
Type: [Static / Dynamic — from field]

If static:
  Values: [list all values, one per line]

If dynamic:
  Source table: [table name]
  Source field: [field used for the list]
  Display field: [second field shown, or "none"]
  Sort: [Yes/No — by what field]

── Usage ────────────────────────────────────────────
Layouts using this value list: [N]
• [Layout Name] — field "[field name]" ([control style: popup / checkbox / radio])

Scripts referencing this value list: [N]
• [Script Name] — [how it's used]

Calculation references: [N or "none"]

── Issues Found ─────────────────────────────────────
[Numbered, severity-tagged]

Examples of issues to flag:
- [HIGH] Dynamic source field is a text field with no sort — list order is unpredictable
- [HIGH] Value list used in a checkbox set but source has >50 values — performance risk
- [MEDIUM] Value list used in script validation but script uses a hardcoded string comparison instead of referencing the list
- [LOW] Static list has duplicate or inconsistently capitalised values
- [LOW] Value list is defined but not used in any layout or script

── Recommendations ──────────────────────────────────
[Prioritised, specific, plain text]
1. [High] ...
2. [Medium] ...
3. [Low] ...

── How to Apply Changes ─────────────────────────────
Static list changes (add/remove/reorder values) can be applied manually
in FileMaker's Manage Value Lists dialog (File → Manage → Value Lists).

Dynamic source changes require updating the source table or field reference
in the value list definition.
```

### 4. Offer Next Steps

```text
Next steps:
- If you want to add, remove, or reorder values in a static list, describe the change — I can write the updated list.
- If the value list drives a script validation, use the refactor-script workflow to align the script logic.
- For layout control-style issues (popup vs. checkbox), use the analyze-layout workflow.
```

Do not write to the Outbox. Do not generate any FMXML.
