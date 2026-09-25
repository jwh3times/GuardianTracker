import { useMemo, useState } from "react";
import { Link, useNavigate } from "react-router";
import { Icon } from "../../components/Icon";
import { Button } from "../../components/primitives";
import { RoleBadge } from "../admin/AdminKit";
import { useFlags } from "../../contexts/FlagsContext";
import { useRollTargetMatches, useRollTargets } from "../../data/rolltargets";
import { ApiError } from "../../lib/api";
import { errorState } from "../../lib/errorState";
import type { GTItem, PerkColumn, RollTarget } from "../../types/design";
import { isTargetPerk } from "../rolltargets/rollTargetsView";
import { NearMissSummary } from "../rolltargets/NearMissSummary";
import {
  groupMatchesByCopy,
  matchesForItem,
  weaponBoundTargetsFor,
  type CopyMatchGroup,
} from "./rollTargetSectionView";

/**
 * The per-weapon roll-target block in the item detail drawer (roll targets
 * slice 6, #371), gated by the `god-roll` flag. Reads `data/rolltargets.ts`'s
 * existing list and match-report queries — no new query identity, no
 * `types/api` import — and shares one cache entry with the Roll targets page.
 *
 * Management (notes, delete, DIM import) stays on `/rolls`; this section is
 * read-only. It always surfaces a "Manage roll targets" link there.
 */
export function RollTargetSection({
  item,
  perkColumns,
  perksLoading,
}: {
  item: GTItem;
  perkColumns?: PerkColumn[];
  perksLoading?: boolean;
}) {
  const navigate = useNavigate();
  const { flagState, isLoading: flagsLoading } = useFlags();
  const { enabled, accessible, locked } = flagState("god-roll");

  // Roll targets are weapon-specific. Until the drawer's own perks query
  // resolves, we don't yet know whether this item is a weapon — wait rather
  // than guess, so a still-loading armor piece doesn't briefly show as one.
  const isWeapon = !perksLoading && !!perkColumns && perkColumns.length > 0;

  // Never issue the matches query (a live Bungie profile read) unless the
  // section can actually show something: the flags query has resolved (an
  // unresolved flag fails open elsewhere in the app — e.g. `FlaggedRoute` —
  // but that tradeoff is wrong for a query that costs a live Bungie call),
  // the flag is accessible, and the drawer is open on a resolved weapon. The
  // list query is cheap (no Bungie call) but gated the same way — nothing
  // reads it unless this section is going to render.
  const fetchEnabled = !flagsLoading && accessible && isWeapon;

  const { targets } = useRollTargets({ enabled: fetchEnabled });
  const {
    matches,
    isLoading: matchesLoading,
    isError: matchesError,
    error: matchesErrorObj,
    retry: retryMatches,
  } = useRollTargetMatches({ enabled: fetchEnabled });

  const [unwantedOpen, setUnwantedOpen] = useState(false);

  const targetsById = useMemo(
    () => new Map<string, RollTarget>(targets.map((t) => [t.id, t])),
    [targets],
  );

  if (!enabled) return null;

  if (locked) {
    return (
      <section className="gt-drawer-block" aria-labelledby="rt-drawer-title">
        <h3 id="rt-drawer-title" className="gt-section-title">
          Your roll targets
        </h3>
        <div className="gt-godroll-locked">
          <Icon
            name="lock"
            size="1.1rem"
            style={{ color: "var(--c-exotic)" }}
          />
          <div>
            <div style={{ color: "var(--c-text-2)", fontSize: "var(--t-sm)" }}>
              Roll targets is an{" "}
              <strong style={{ color: "var(--c-exotic)" }}>Alpha</strong>{" "}
              feature.
            </div>
            <div className="gt-item-type">
              See which of your copies match the rolls you&rsquo;re chasing.
            </div>
          </div>
        </div>
      </section>
    );
  }

  if (!isWeapon) return null;

  const matchesReady = !matchesLoading && !matchesError && !!matches;

  return (
    <section className="gt-drawer-block" aria-labelledby="rt-drawer-title">
      <h3 id="rt-drawer-title" className="gt-section-title">
        Your roll targets <RoleBadge role="alpha" />
      </h3>

      {matchesError ? (
        <RollTargetSectionFailure
          item={item}
          targets={targets}
          error={matchesErrorObj}
          onRetry={retryMatches}
          onReconnect={() => navigate("/reauthorize")}
        />
      ) : !matchesReady ? (
        <div className="gt-perks-loading">Checking your copies…</div>
      ) : (
        <RollTargetSectionReady
          item={item}
          matches={matches}
          targetsById={targetsById}
          unwantedOpen={unwantedOpen}
          onToggleUnwanted={() => setUnwantedOpen((v) => !v)}
        />
      )}

      <Link to="/rolls" className="gt-link">
        Manage roll targets
      </Link>
    </section>
  );
}

function RollTargetSectionFailure({
  item,
  targets,
  error,
  onRetry,
  onReconnect,
}: {
  item: GTItem;
  targets: RollTarget[];
  error: unknown;
  onRetry: () => void;
  onReconnect: () => void;
}) {
  const isReauth =
    error instanceof ApiError && error.code === "BUNGIE_REAUTH_REQUIRED";
  const copy = errorState(error);
  const neutral = weaponBoundTargetsFor(targets, item.id);
  return (
    <>
      {neutral.length > 0 && (
        <ul className="gt-rt-list">
          {neutral.map((t) => (
            <li key={t.id} className="gt-rt-row">
              <div className="gt-rt-row-body">
                <div className="gt-rt-row-top">
                  <span className="gt-chip gt-rt-chip-neutral">
                    Match status unknown
                  </span>
                </div>
                {t.perks.length > 0 && (
                  <div className="gt-rt-perks">
                    {t.perks.map((p) => (
                      <span key={p} className="gt-chip">
                        {p}
                      </span>
                    ))}
                  </div>
                )}
                {t.notes && (
                  <div className="gt-wl-notes">&quot;{t.notes}&quot;</div>
                )}
              </div>
            </li>
          ))}
        </ul>
      )}
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: "var(--s-2)",
          flexWrap: "wrap",
        }}
      >
        <span className="gt-action-meta">
          {isReauth
            ? "Reconnect Bungie to see whether anything you own matches."
            : copy.body}
        </span>
        <Button variant="outline" sm onClick={isReauth ? onReconnect : onRetry}>
          {isReauth ? "Reconnect" : "Retry"}
        </Button>
      </div>
    </>
  );
}

function RollTargetSectionReady({
  item,
  matches,
  targetsById,
  unwantedOpen,
  onToggleUnwanted,
}: {
  item: GTItem;
  matches: NonNullable<ReturnType<typeof useRollTargetMatches>["matches"]>;
  targetsById: Map<string, RollTarget>;
  unwantedOpen: boolean;
  onToggleUnwanted: () => void;
}) {
  const chasing = weaponBoundTargetsFor(matches.unmatchedTargets, item.id);
  const wanted = matchesForItem(matches.wanted, item.id);
  const unwanted = matchesForItem(matches.unwanted, item.id);
  const wantedGroups = groupMatchesByCopy(wanted, targetsById);
  const unwantedGroups = groupMatchesByCopy(unwanted, targetsById);

  if (
    chasing.length === 0 &&
    wantedGroups.length === 0 &&
    unwantedGroups.length === 0
  ) {
    return <p className="gt-action-meta">No roll targets for this weapon.</p>;
  }

  return (
    <>
      {wantedGroups.length > 0 && (
        <div className="gt-rt-drawer-group">
          <div className="gt-perk-col-label">Copies you own that match</div>
          <ul className="gt-rt-copies">
            {wantedGroups.map((g) => (
              <CopyRow key={g.instanceId} group={g} />
            ))}
          </ul>
        </div>
      )}

      {chasing.length > 0 && (
        <div className="gt-rt-drawer-group">
          <div className="gt-perk-col-label">Still chasing</div>
          <ul className="gt-rt-list">
            {chasing.map((t) => (
              <li key={t.id} className="gt-rt-row">
                <div className="gt-rt-row-body">
                  <div className="gt-rt-perks">
                    {t.perks.map((p) => (
                      <span key={p} className="gt-chip">
                        {p}
                      </span>
                    ))}
                  </div>
                  <NearMissSummary bestCopy={t.bestCopy} />
                  {t.notes && (
                    <div className="gt-wl-notes">&quot;{t.notes}&quot;</div>
                  )}
                </div>
              </li>
            ))}
          </ul>
        </div>
      )}

      {unwantedGroups.length > 0 && (
        <div className="gt-rt-disclosure">
          <button
            className="gt-rt-disclosure-head"
            onClick={onToggleUnwanted}
            aria-expanded={unwantedOpen}
          >
            <h4 className="gt-perk-col-label" style={{ marginBottom: 0 }}>
              Matches a roll you marked unwanted ({unwantedGroups.length})
            </h4>
            <Icon
              name="chevronDown"
              size="1rem"
              style={{
                color: "var(--c-text-3)",
                transform: unwantedOpen ? "rotate(180deg)" : "none",
                transition: "transform var(--dur)",
              }}
            />
          </button>
          {unwantedOpen && (
            <ul className="gt-rt-copies">
              {unwantedGroups.map((g) => (
                <CopyRow key={g.instanceId} group={g} />
              ))}
            </ul>
          )}
        </div>
      )}
    </>
  );
}

function CopyRow({ group }: { group: CopyMatchGroup }) {
  return (
    <li className="gt-rt-copy">
      <span className="gt-rt-copy-label mono">{group.label}</span>
      <div className="gt-rt-perks">
        {group.perks.map((p) => (
          <span
            key={p}
            className="gt-chip"
            data-highlight={isTargetPerk(p, group.targetPerks)}
          >
            {p}
          </span>
        ))}
      </div>
      <div className="gt-rt-perks">
        {group.satisfiedTargets.map((t) => (
          <span key={t.targetId} className="gt-perk">
            {t.label}
          </span>
        ))}
      </div>
    </li>
  );
}
