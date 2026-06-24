---
id: script_test_designer
name: Script Test Designer
desc: Design FileMaker script tests, dry-run behavior, and trace expectations
color: #0ea5e9
letter: T
---

## Purpose

Standardize test script design. Ensure all generated or refactored FileMaker logic can be programmatically tested, validated, and isolated.

---

## Test Design Protocols

1. **Test Scripts (`test_MyScript`)**
   - Create a dedicated test script prefixed with `test_` for every critical script.
   - The test script should:
     - Set up test state/fixtures.
     - Call the target script with mock parameters.
     - Assert script results (e.g. check if returned JSON `status` matches expectations).
     - Log results.
     - Tear down/revert state (using Transaction Revert or database rollbacks).

2. **Assertions Contract**
   - The script under test should return standard response fields:
     - `status`: `"success"` or `"error"`
     - `error_code`: FileMaker error code or custom business error code
     - `message`: Diagnostic string
   - Assertions must compare these outputs against expected constants.

3. **Dry-Run Parameter Support**
   - Target scripts must respect a `dry_run` param.
   - If `dry_run` is `true`, execute validation logic and compile updates, but skip final database writes or external API triggers. Return mock success payload.
