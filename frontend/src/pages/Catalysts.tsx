import React, { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  Badge,
  Button,
  EmptyState,
  Icon,
  ItemTile,
  PageHead,
  ProgressBar,
} from "../components/kit";
import { useAuth } from "../contexts/AuthContext";
import { apiFetch } from "../lib/api";
import { errorState } from "../lib/errorState";
import { LoadingSpinner } from "../components/ui/LoadingSpinner";
import type { Catalyst, CraftPattern } from "../types/design";
import type { APIRecordsEnvelope } from "../types/api";

function CatalystCard({ c }: { c: Catalyst }) {
  const color =
    c.status === "complete"
      ? "var(--c-complete)"
      : c.status === "in-progress"
        ? "var(--c-progress)"
        : "var(--c-text-3)";
  return (
    <div className="gt-item gt-item--prog" data-rarity="exotic">
      <div className="gt-item-top">
        <ItemTile rarity="exotic" type={c.type || "Weapon"} icon={c.icon} />
        <div className="gt-item-head">
          <div className="gt-item-name">{c.name}</div>
          <div className="gt-item-type">
            {c.type ? `${c.type} Catalyst` : "Catalyst"}
          </div>
          <div className="gt-item-badges" style={{ marginTop: "var(--s-1)" }}>
            <Badge kind={c.status} dot />
          </div>
        </div>
      </div>
      {c.obj ? (
        <ProgressBar
          value={c.obj.cur}
          max={c.obj.max}
          label={c.obj.label}
          showVal
          color={color}
          size="tall"
        />
      ) : (
        <div className="gt-prog-empty mono">Not yet acquired</div>
      )}
      <div className="gt-item-src">
        <Icon name="bolt" size="0.8rem" style={{ color: "var(--c-text-4)" }} />
        {c.source}
      </div>
    </div>
  );
}

function CraftCard({ c }: { c: CraftPattern }) {
  const done = c.patterns.cur >= c.patterns.max;
  return (
    <div className="gt-item gt-item--prog" data-rarity="legendary">
      <div className="gt-item-top">
        <ItemTile rarity="legendary" type={c.type} icon={c.icon} />
        <div className="gt-item-head">
          <div className="gt-item-name">{c.name}</div>
          <div className="gt-item-type">{c.type}</div>
          <div className="gt-item-badges" style={{ marginTop: "var(--s-1)" }}>
            {done ? (
              <Badge kind="complete" dot>
                Craftable
              </Badge>
            ) : (
              <Badge kind="in-progress" dot>
                {c.note}
              </Badge>
            )}
          </div>
        </div>
      </div>
      <ProgressBar
        value={c.patterns.cur}
        max={c.patterns.max}
        label="Patterns extracted"
        showVal
        complete={done}
        color={done ? "var(--c-complete)" : "var(--c-exotic)"}
        size="tall"
      />
      <div className="gt-item-src">
        <Icon name="bolt" size="0.8rem" style={{ color: "var(--c-text-4)" }} />
        {c.source}
      </div>
    </div>
  );
}

type Tab = "catalysts" | "crafting";
type Filter = "all" | "missing" | "in-progress" | "complete";

export function Catalysts() {
  const { user } = useAuth();
  const membershipType = user?.membershipType;
  const membershipId = user?.membershipId;

  const {
    data: catalystsData,
    isLoading: catsLoading,
    isError: catsError,
    error: catsErr,
    refetch: refetchCats,
  } = useQuery({
    queryKey: ["catalysts", membershipType, membershipId],
    queryFn: () =>
      apiFetch<APIRecordsEnvelope<Catalyst>>(
        `/api/catalysts/${membershipType}/${membershipId}`,
      ),
    enabled: membershipType != null && !!membershipId,
  });
  const catalysts = catalystsData?.items ?? [];

  const {
    data: craftingData,
    isLoading: craftLoading,
    isError: craftError,
    error: craftErr,
    refetch: refetchCraft,
  } = useQuery({
    queryKey: ["crafting", membershipType, membershipId],
    queryFn: () =>
      apiFetch<APIRecordsEnvelope<CraftPattern>>(
        `/api/crafting/${membershipType}/${membershipId}`,
      ),
    enabled: membershipType != null && !!membershipId,
  });
  const crafting = craftingData?.items ?? [];

  const [tab, setTab] = useState<Tab>("catalysts");
  const [filter, setFilter] = useState<Filter>("all");

  if (catsLoading || craftLoading) {
    return (
      <div className="gt-page">
        <div className="gt-page-loading">
          <LoadingSpinner size="lg" />
        </div>
      </div>
    );
  }

  // Surface real failures (private profile, manifest warming up, Bungie down)
  // instead of rendering an empty grid that looks like "you have nothing".
  if (catsError || craftError) {
    const es = errorState(catsErr ?? craftErr);
    return (
      <div className="gt-page">
        <PageHead
          title="Catalysts & Crafting"
          sub="Long-grind progress, all in one place"
        />
        <div className="gt-card">
          <EmptyState
            icon={es.icon}
            color="var(--c-text-3)"
            title={es.title}
            body={es.body}
            action={
              <div style={{ display: "flex", gap: "var(--s-2)" }}>
                {es.privacyLink && (
                  <a
                    href="https://www.bungie.net/7/en/User/Account/Privacy"
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    <Button variant="outline" sm icon="external">
                      Bungie privacy settings
                    </Button>
                  </a>
                )}
                <Button
                  variant="outline"
                  sm
                  onClick={() => {
                    void refetchCats();
                    void refetchCraft();
                  }}
                >
                  Retry
                </Button>
              </div>
            }
          />
        </div>
      </div>
    );
  }

  const catFiltered =
    filter === "all" ? catalysts : catalysts.filter((c) => c.status === filter);
  const craftFiltered =
    filter === "all"
      ? crafting
      : crafting.filter((c) => {
          const done = c.patterns.cur >= c.patterns.max;
          return filter === "complete"
            ? done
            : filter === "missing"
              ? c.patterns.cur === 0
              : !done && c.patterns.cur > 0;
        });
  const catDone = catalysts.filter((c) => c.status === "complete").length;
  const craftable = crafting.filter(
    (c) => c.patterns.cur >= c.patterns.max,
  ).length;

  return (
    <div className="gt-page">
      <PageHead
        title="Catalysts & Crafting"
        sub="Long-grind progress, all in one place"
        right={
          <div className="gt-subtabs">
            <button
              className="gt-subtab"
              data-on={tab === "catalysts"}
              onClick={() => setTab("catalysts")}
            >
              Catalysts
            </button>
            <button
              className="gt-subtab"
              data-on={tab === "crafting"}
              onClick={() => setTab("crafting")}
            >
              Crafting Patterns
            </button>
          </div>
        }
      />

      <div className="gt-coll-toolbar">
        <div className="gt-filterbar">
          {(["all", "missing", "in-progress", "complete"] as Filter[]).map(
            (f) => (
              <button
                key={f}
                className="gt-fchip"
                data-on={filter === f}
                onClick={() => setFilter(f)}
              >
                {f === "all"
                  ? "All"
                  : f === "in-progress"
                    ? "In progress"
                    : f[0].toUpperCase() + f.slice(1)}
              </button>
            ),
          )}
        </div>
        <span className="gt-action-meta mono">
          {tab === "catalysts"
            ? `${catDone}/${catalysts.length} complete`
            : `${craftable}/${crafting.length} craftable`}
        </span>
      </div>

      {tab === "catalysts" ? (
        catFiltered.length ? (
          <div className="gt-itemgrid gt-itemgrid--prog">
            {catFiltered.map((c) => (
              <CatalystCard key={c.id} c={c} />
            ))}
          </div>
        ) : (
          <div className="gt-card">
            <EmptyState
              icon="filter"
              title="Nothing here"
              body="No catalysts match this filter."
              action={
                <Button variant="outline" sm onClick={() => setFilter("all")}>
                  Clear
                </Button>
              }
            />
          </div>
        )
      ) : craftFiltered.length ? (
        <div className="gt-itemgrid gt-itemgrid--prog">
          {craftFiltered.map((c) => (
            <CraftCard key={c.id} c={c} />
          ))}
        </div>
      ) : (
        <div className="gt-card">
          <EmptyState
            icon="filter"
            title="Nothing here"
            body="No patterns match this filter."
            action={
              <Button variant="outline" sm onClick={() => setFilter("all")}>
                Clear
              </Button>
            }
          />
        </div>
      )}
    </div>
  );
}
