---
id: explore_schema_object
name: Explore Schema Object
description: Entry point when the object type or target is ambiguous. Searches across all schema object types, resolves exact match, confirms with the developer, then routes to the appropriate analysis or modification workflow.
allowed_tools:
  - get_active_context
  - find_script
  - find_layout
  - find_table
  - inspect_relationships
  - search_index
  - read_layout
  - xml_extract_steps
fallback_tools:
  - list_dir
  - read_file
  - search_file
---

# Explore Schema Object Workflow

## Purpose

Used when the developer's message refers to a named FileMaker object (script, layout, table, value list, relationship) but does not make the type or exact target explicit enough to route directly. This workflow searches across all types, presents all matches, and asks one combined confirmation before proceeding.

## Procedure

### 1. Resolve Active Context — Silent

Call `get_active_context` to get the active database name. Do not output anything to the developer yet.

### 2. Search Across All Object Types — Silent

Using the name or description from the user's message, search across all relevant object types in parallel:

- `find_script(name, database)` — scripts
- `find_layout(name, database)` — layouts
- `find_table(name, database)` — tables / fields
- `search_index(query: name)` — catches value lists and other objects not covered above

Collect every result. If the active database is empty, search without the database filter — results will span all files.

### 3. Ask ONE Combined Clarification Message

After collecting all results, present everything in a single message. Do not make multiple back-and-forth questions.

**Template when results span multiple types:**
```text
I found "[name]" in several places:

Scripts:
• [Script Name] — [Database] (folder: [folder])

Layouts:
• [Layout Name] — [Database]

Tables:
• [Table Name] — [Database]

Which one did you mean, and what would you like to do with it?
```

**Template when results span multiple files (same type):**
```text
I found "[name]" in multiple files:

• [Script Name] — [Database A] (folder: [folder])
• [Script Name] — [Database B] (folder: [folder])

Which file did you mean, and what would you like to do with it?
```

**Template when a single match is found:**
```text
I found one match: [Type] "[Name]" in [Database].

Is this what you meant? If so, what would you like to do with it?
- Analyze / explain what it does
- Suggest improvements
- Check for issues or risks
- Other (describe)
```

**Template when nothing is found:**
```text
I couldn't find anything matching "[name]" in [Database].

Could you clarify:
1. The exact name of the script, layout, table, or value list?
2. Which FileMaker file it lives in?
```

**Wait for the developer's response. Do not read file contents, generate code, or take any further action until confirmed.**

### 4. Route to the Appropriate Specialist Workflow

Once the developer confirms the object type and target, continue using the procedure for the corresponding specialist workflow:

| Object type confirmed | Follow procedure from |
|---|---|
| Script — improvement/refactor | `refactor_script` workflow Step 2 onward |
| Script — add steps | `add_to_script` workflow Step 2 onward |
| Script — analyze/explain | `analyze_script` workflow Step 2 onward |
| Layout | `analyze_layout` workflow Step 2 onward |
| Table / Fields | `analyze_table` workflow Step 2 onward |
| Relationship | `analyze_relationship` workflow Step 2 onward |
| Value List | `analyze_valuelist` workflow Step 2 onward |

Do not re-ask which object — the developer already confirmed it. Pick up directly at the read/analysis phase of the target workflow.
