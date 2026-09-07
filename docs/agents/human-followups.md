# Human follow-ups after agent work

Before reporting completed work, account for every required action that remains
with a human: account access, credentials, approvals, product/provider decisions,
manual acceptance, or historical-exposure assessment. Complete authorized work
that the agent can perform first. Routine CI repair, merging, and automated
validation stay engineering work unless a verified external constraint requires
a human.

Every required human action arising from completed agent work must have a
follow-up **issue in the private companion**, membership on the shared Guardian
Tracker project, and **published step-by-step instructions in the private wiki**.
This is an explicit exception to public-by-default issues and unclaimed drafts.
A final-response bullet, board draft, or wiki checklist alone is insufficient.

## Record the handoff

1. Identify each independently completable action, why it needs a human, and what
   it blocks. Inspect existing issues, board items, and the wiki before creating
   anything; reuse a matching open follow-up. Preserve completed records unless
   new evidence establishes new work. Group steps only when they share an owner
   and acceptance outcome.
2. Discover the authorized private companion through the existing independent
   `private/` checkout and its README, or established session configuration. Verify
   the repository identity and private visibility through GitHub. Keep its remote,
   issue links, wiki links, and operational identifiers out of public committed
   files and public issue/PR bodies. If access or identity cannot be verified,
   retain a value-free draft locally and report the publication blocker; never
   substitute a public issue or claim the handoff is complete.
3. Create or update the private follow-up issue. Include the originating work's
   issue/PR/release, the completed agent work, the remaining human action, its
   reason, prerequisites, responsible person or role, and observable completion
   evidence. Apply `ready-for-human` (create the canonical label if absent) and
   remove conflicting `ready-for-agent` labels. Do not assign a person without
   an established owner; state the required role when ownership is undecided.
4. Add the issue to the shared project using
   [the issue-tracker conventions](issue-tracker.md). Set its status accurately
   and its `Triage` field to `ready-for-human` where available. Record prerequisites
   and downstream blockers using native issue dependencies and the existing board
   fields. A follow-up can wait on a prerequisite and still require a human;
   unrelated engineering work must remain unblocked. Reuse/convert a matching
   draft rather than leaving duplicate active work items.
5. Publish a procedure in the private wiki, either a dedicated page or a stable
   section of the existing human-todo page. Link it from the wiki's human-todo
   index and back to the follow-up issue. Write numbered steps in execution order
   that a human can follow without this conversation. Include:
   - the purpose, required account/role/access, prerequisites, and starting state;
   - verified dashboard navigation or commands, with value-free placeholders and
     the intended execution environment;
   - expected results and checks, relevant failure/stop conditions, and recovery
     or rollback steps where the operation needs them;
   - the evidence to record, its safe destination, and the completion criteria.
     Verify load-bearing provider/UI facts before writing instructions. If a choice
     is still unmade, document how to make and record that decision; leave dependent
     execution blocked instead of inventing provider-specific steps. A wizard may
     assist the procedure but does not replace the issue or readable wiki steps.
6. Add the published wiki page/section link to the issue. Verify both links, board
   membership, labels, and that the remote wiki contains the final instructions.
   Issues and the project own live status; the wiki owns the procedure. Keep
   credentials, tokens, recovery codes, resolved secret references, and private
   keys out of both, using controlled evidence links and value-free attestations.
7. Report the private issue and wiki links to the authorized user, plus any
   publication/access gaps. Keep the human follow-up open until its acceptance
   evidence exists; closing the originating implementation issue does not close
   the human action. If no human action remains, say so without creating an empty
   issue or procedure.

The handoff is complete only when every required human action has a verified
private issue, project entry, and linked published procedure. Creating those
records is part of finishing the agent work; it does not authorize performing
the human's approval, credential operation, or production change.
