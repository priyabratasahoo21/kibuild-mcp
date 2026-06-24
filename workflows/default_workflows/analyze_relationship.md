---
id: analyze_relationship
name: Analyze Relationship
description: Analyze a FileMaker relationship — predicate correctness, join type, portal impact, and improvement recommendations. Output is text feedback only — no FMXML generation.
allowed_tools:
  - get_active_context
  - inspect_relationships
  - find_relationship_predicates
  - find_to_references
  - find_layout_references_to_tables
  - find_field_references_in_relationships
  - search_index
fallback_tools:
  - read_file
  - search_file
---

# Analyze Relationship Workflow

## Purpose

Analyze a FileMaker relationship between two table occurrences — its predicates, join direction, portal usage, and cross-schema impact. Output is written analysis and recommendations only. FileMaker relationships cannot be imported via XML paste; all changes must be applied manually in the Relationship Graph.

## Procedure

### 1. Resolve the Relationship — Silent Tool Call First

**Do NOT ask any questions yet. Resolve the target before saying anything.**

**Step 1a — Get active context:**
Call `get_active_context` to get the active database name.

**Step 1b — Search for the relationship:**
Relationships are identified by table occurrence names or portal names. Search using:
- `inspect_relationships(database)` — returns all relationships in the file
- `search_index(query: "<table name or relationship name>")` — locate by name

**Step 1c — Evaluate and ask ONE combined confirmation message:**

If multiple relationships match (same table in multiple joins, or similar names):
```text
I found several relationships involving "[name]":

• [TableOccurrenceA] → [TableOccurrenceB] — predicate: [FieldA = FieldB]
• [TableOccurrenceC] → [TableOccurrenceA] — predicate: [FieldC = FieldD]

Which relationship did you mean? And what would you like me to look at?
- Predicate correctness (are the right fields joined?)
- Join type (equality, inequality, cartesian × issue)
- Portal filtering and sort
- Whether allow-create / allow-delete is correctly set
- Impact: which layouts show portals through this relationship
- Other (describe)
```

If a **single relationship** is found:
```text
I'll analyze the relationship: [TableOccurrenceA] → [TableOccurrenceB] in [Database]
Predicate: [FieldA = FieldB]

What would you like me to focus on?
- Predicate correctness
- Join type
- Portal filtering and sort
- Allow-create / allow-delete settings
- Cross-schema impact (which layouts and scripts use this relationship)
- Other (describe)
```

**Wait for the developer's response before proceeding.**

### 2. Read the Relationship Definition and Run Impact Checks

After confirmation:
1. Call `find_relationship_predicates` to get the full join definition.
2. Call `find_to_references` to find all table occurrences that reference the join table.
3. Call `find_layout_references_to_tables` to identify layouts that expose portals through this relationship.
4. Call `find_field_references_in_relationships` to check if any shared fields are predicate components elsewhere.

### 3. Analyze and Report

**Report format:**

```text
Relationship Analysis: [TableOccurrenceA] → [TableOccurrenceB] — [Database]

── Predicate ────────────────────────────────────────
[TableOccurrenceA]::[FieldA] [operator] [TableOccurrenceB]::[FieldB]
Join type: [equality / inequality / cartesian / range]

── Assessment ───────────────────────────────────────
[OK / WARNING / ERROR] [explanation]

Examples of issues to flag:
- Cartesian product (×) where equality was likely intended
- Join on a text field that may have case or whitespace mismatches
- Multiple predicates where the logic may produce unintended AND/OR behaviour
- Missing predicate (effectively a cartesian join) creating performance risk
- Allow-create enabled on a relationship that should be read-only

── Allow Create / Allow Delete ──────────────────────
Allow create: [Yes / No] — [assessment: appropriate / risky / missing]
Allow delete: [Yes / No] — [assessment]

── Portal Impact ────────────────────────────────────
Layouts with portals through this relationship: [N]
• [Layout Name] — [Database] (portal filter: [Yes/No], sort: [Yes/No])

── Affected Scripts ─────────────────────────────────
Scripts that navigate via this relationship: [list or "none found"]

── Issues Found ─────────────────────────────────────
[Numbered, severity-tagged list]
1. [HIGH] ...
2. [MEDIUM] ...
3. [LOW] ...

── Recommendations ──────────────────────────────────
[Prioritised, plain text — specific predicate changes, not generic advice]

── How to Apply Changes ─────────────────────────────
FileMaker relationships cannot be imported via XML.
Changes must be applied manually in the Relationship Graph
(File → Manage → Database → Relationships tab).
```

### 4. Offer Next Steps

```text
Next steps:
- If you want to add or change a predicate, describe what you need — I'll write the exact definition.
- For portal scripts triggered from this layout, use the analyze-layout or refactor-script workflow.
```

Do not write to the Outbox. Do not generate any FMXML.
