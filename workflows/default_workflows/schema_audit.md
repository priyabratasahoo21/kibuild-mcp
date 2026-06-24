---
id: schema_audit
name: Schema Audit
description: Audit a module or database schema for structure, naming, dependencies, and risk.
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
  - audit_report.md
---

# Schema Audit Workflow

## Purpose

Audit a FileMaker module or database for schema health, naming consistency, dependency risk, and maintainability.

## Procedure

### 1. Select Scope

Scope can be:

- whole solution
- module
- database
- table
- field group
- relationship graph area

If scope is unclear, use active context or ask the developer.

### 2. Check Freshness

Do not audit stale schema silently.

### 3. Read Indexes First

Use structured indexes:

```text
table_index.json
field_index.json
relationship_graph.json
dependency_graph.json
module_index.json
```

Only read exact XML for high-risk or unclear objects.

### 4. Audit Dimensions

Check:

- naming consistency
- orphaned or duplicate fields
- unclear table occurrence naming
- layout/table occurrence mismatch
- scripts touching unexpected tables
- cross-module coupling
- unused or high-risk relationships
- calculated fields with heavy dependencies
- fields/scripts/layouts marked do-not-modify

### 5. Write Audit Report

Write versioned report to:

```text
outbox/docs/schema_audit_{scope_slug}/vNNN_{timestamp}/audit_report.md
```

## Final Response

```text
Schema audit complete: {scope}

Top findings:
{short list}

Report:
{artifact version path}
```

