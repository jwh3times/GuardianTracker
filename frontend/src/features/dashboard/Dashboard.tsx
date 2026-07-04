import React from "react";
import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import {
  Badge,
  Button,
  CountdownChip,
  DataFreshnessChip,
  EmptyState,
  ItemTile,
  ProgressBar,
  RadialProgress,
  Skeleton,
} from "../../components/primitives";
import { PageHead, Panel } from "../../components/composite";
import { Icon } from "../../components/Icon";
import { useAuth } from "../../contexts/AuthContext";
import { useCharacters } from "../../contexts/CharacterContext";
import { apiFetch } from "../../lib/api";
import { errorState } from "../../lib/errorState";
import { collectionsQuery } from "../../lib/queries";
import { toWishlistEntry } from "../../lib/adapters";
import { isFirstRun, markFirstRunDone } from "../../lib/firstRun";
import { GetStartedPanel } from "./GetStartedPanel";
import type { DailyAction, SummaryCategory, Weekly } from "../../types/design";
import type {
  ProfileResponse,
  APICategoryCount,
  WishListItem,
} from "../../types/api";

function formatDuration(
  d: import("../../types/design").Duration | undefined,
): string {
  if (!d) return "";
  if (d.d && d.d > 0) return `${d.d}d ${d.h ?? 0}h`;
  if (d.h && d.h > 0) return `${d.h}h ${d.m ?? 0}m`;
  return `${d.m ?? 0}m`;
}

const emblemStyle = (url?: string): React.CSSProperties | undefined =>
  url
    ? {
        backgroundImage: `url(${url})`,
        backgroundSize: "cover",
        backgroundPosition: "center",
      }
    : undefined;

export function Dashboard() {
  const { user } = useAuth();
  const { activeCharacter } = useCharacters();
  const navigate = useNavigate();
  const go = (path: string) => navigate(path);

  const [showGetStarted, setShowGetStarted] = React.useState(() =>
    isFirstRun(user?.membershipId),
  );
  const dismissGetStarted = () => {
    markFirstRunDone(user?.membershipId);
    setShowGetStarted(false);
  };

  const { data: profileData } = useQuery({
    queryKey: ["currentUser"],
    queryFn: () => apiFetch<ProfileResponse>("/api/auth/profile"),
  });

  const membershipType =
    profileData?.user.membershipType ?? user?.membershipType;
  const membershipId = profileData?.user.membershipId ?? user?.membershipId;

  // Shares the "missing" collections cache entry with Settings and the
  // Collections page (one fetch across all three) via the shared query helper.
  const {
    data: real,
    isLoading: collectionsLoading,
    isError: collectionsFailed,
    error: collectionsError,
    refetch: refetchCollections,
  } = useQuery(collectionsQuery(membershipType, membershipId, false));

  const {
    data: weeklyData,
    isLoading: weeklyLoading,
    isError: weeklyFailed,
  } = useQuery({
    queryKey: ["weekly"],
    queryFn: () => apiFetch<Weekly>("/api/weekly/recommendations"),
    enabled: membershipType != null && !!membershipId,
  });

  const {
    data: wishlistItems,
    isLoading: wishlistLoading,
    isError: wishlistFailed,
  } = useQuery({
    queryKey: ["wishlist"],
    queryFn: () => apiFetch<WishListItem[]>("/api/wishlist"),
    enabled: !!user,
  });

  const displayName =
    profileData?.user.displayName || user?.displayName || "Guardian";

  // Normalize wishlist rows through the adapter so rarity/availability handling
  // stays in one place (lib/adapters) rather than reading raw API fields here.
  const wishlistEntries = (wishlistItems ?? []).map(toWishlistEntry);
  const availableNowCount = wishlistEntries.filter((e) => e.avail.now).length;

  const fromReal = (
    c: APICategoryCount | undefined,
    fallback: SummaryCategory,
  ): SummaryCategory => {
    if (c && c.total > 0) {
      return {
        id: fallback.id,
        label: fallback.label,
        pct: Math.round((c.collected / c.total) * 100),
        count: [c.collected, c.total],
      };
    }
    return fallback;
  };

  const mWeapons: SummaryCategory = {
    id: "weapons",
    label: "Weapons",
    pct: 0,
    count: [0, 0],
  };
  const mArmor: SummaryCategory = {
    id: "armor",
    label: "Armor",
    pct: 0,
    count: [0, 0],
  };
  const mExotics: SummaryCategory = {
    id: "exotics",
    label: "Exotics",
    pct: 0,
    count: [0, 0],
  };
  const mCosmetics: SummaryCategory = {
    id: "cosmetics",
    label: "Cosmetics",
    pct: 0,
    count: [0, 0],
  };
  const categories: SummaryCategory[] = [
    fromReal(real?.summary.weapons, mWeapons),
    fromReal(real?.summary.armor, mArmor),
    fromReal(real?.summary.exotics, mExotics),
    fromReal(real?.summary.cosmetics, mCosmetics),
  ];
  const overall = Math.round(
    categories.reduce((sum, c) => sum + c.pct, 0) / categories.length,
  );

  return (
    <div className="gt-page gt-dash">
      <PageHead
        title={`${showGetStarted ? "Welcome" : "Welcome back"}, ${displayName}`}
        sub="Your collection at a glance"
        right={
          <CountdownChip
            prefix="Weekly reset"
            time={weeklyData?.resetIn ?? { d: 0, h: 0, m: 0 }}
            icon="clock"
          />
        }
      />

      {showGetStarted && <GetStartedPanel onDismiss={dismissGetStarted} />}

      {/* HERO COMPLETION */}
      <Panel pad={false} style={{ padding: "var(--s-5)" }}>
        {collectionsFailed ? (
          (() => {
            const copy = errorState(collectionsError);
            return (
              <EmptyState
                icon={copy.icon}
                title={copy.title}
                body={copy.body}
                action={
                  <Button onClick={() => void refetchCollections()}>
                    Retry
                  </Button>
                }
              />
            );
          })()
        ) : (
          <div className="gt-hero">
            {activeCharacter && (
              <div className="gt-hero-char">
                <span
                  className="gt-avatar"
                  data-cls={activeCharacter.cls}
                  style={emblemStyle(activeCharacter.emblemUrl)}
                >
                  {activeCharacter.emblemUrl ? "" : activeCharacter.name[0]}
                </span>
                <div className="gt-hero-char-main">
                  <span className="gt-hero-char-name">
                    {activeCharacter.cls}
                  </span>
                  <span className="gt-hero-char-sub mono">
                    {activeCharacter.race} ·{" "}
                    <Icon
                      name="bolt"
                      size="0.7rem"
                      style={{ color: "var(--c-exotic)" }}
                    />{" "}
                    {activeCharacter.power}
                  </span>
                </div>
              </div>
            )}
            <div className="gt-hero-radial">
              {collectionsLoading ? (
                <Skeleton
                  w="clamp(8rem,9vw,10rem)"
                  h="clamp(8rem,9vw,10rem)"
                  r="50%"
                />
              ) : (
                <RadialProgress
                  value={overall}
                  size="clamp(8rem,9vw,10rem)"
                  color="var(--c-exotic)"
                  sub="overall"
                  pctSize="clamp(2rem,3vw,2.6rem)"
                />
              )}
            </div>
            <div className="gt-hero-bars">
              {collectionsLoading
                ? ["weapons", "armor", "exotics", "cosmetics"].map((id) => (
                    <div
                      key={id}
                      className="gt-hero-bar"
                      style={{ pointerEvents: "none" }}
                    >
                      <div className="gt-hero-bar-top">
                        <Skeleton w="5rem" h="0.75rem" />
                        <Skeleton w="3.5rem" h="0.75rem" />
                      </div>
                      <Skeleton
                        w="100%"
                        h="0.5rem"
                        r="var(--r-pill)"
                        style={{ marginTop: "var(--s-2)" }}
                      />
                    </div>
                  ))
                : categories.map((c) => (
                    <button
                      key={c.id}
                      className="gt-hero-bar"
                      onClick={() =>
                        go(c.id === "cosmetics" ? "/cosmetics" : "/collections")
                      }
                    >
                      <div className="gt-hero-bar-top">
                        <span className="gt-hero-bar-label">{c.label}</span>
                        <span className="mono gt-hero-bar-count">
                          {c.count[0].toLocaleString()}/
                          {c.count[1].toLocaleString()}
                        </span>
                      </div>
                      <ProgressBar
                        value={c.pct}
                        showVal
                        valText={`${c.pct}%`}
                        color={
                          c.id === "exotics"
                            ? "var(--c-exotic)"
                            : "var(--c-signal)"
                        }
                      />
                    </button>
                  ))}
            </div>
          </div>
        )}
      </Panel>

      {/* DO THIS TODAY */}
      <Panel
        title="Do this today"
        icon="bolt"
        accent="var(--c-signal)"
        right={
          <button className="gt-link" onClick={() => go("/this-week")}>
            Full week <Icon name="chevron" size="0.8rem" />
          </button>
        }
      >
        {!weeklyLoading && weeklyData?.recommended?.[0] && (
          <button
            className="gt-today-row"
            style={{ marginBottom: "var(--s-2)" }}
            onClick={() => go("/this-week")}
          >
            <Icon
              name="bolt"
              size="1.2rem"
              style={{ color: "var(--c-signal)" }}
            />
            <div className="gt-today-main">
              <div className="gt-action-meta mono">Best thing to do today</div>
              <div className="gt-today-text">
                <strong>{weeklyData.recommended[0].text}</strong>
                {weeklyData.recommended[0].detail && (
                  <>
                    {" "}
                    —{" "}
                    <span style={{ fontWeight: "normal" }}>
                      {weeklyData.recommended[0].detail}
                    </span>
                  </>
                )}
              </div>
            </div>
          </button>
        )}
        <div className="gt-today">
          {weeklyFailed ? (
            <div
              className="gt-today-row"
              style={{ pointerEvents: "none", opacity: 0.6 }}
            >
              <Icon
                name="bolt"
                size="1.2rem"
                style={{ color: "var(--c-text-3)" }}
              />
              <div className="gt-today-main">
                <div className="gt-today-text">
                  Couldn't load this week's data — try again after a refresh.
                </div>
              </div>
            </div>
          ) : weeklyLoading ? (
            [0, 1, 2].map((i) => (
              <div
                key={i}
                className="gt-today-row"
                style={{ pointerEvents: "none" }}
              >
                <Skeleton w="1.2rem" h="1.2rem" r="50%" />
                <div
                  className="gt-today-main"
                  style={{
                    flex: 1,
                    display: "flex",
                    flexDirection: "column",
                    gap: "var(--s-2)",
                  }}
                >
                  <Skeleton w="70%" h="0.9rem" />
                  <Skeleton w="45%" h="0.7rem" />
                </div>
                <Skeleton w="4rem" h="1.4rem" r="var(--r-sm)" />
              </div>
            ))
          ) : (weeklyData?.dailyActions ?? []).length === 0 ? (
            <div
              className="gt-today-row"
              style={{ pointerEvents: "none", opacity: 0.6 }}
            >
              <Icon
                name="bolt"
                size="1.2rem"
                style={{ color: "var(--c-text-3)" }}
              />
              <div className="gt-today-main">
                <div className="gt-today-text">
                  Nothing urgent today — check back after daily reset.
                </div>
              </div>
            </div>
          ) : (
            (weeklyData?.dailyActions ?? []).map((action: DailyAction) => {
              const iconColor =
                action.category === "xur"
                  ? "var(--c-exotic)"
                  : action.category === "vendor"
                    ? "var(--c-legendary)"
                    : action.category === "activity"
                      ? "var(--c-rare)"
                      : "var(--c-signal)";
              const timingLabel =
                action.category === "xur" ? "leaves in" : "resets in";
              const timingStr = formatDuration(action.resetsIn);
              return (
                <button
                  key={action.id}
                  className="gt-today-row"
                  onClick={() => go("/this-week")}
                >
                  <Icon
                    name={action.icon as any}
                    size="1.2rem"
                    style={{ color: iconColor }}
                  />
                  <div className="gt-today-main">
                    <div className="gt-today-text">
                      <strong>{action.text}</strong>
                      {action.detail && (
                        <>
                          {" "}
                          —{" "}
                          <span style={{ fontWeight: "normal" }}>
                            {action.detail}
                          </span>
                        </>
                      )}
                    </div>
                    {timingStr && (
                      <div className="gt-action-meta mono">
                        {timingLabel} {timingStr}
                      </div>
                    )}
                  </div>
                  <Badge kind={action.badge as any} dot />
                </button>
              );
            })
          )}
        </div>
      </Panel>

      <div className="gt-dash-cols">
        <Panel
          title="This week — preview"
          icon="week"
          right={
            <button className="gt-link" onClick={() => go("/this-week")}>
              See all <Icon name="chevron" size="0.8rem" />
            </button>
          }
        >
          <ul className="gt-vendor-list">
            {weeklyFailed ? (
              <li
                className="gt-today-row"
                style={{ pointerEvents: "none", opacity: 0.6 }}
              >
                <Icon
                  name="bolt"
                  size="1.2rem"
                  style={{ color: "var(--c-text-3)" }}
                />
                <div className="gt-today-main">
                  <div className="gt-today-text">
                    Couldn't load this week's data — try again after a refresh.
                  </div>
                </div>
              </li>
            ) : weeklyLoading ? (
              [0, 1].map((i) => (
                <li key={i} className="gt-milestone">
                  <div
                    className="gt-milestone-l"
                    style={{
                      display: "flex",
                      flexDirection: "column",
                      gap: "var(--s-2)",
                      flex: 1,
                    }}
                  >
                    <Skeleton w="4rem" h="0.65rem" />
                    <Skeleton w="70%" h="0.9rem" />
                    <Skeleton w="55%" h="0.7rem" />
                  </div>
                  <Skeleton w="5rem" h="1.5rem" r="var(--r-sm)" />
                </li>
              ))
            ) : (
              (weeklyData?.milestones ?? []).slice(0, 2).map((m) => (
                <li key={m.id} className="gt-milestone">
                  <div className="gt-milestone-l">
                    <div className="gt-action-meta mono">{m.label}</div>
                    <div className="gt-item-name">{m.name}</div>
                    <div className="gt-item-type">Reward: {m.reward}</div>
                  </div>
                  {m.missing != null && m.missing > 0 && (
                    <Badge kind="missing" dot>
                      {m.missing} missing
                    </Badge>
                  )}
                </li>
              ))
            )}
          </ul>
        </Panel>

        <Panel
          title="Wishlist available now"
          icon="wishlist"
          accent="var(--c-avail)"
          right={
            <button className="gt-link" onClick={() => go("/wishlist")}>
              Manage <Icon name="chevron" size="0.8rem" />
            </button>
          }
        >
          {!wishlistFailed && (
            <div className="gt-avail-head">
              {wishlistLoading ? (
                <Skeleton
                  w="2.5rem"
                  h="1.2rem"
                  style={{ display: "inline-block" }}
                />
              ) : (
                <span className="gt-avail-num mono">{availableNowCount}</span>
              )}
              <span className="gt-avail-text">
                of {wishlistEntries.length} wishlist items available now
              </span>
            </div>
          )}
          <div className="gt-avail-list">
            {wishlistFailed ? (
              <div
                className="gt-item gt-item--compact"
                style={{ pointerEvents: "none", opacity: 0.6 }}
              >
                <Icon
                  name="wishlist"
                  size="1.2rem"
                  style={{ color: "var(--c-text-3)" }}
                />
                <div className="gt-item-head" style={{ flex: 1 }}>
                  <div className="gt-item-name">
                    Couldn't load your wishlist.
                  </div>
                </div>
              </div>
            ) : wishlistLoading ? (
              [0, 1, 2].map((i) => (
                <div
                  key={i}
                  className="gt-item gt-item--compact"
                  style={{ pointerEvents: "none" }}
                >
                  <Skeleton w="1.9rem" h="1.9rem" r="var(--r-sm)" />
                  <div
                    className="gt-item-head"
                    style={{
                      flex: 1,
                      display: "flex",
                      flexDirection: "column",
                      gap: "var(--s-2)",
                    }}
                  >
                    <Skeleton w="65%" h="0.9rem" />
                    <Skeleton w="40%" h="0.7rem" />
                  </div>
                  <Skeleton w="3.5rem" h="1.2rem" r="var(--r-sm)" />
                </div>
              ))
            ) : (
              wishlistEntries
                .slice()
                .sort((a, b) => (b.avail.now ? 1 : 0) - (a.avail.now ? 1 : 0))
                .slice(0, 3)
                .map((entry) => (
                  <button
                    key={entry.id}
                    className="gt-item gt-item--compact"
                    data-rarity={entry.rarity}
                    onClick={() => go("/wishlist")}
                  >
                    <ItemTile
                      rarity={entry.rarity}
                      type={entry.type}
                      icon={entry.icon}
                      style={{ width: "1.9rem" }}
                    />
                    <div className="gt-item-head" style={{ flex: 1 }}>
                      <div className="gt-item-name">{entry.name}</div>
                      <div className="gt-item-type">{entry.type}</div>
                    </div>
                    {entry.avail.now && <Badge kind="avail-now" dot />}
                  </button>
                ))
            )}
          </div>
        </Panel>
      </div>

      <div className="gt-dash-actions">
        <Button
          variant="outline"
          icon="collections"
          onClick={() => go("/collections")}
        >
          View Collections
        </Button>
        <Button
          variant="outline"
          icon="wishlist"
          onClick={() => go("/wishlist")}
        >
          Manage Wishlist
        </Button>
        <DataFreshnessChip updatedAt={real?.fetchedAt} />
      </div>
    </div>
  );
}
