# Implementation Prompt — {{SPEC_ID}}: {{SPEC_NAME}}

You are an autonomous implementation agent working in this repository. Implement the spec below without changing unrelated parts of the codebase.

Your current task is to implement **one incremental spec** within a larger roadmap. You must obey the spec boundaries and acceptance criteria for:

- Spec ID: `{{SPEC_ID}}`
- Spec name: `{{SPEC_NAME}}`
- Mode: **{{MODE}}**

---

## Spec Text

Use the following spec as the **single source of truth** for what must be implemented:

```markdown
{{SPEC_BODY}}
```

If you see contradictions or ambiguities, call them out in your report and make the smallest reasonable assumption to move forward.

---

## Acceptance Commands ({{MODE}} Mode)

The following commands are required to pass in a clean run at the end of your work:

{{ACCEPTANCE_COMMANDS}}

- You must not consider the spec "done" unless all acceptance commands succeed.
- If commands fail for unrelated reasons, document them and leave the repo as clean as possible.

---

## Remaining Tasks from Previous Attempts

You may be in the middle of an iterative loop. The verifier may have already identified remaining tasks:

```json
{{PREVIOUS_REMAINING_TASKS}}
```

Treat these as **high-priority TODOs** that must be addressed before you can consider the spec done.

---

## Required Deliverables

In your final output, you MUST provide all of the following sections, in this order:

1. `SUMMARY` — 3–7 bullet points describing what you implemented and why.
2. `CHANGELOG` — bullet list of files you touched and the change made.
3. `TRACEABILITY` — map each acceptance criterion to evidence in the code.
4. `RUNBOOK` — steps to run the CLI feature plus the acceptance commands.
5. `MANUAL SMOKE TEST` — human-verifiable steps with expected outcomes.
6. `OPEN ISSUES & RISKS` — known gaps, follow-ups, or blocked work.

Refer back to the spec frequently and keep your output concise and actionable.