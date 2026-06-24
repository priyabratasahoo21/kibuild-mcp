---
id: document_solution
name: Document FileMaker Solution
description: Generate documentation for a solution, module, database, script group, or layout group.
allowed_tools:
  - route_request
  - get_context_freshness
  - get_module_context
  - get_database_context
  - search_objects
  - get_dependencies
  - get_impact_report
  - read_artifact
  - write_outbox_artifact
fallback_tools:
  - list_dir
  - read_file
  - search_file
required_outputs:
  - documentation.md
---

# Document Solution Workflow

## Purpose

Generate human-readable documentation for a FileMaker solution or a scoped part of it.

## Procedure

### 1. Select Scope

Resolve whether the developer wants documentation for:

- whole solution
- module
- database
- script folder
- table group
- layout group
- specific object

### 2. Check Freshness

Report freshness in the documentation metadata.

### 3. Use Summaries and Indexes First

Read:

```text
solution_map.md
module_summary.md
file_summary.md
script_index.json
layout_index.json
table_index.json
dependency_graph.json
```

Read exact XML only for important or ambiguous objects.

### 4. Generate Documentation

Include:

- purpose
- main modules/files
- key scripts
- important layouts
- core tables
- dependencies
- integration points
- known risks or human notes

### 5. Write Versioned Documentation Artifact

Write to:

```text
outbox/docs/{scope_slug}/vNNN_{timestamp}/documentation.md
```

## Final Response

```text
Documentation created: {scope}

Sections:
{short list}

Version:
{artifact version path}
```

