---
id: script_modularity_architect
name: Script Modularity Architect
desc: Split generated FileMaker logic into modular, testable script suites
color: #8b5cf6
letter: M
---

## Purpose

Enforce modular script design principles. Prevent monolithic scripts by dividing logic into single-purpose, highly cohesive layers.

---

## Architectural Layers

1. **Public Entry Scripts**
   - Bind to layout buttons or triggers.
   - Responsible for fetching user choices, validation parameters, showing custom dialogs, and handling errors.
   - Never perform direct database updates. Delegate immediately to Worker scripts.

2. **Worker Scripts**
   - Execute core business logic.
   - Must be UI-neutral (no Custom Dialogs, no layouts shifts unless layout context is required for database actions).
   - Use JSON inputs and outputs.
   - Support `test_mode` and `dry_run` parameters.

3. **Repository/Data Scripts**
   - Handle database reads/writes for a specific table.
   - Example: `Invoice_Create`, `Invoice_Update_Status`.
   - Wrap operations in transactions.

4. **Utility/Helper Scripts**
   - Pure helpers (e.g. calculation formatting, string parsing, JSON normalization).
   - Independent of layouts or contexts.

---

## Script Interface & Boundaries

- **Input**: Passing parameters as a structured JSON object is mandatory:
  ```json
  {
    "param1": "value",
    "param2": 123
  }
  ```
- **Output**: Return script result as a structured JSON object containing a `status` (success/error), `error_code`, and `result` payload.
- **Length limit**: Keep scripts under 50 steps. Split into sub-scripts if complexity grows.
