# KiBuild Default Workflow Templates

These files define repeatable developer workflows for the KiBuild agent.
They are runtime templates owned by the sidecar, not planning docs.

Workflows are different from skills:

- **Workflow**: the procedure for a job, such as create script, refactor script, or audit schema.
- **Skill**: specialist knowledge or style guidance that can support a workflow, such as FileMaker XML serialization or dependency analysis.

Runtime injection order should be:

```text
1. sidecar/workflows/default_workflows/base_agent_rules.md
2. active_context.json
3. selected module/database summaries
4. selected workflow from this folder
5. enabled specialist skills
6. exact artifacts retrieved through tools
```

Recommended first workflow to implement in the sidecar:

```text
create_script.md
```

Available workflow templates:

```text
implementation_plan.md
solution_blueprint.md
verify_build.md
create_script.md
refactor_script.md
analyze_script.md
test_script.md
create_layout.md
schema_audit.md
document_solution.md
```
