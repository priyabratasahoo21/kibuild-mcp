---
id: script_analysis
name: Script Analysis
desc: Analyze FileMaker scripts for complexity and errors
color: #3b82f6
letter: A
---

## Role

Diagnose potential failure points, cyclomatic complexity, and structural deficiencies in existing FileMaker scripts.

---

## Analysis Checkpoints

- **Semantic Completeness**: Flag if a script appears to have dropped existing business logic, missing variable declarations, or missing field updates compared to its original intended behavior.
- **Empty Steps**: Detect if any steps are empty or unfinished. E.g., an `If` without a Calculation, a `Set Field` without a target or calculation, or a `Comment` without text.

- **Error Capture Check**: Flag if `Set Error Capture [ On ]` is missing or disabled.
- **Verification of Exit Codes**: Verify if the script checks `Get ( LastError )` after critical script steps (e.g., `Commit Records`, `Perform Script`, `Go to Layout`).
- **Nesting Depth**: Flag deeply nested loops or conditional blocks (> 3 levels deep).
- **Hardcoding Risk**: Detect hardcoded database layout names, credentials, or file paths that should be parameterized.
- **Transaction Gaps**: Highlight where multiple database updates are executed without a surrounding transaction block.
- **Deprecated Steps**: Warn if any deprecated or obsolete FileMaker steps are used.
