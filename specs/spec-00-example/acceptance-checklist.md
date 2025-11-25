# Acceptance Checklist — spec-00-example

## Required Commands (source: `~/.helm/settings.json`)

- `pnpm all` — default regression suite from the global Helm settings; run after implementation to ensure regressions are caught.

## Manual Review

- [ ] The example spec files remain intact and informative for future contributors.
- [ ] Metadata status reflects the latest verifier run.
- [ ] `implementation-report.md` notes when the runner last touched this spec (placeholder until the first pass).
- [ ] Acceptance commands listed above still mirror the defaults from `~/.helm/settings.json`.

## Verifier Guidance

- Confirm every acceptance command defined in `~/.helm/settings.json` is represented with a clear purpose statement here.
- Ensure metadata transitions (`status`, `dependsOn`) stay accurate as work progresses.
- Check `implementation-report.md` for an updated timestamp or placeholder indicating whether the runner has executed yet.
- Flag any drift between this template and new specs so future authors inherit a consistent process.
