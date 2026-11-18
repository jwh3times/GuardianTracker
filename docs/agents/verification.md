# Verification

How to prove a premise, a test or a gate before acting on it or reporting it. Each rule here comes
from a result that was confidently wrong.

## Board cards are hypotheses

Never hand a card straight to an implementation agent, and never implement its prescription without
re-deriving it from the code. Cards name the right problem far more reliably than the right fix:
across the slices surveyed, about half had a materially false premise, and each would have shipped
a silent behavior change inside a slice that promised none.

1. **Survey first: count the real call sites yourself and read each one.** A card that says "five
   call sites" counted with a grep that answered a slightly different question; the divergent sites
   are the whole story.
2. **Take a card's "closing as working-as-intended is a valid outcome" literally.** It is there
   because the filer was unsure.
3. **When a card says to check the rest of the file for the same drift, do it.**
4. **Report a false premise to the maintainer** rather than silently reinterpreting the card: it
   often changes what the right work is.
5. **When a card offers a "smallest" option, work out each part of its condition** and ask what
   breaks if that part were the only one. A right diagnosis can carry a half-right mechanism.
6. Then implement, then mutation-check (below).

The same suspicion applies to:

- **a load-bearing code comment**: one claiming two consumers "gate on different identity facts"
  was the only reason a duplication survived;
- **a safety claim in a comment** ("fails closed"), which may only be tolerated;
- **an ADR you wrote yourself**: survey before implementing even an accepted one, and record a
  deviation as an implementation note;
- **an external fact on a card** (a Bungie response shape, a Manifest relationship): the survey step
  is a capture or a citation, not a code read (ADR 0003);
- **a card's failure branch**: diagnose a failure before taking its prescribed fallback;
- **"observed" evidence from calling a helper directly**, which isn't a UI repro. Check that a red
  test fails on the claimed symptom before trusting it.

## Break the code to prove the test

After fixing a flaky test or writing one that guards a race or an invariant, break the code under
test and confirm it goes red. "It stopped failing" is not evidence.

- **Break the invariant at its source** and tag the scaffolding so cleanup is one grep. Undo it with
  an inverse edit, never `git checkout --` or `git restore` on a file with uncommitted work.
- **Verify the break is caught under every configuration the bug appeared in.** A fix for a
  `GOMAXPROCS=1` flake was caught at 8 and missed at 1.
- **If a test infers a verdict from end state, ask what else could write that state afterwards.**
- **Design at least one mutant that gives wrong but believable output** (a plausible argument of
  the same type, a dropped edge case), not only line deletions, which die to any test that runs
  the line.
- **Design one mutant that keeps the call you assert on and changes its effect.** The recurring
  weak assertion is an intermediate signal standing in for an observable outcome: a React Query
  key's head, an invalidation that a later refetch hides, a set of one sample. Assert what a user
  would notice: a second request, a value rendered while the mutation is in flight. When a weak
  assertion turns up, check whether the same template was already copied into shipped code.
- **A surviving mutant can indict the code, not the test.** Ask "is this code doing anything?" first:
  one survivor was a dead intersection copied from a module where it was load-bearing.
- **A conjunct that survives may be implied by its siblings.** Prove that before writing a test for
  it; if so, delete it and document the invariant. A guard that matters only inside an
  unschedulable window is a known survivor, not a reason for a sleep-based test.
- **Re-run the sweep after any fix.** A stale `-run` pattern reports a false result.

## A gate that ran nothing looks clean

- **Plant a finding to prove a gate ran.** Staticcheck on the wrong toolchain fails while loading
  and prints nothing that names project code (`AGENTS.md`, "Staticcheck needs the pinned
  toolchain"). Append an unused function, confirm it is reported, then restore the exact bytes.
- **Check a skip is real.** The SQLite-backed manifest tests do run on a machine with a C toolchain
  and `CGO_ENABLED=1`; run one with `-v` and look for `PASS` against `SKIP` rather than trusting
  either claim.
- **A probe can be blind by default.** Playwright's default `chromium.launch()` is the headless
  shell, whose `Page.getInstallabilityErrors` returns an empty list even with no manifest. Use
  `chromium.launchPersistentContext(<dir>, { channel: "chromium" })` (a non-persistent context
  always reports `in-incognito`), and mutation-check the probe.

## Claims about a flake

- **Run a scheduling flake with `GOMAXPROCS=1` first.** It turned a one-in-a-full-suite failure into
  30 of 30. A failure rate that drops as `GOMAXPROCS` rises is an ordering assumption, not a data
  race: if `-race` is silent, look for "who runs first".
- **Before blaming a library, read the exact assertion and the test's duration.** An `ECONNREFUSED`
  on 8081 after a Vitest failure was an in-flight request hitting the real port only because
  teardown had closed MSW, not a socket leak. Interceptor debug logs
  (`DEBUG='interceptors:*' DEBUG_LEVEL=verbose`) showing `disable` before the request are the tell.
  Mutation-check the test with any workaround in place: a test that survives a broken implementation
  was masked, not protected.
- **Before designing a fix, grep for a helper that already exists for the rule.**
