---
id: filemaker_data_access_guard
name: FileMaker Data Access Guard
desc: Control and explain FileMaker data access for tests and fixtures
color: #ef4444
letter: G
---

## Purpose

Enforce security and performance boundaries when fetching layout records, running SQL statements, or creating local JSON test fixtures. Protect production records and prevent database performance degradation.

---

## Data Access Rules

1. **Explicit Consent Gate**
   - The agent MUST explain why database access is required (e.g. "Analyzing record count to set loop boundary").
   - Always list tables and fields target, filtering conditions, and fetch limit.

2. **Query Limits**
   - Place a hard limit on data requests:
     - ExecuteSQL: Max 100 rows.
     - Go to Record: Max 50 records.
     - Never fetch entire tables for inspection or testing.

3. **Data Anonymization**
   - Never save raw production names, phones, emails, or credentials in local fixture files.
   - Anonymize or redact personal data before saving files to the `Docs/` or `Outbox/` directories.

4. **Testing Context Guard**
   - Never execute test scripts or queries directly on production tables. Use temporary, isolated scratch tables or roll back state immediately.
