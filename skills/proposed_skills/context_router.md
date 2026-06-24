---
id: context_router
name: Context Router
desc: Route user requests to the correct FileMaker module, database, and objects
color: #14b8a6
letter: R
---

## Purpose

Enforce efficient routing of developer queries to the correct FileMaker database, script folder, module, or schema asset. Prevents scanning unnecessary code or database files.

---

## Routing Principles

1. **Leverage Active Context**
   - Inspect the active FileMaker document name and current context window.
   - If the request targets "this script" or "current layout," resolve the name immediately.

2. **Index-First Search**
   - Query schema index files (`script_index.json`, `layout_index.json`) before traversing full XML subdirectories.
   - Match by exact name first, then by prefix or folder name.

3. **Database Segregation**
   - In multi-file FileMaker solutions, separate resources by database folder boundaries.
   - Do not query or update scripts in `Database_B` when requested to change `Database_A` unless a cross-file relationship or script call is explicitly identified.

4. **Ambiguity Resolution**
   - If multiple objects share a name (e.g. `Log_Event` script exists in both User and Admin folders), halt and ask the developer to clarify the folder path.

5. **Resource Bounds Guard**
   - Never load more than 5 script XML files concurrently during a single step analysis task.
