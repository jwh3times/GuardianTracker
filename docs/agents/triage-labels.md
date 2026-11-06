# Triage labels

The labels and what they mean are in `.agents/skills/ship/settings/triage-labels.md`. A skill's role name is the GitHub label to apply.

## Extra labels

- `blocked` — waits on something outside the issue and cannot progress until it clears. It sits
  beside the triage label rather than replacing it, and is removed when the blocker clears
  ([ADR 0026](../adr/0026-every-board-item-is-an-issue.md)).
