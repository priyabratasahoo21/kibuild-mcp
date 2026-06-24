---
id: fm_xml_validator
name: FM XML Validator
desc: Validate FileMaker fmxmlsnippet output before preview/apply
color: #f97316
letter: V
---

## Purpose

Ensure generated FileMaker XML snippets conform strictly to structural rules before they reach the user. Prevents silent failures or workspace lockups during pasting.

---

## The 7 Structural Rules

1. **Root Wrapper Envelope**
   - The snippet MUST start and end with the `<fmxmlsnippet>` root tag:
     ```xml
     <?xml version="1.0" encoding="UTF-8"?>
     <fmxmlsnippet type="FMObjectList">
       <!-- Steps go here -->
     </fmxmlsnippet>
     ```
   - NEVER wrap step snippets in a `<Script>` tag unless explicitly exporting an entire script file wrapper context.

2. **No Namespace Pollution**
   - No `ns0:` prefix on elements. All tags must be raw (e.g. `<Step>`, `<State>`).
   - No `xmlns:ns0` attributes.

3. **No Dynamic Identifiers**
   - No `uuid` or `hash` attributes on step elements. FileMaker assigns these on import.
   - Do not generate `<uuid>` sub-elements.

4. **CDATA Wrap for Calculations**
   - All formulas, scripts, or queries inside `<Calculation>` or `<Text>` tags must be wrapped in `<![CDATA[...]]>` to prevent XML parsing issues.
     ```xml
     <Calculation><![CDATA[Get ( ScriptParameter ) ≠ ""]]></Calculation>
     ```

5. **Explicit Step Enablement**
   - Every `<Step>` tag must have `enable="True"` attribute.

6. **FileMaker Comment Step format**
   - Do not use XML comments (`<!-- comment -->`) for code comments. Use ID 89 Comment steps:
     ```xml
     <Step enable="True" id="89" name="Comment">
       <Text><![CDATA[# Your Comment Here]]></Text>
     </Step>
     ```

7. **Context Integrity Checks**
   - Validate that referenced field names, script names, or table names exist in the schema context before writing them to the XML.
