# Spec Breakdown Guide

Large specs should be divided into smaller, incremental `spec-XX-*` folders so that each attempt stays focused and verifiable. Use the following checklist when splitting work:

1. **Identify independent threads.** Break the product spec into vertical slices (CLI command, backend API, docs updates, etc.). Each slice should deliver end-user value or clear internal infrastructure.
2. **Define crisp acceptance criteria.** Every spec must include acceptance commands and a short checklist so the verifier can decide pass/fail without guesswork.
3. **List dependencies explicitly.** If a spec requires another one to finish first, add its ID to the `dependsOn` array in `metadata.json`.
4. **Keep scope tight.** Prefer many short specs over one monolithic document. Aim for work that can be implemented and verified in a single focused session.
5. **Document follow-ups.** If you intentionally defer work, capture it in the next spec’s `metadata.json` notes to maintain traceability.

### Suggested workflow

1. Paste the raw product spec into a scratch pad.
2. Highlight nouns (features, commands, toggles) and verbs (actions) to reveal natural slices.
3. Create a new `spec-XX-*` folder per slice, incrementing the numeric prefix.
4. For each folder:
   - Write `SPEC.md` (summary, goals, acceptance criteria).
   - Draft `acceptance-checklist.md` with concrete checks.
   - Initialize `metadata.json` with status "todo", dependencies, and acceptance commands.
   - Add a placeholder `implementation-report.md`.
When in doubt, split aggressively. Smaller specs reduce context for both humans and AI copilots, speeding up implementation and review.