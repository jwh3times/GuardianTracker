import React, { useEffect, useMemo, useReducer } from "react";
import { useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { CountdownChip } from "../../components/primitives";
import {
  ActionList,
  MilestoneModule,
  PageHead,
  Panel,
  XurModule,
} from "../../components/composite";
import { LoadingSpinner } from "../../components/LoadingSpinner";
import { useAuth } from "../../contexts/AuthContext";
import { useCharacters } from "../../contexts/CharacterContext";
import { apiFetch } from "../../lib/api";
import type { Weekly } from "../../types/design";

const DONE_KEY_PREFIX = "gt_done:";

// Checkmarks persist in localStorage keyed by the weekly reset timestamp, so
// they survive refetches and reloads but clear naturally at the weekly reset.
function loadDoneIds(resetAt: string | undefined): Set<string> {
  if (!resetAt) return new Set();
  try {
    const raw = localStorage.getItem(DONE_KEY_PREFIX + resetAt);
    return raw ? new Set(JSON.parse(raw) as string[]) : new Set();
  } catch {
    return new Set();
  }
}

function saveDoneIds(resetAt: string, ids: Set<string>) {
  localStorage.setItem(DONE_KEY_PREFIX + resetAt, JSON.stringify([...ids]));
}

function purgeOldWeeks(currentResetAt: string) {
  for (let i = localStorage.length - 1; i >= 0; i--) {
    const key = localStorage.key(i);
    if (
      key?.startsWith(DONE_KEY_PREFIX) &&
      key !== DONE_KEY_PREFIX + currentResetAt
    ) {
      localStorage.removeItem(key);
    }
  }
}

export function ThisWeek() {
  const { user } = useAuth();
  const { activeCharacter, isLoading: charactersLoading } = useCharacters();
  const navigate = useNavigate();

  const characterID = activeCharacter?.id;

  const { data: w, isLoading } = useQuery({
    queryKey: ["weekly", characterID ?? null],
    queryFn: () =>
      apiFetch<Weekly>(
        `/api/weekly/recommendations${characterID ? `?characterId=${encodeURIComponent(characterID)}` : ""}`,
      ),
    enabled: !!user && !charactersLoading,
  });

  const resetAt = w?.resetAt;

  // localStorage is the source of truth; `version` forces a re-read after a
  // toggle. A new reset week derives a fresh (empty) checklist automatically.
  const [version, bump] = useReducer((x: number) => x + 1, 0);
  const doneIds = useMemo(
    () => loadDoneIds(resetAt),
    // eslint-disable-next-line react-hooks/exhaustive-deps -- version invalidates the localStorage read
    [resetAt, version],
  );

  useEffect(() => {
    if (resetAt) purgeOldWeeks(resetAt);
  }, [resetAt]);

  const acts = (w?.recommended ?? []).map((x) => ({
    ...x,
    done: doneIds.has(x.id),
  }));

  const toggle = (id: string) => {
    if (!resetAt) return;
    const n = new Set(doneIds);
    if (n.has(id)) {
      n.delete(id);
    } else {
      n.add(id);
    }
    saveDoneIds(resetAt, n);
    bump();
  };

  const doneCount = acts.filter((a) => a.done).length;

  if (isLoading) {
    return (
      <div
        className="gt-page"
        style={{
          display: "flex",
          justifyContent: "center",
          padding: "var(--s-8)",
        }}
      >
        <LoadingSpinner size="lg" />
      </div>
    );
  }

  return (
    <div className="gt-page" data-onboarding-target="this-week">
      <PageHead
        title="This Week"
        sub={w ? `Resets ${w.resetLabel}` : "Weekly activities"}
        right={
          <CountdownChip
            prefix="Resets in"
            time={w?.resetIn ?? { d: 0, h: 0, m: 0 }}
            icon="clock"
          />
        }
      />

      {w?.degraded && (
        <div className="gt-card" style={{ padding: "var(--s-3)" }}>
          <span className="mono" style={{ color: "var(--c-text-3)" }}>
            Item names are still loading on the server — some labels may appear
            as placeholders and will fill in shortly.
          </span>
        </div>
      )}

      <Panel
        title="Recommended for you"
        icon="sparkle"
        accent="var(--c-signal)"
        right={
          <span className="gt-action-meta mono">
            {doneCount}/{acts.length} done
          </span>
        }
      >
        <ActionList items={acts} onToggle={toggle} />
      </Panel>

      <div className="gt-week-cols">
        <XurModule
          xur={w?.xur ?? null}
          activeClassName={activeCharacter?.cls}
          onSelect={(hash) => navigate("/collections?item=" + hash)}
        />
        <MilestoneModule milestones={w?.milestones ?? []} />
      </div>
    </div>
  );
}
