---
id: fm_xml_serializer
name: FM JSON Serializer
desc: Encode FileMaker scripts as deterministic JSON payloads to be compiled into valid fmxmlsnippet XML
color: #f59e0b
letter: X
---

## Purpose

Enforce strict serialization rules to generate deterministic JSON that the KiBuild backend compiler will transpile into perfectly formatted FileMaker clipboard-compatible `<fmxmlsnippet>` XML.

---

## JSON Payload Structure

Instead of generating raw XML wrapped in ` ```xml ` blocks, you must ALWAYS output a JSON array of step objects, wrapped in a ` ```json ` block. 
The KiBuild Go backend will dynamically inject this JSON into safe XML templates.

### Format

```json
[
  { "stepName": "If", "parameters": { "Calculation": "Get ( LastError ) ≠ 0" } },
  { "stepName": "Set Field", "parameters": { "TargetTable": "Invoices", "TargetField": "Status", "Calculation": "\"Paid\"" } },
  { "stepName": "Perform Script", "parameters": { "ScriptName": "Log Event", "Calculation": "\"Updated Status\"" } }
]
```

### Supported Core Parameters
When specifying `parameters`, strictly use the expected template keys:
- `Calculation`: The primary FileMaker calculation string. Do NOT wrap in `<Calculation>` tags.
- `Name`: Used for variable names (e.g. `$var` in `Set Variable`).
- `TargetTable` / `TargetField`: Used for `Set Field` targeting.
- `LayoutName`: Used in `Go to Layout`.
- `ScriptName`: Used in `Perform Script`.
- `NoInteract`: Boolean for dialog suppression.
- `State`: Boolean for toggles (e.g., `Set Error Capture`).
- `Title`: Used in `Show Custom Dialog`.

---

## The 80/20 Fallback Strategy (Raw XML)

If you need a highly complex or obscure FileMaker step (e.g., `Insert From URL` with cURL options, `Execute SQL`) that is outside the Core template engine, use the `raw_xml` fallback.

**Rules for `raw_xml`:**
- Must be a single valid `<Step>` element with `enable="True"` and the correct numeric `id`.
- Do NOT wrap in `<fmxmlsnippet>` — the compiler adds the wrapper.
- Do NOT include XML comments (`<!-- -->`). Use a `# (comment)` step instead.
- CDATA sections inside `raw_xml` must not contain the sequence `]]>`.

```json
[
  { "stepName": "If", "parameters": { "Calculation": "$$Flag" } },
  { 
    "raw_xml": "<Step enable=\"True\" id=\"160\" name=\"Insert from URL\"><URL><Calculation><![CDATA[\"https://api.example.com\"]]></Calculation></URL></Step>" 
  }
]
```

---

## Cross-File Perform Script (REQUIRED — prevents dropped references)

The `Perform Script` JSON template only encodes a local `ScriptName`. It CANNOT
encode a call into another file — the file linkage (`DataSourceReference`) is lost,
which silently turns a cross-file call into a self-call (often an infinite loop).

**Rule:** If a `Perform Script` step targets a script in a *different* file (the
original step shows `File: "X"`, or the source `<Step>` contains a
`<DataSourceReference>`), do NOT use the structured `ScriptName` form. Instead use
`raw_xml` and copy the original `<Step>` element verbatim from the source
`.fmxmlsnippet`, preserving its `<Script>` element AND its `<DataSourceReference>`
(including the numeric `id`). Only strip disallowed attributes (`uuid`/`hash`).

```json
[
  {
    "raw_xml": "<Step enable=\"True\" id=\"1\" name=\"Perform Script\"><DataSourceReference name=\"MainPOS\" id=\"266\"/><Script name=\"Login Screen\" id=\"1133\"/></Step>"
  }
]
```

For a same-file `Perform Script`, keep using the structured `ScriptName` form.

---

## Send Mail (SMTP / HTML email)

`Send Mail` has a template — use the structured form, NOT `raw_xml`. The compiler emits
the correct FileMaker step (id 63) and the real clipboard elements. Supported parameter
keys (all optional; provide what you need — each value is a FileMaker calculation):

- `To`, `Cc`, `Bcc` — recipient address calcs
- `Subject`, `Message` — subject and body calcs (put the HTML body string in `Message`;
  there is NO separate HTMLMessage element — that is not a real FileMaker element)
- `FromEmail`, `FromName`, `ReplyTo` — sender identity calcs
- `SMTPServer`, `SMTPPort`, `SMTPUsername`, `SMTPPassword` — SMTP connection calcs
- `Attachment` — file path list (UniversalPathList)
- `Encryption` — one of `SMTPEncryptionNone` | `SMTPEncryptionTLS` | `SMTPEncryptionSSL` (default TLS)
- `WithDialog` — boolean; omit/false = "No dialog" (send silently). `MultipleEmails` — one per found-set record.

The template hard-codes SMTP send mode (`SendViaSMTP=True`). For OAuth (Google/Microsoft)
or other rare variants, use `raw_xml` with the real elements (`SendViaOAuthAuthentication`,
`OAuth*`), keeping `id="63"`.

```json
[
  { "stepName": "Send Mail", "parameters": {
      "To": "$toEmail", "Subject": "\"Welcome\"", "Message": "$htmlBody",
      "FromEmail": "$$SMTP_FROM", "SMTPServer": "$$SMTP_HOST", "SMTPPort": "587",
      "SMTPUsername": "$$SMTP_USER", "SMTPPassword": "$$SMTP_PASS" } }
]
```

---

## Calculation String Rules

- Do NOT wrap calculation strings in `<Calculation>` tags — the template does this.
- Do NOT include `]]>` inside a calculation value. If a string literal must contain `>` after `]]`, rewrite using a variable or `Char(93)`.
- Step names are case-insensitive (`"commit records"` resolves to `"Commit Records/Requests"`), but use exact canonical names when possible.

---

## Output Rules
1. Call `write_outbox_artifact` with `files` containing ONLY `<ScriptName>.json` (the JSON array). Do NOT provide `<ScriptName>.txt` or `<ScriptName>.xml` — the system compiles `.json` → `.xml` and generates `.txt` automatically from the compiled XML.
2. **NEVER generate empty control steps.** `If`, `Else If`, `Exit Loop If`, and `Set Field` must ALWAYS contain their `Calculation` parameter.
3. If compilation fails, the tool returns the exact error. Fix the reported step name or parameter and retry — do not fall back to raw XML unless the step genuinely has no template.
