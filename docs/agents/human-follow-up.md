# Human follow-up

Required human actions left by completed agent work follow the procedure in the installed skills'
`human-follow-up` settings definition (for example
`.agents/skills/end-session/settings/human-follow-up.md`). They are an explicit exception to
public-by-default issues and unclaimed drafts (ADR 0022, ADR 0024): every required action gets a
private companion issue, a board entry and a published private-wiki procedure, even unclaimed. This
standing instruction authorizes creating and updating those issues, board items and wiki pages
without asking again; it does not authorize performing the human's approval, credential operation
or production change. Before publishing, verify through GitHub that the destination is the
companion and is private. Keep its remote, issue links, wiki links and operational identifiers out
of public files, issues and PR bodies.

## Follow-up destination

`discover: git -C private remote get-url origin` — issues in the private companion, selected
explicitly with `-R`, even for follow-ups from public work.

## Follow-up label

`ready-for-human` (create it if absent; remove a conflicting `ready-for-agent`)

## Follow-up board

same as Board — `Triage` `ready-for-human`, `Status` `Todo`; a prerequisite goes in `Blocked By`
and, between issues, a native `blocked_by` edge. A follow-up that waits on a prerequisite never
blocks unrelated engineering work.

## Follow-up wiki

The private companion's wiki: `discover: git -C private remote get-url origin`, with `.wiki.git`
in place of `.git` (default branch `master`). Index page `Human-todo`, linked from `Home`: each
procedure is a dedicated page or a stable numbered section of `Human-todo`, linked from its
`## Currently open` list and back to its issue. Push without asking: the standing authorization
above covers it.
