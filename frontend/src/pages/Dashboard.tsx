import React from "react";
import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import {
  Badge,
  Button,
  CountdownChip,
  DataFreshnessChip,
  Icon,
  ItemTile,
  PageHead,
  Panel,
  ProgressBar,
  RadialProgress,
} from "../components/kit";
import { useAuth } from "../contexts/AuthContext";
import { apiFetch } from "../lib/api";
import { summary, weekly, wishlist } from "../lib/mockData";
import type { SummaryCategory } from "../types/design";
import type {
  ProfileResponse,
  APICollectionSummary,
  APIUserCollections,
} from "../types/api";

export function Dashboard() {
  const { user } = useAuth();
  const navigate = useNavigate();
  const go = (path: string) => navigate(path);

  const { data: profileData } = useQuery({
    queryKey: ["currentUser"],
    queryFn: () => apiFetch<ProfileResponse>("/api/auth/profile"),
  });

  const membershipType = profileData?.user.membershipType ?? user?.membershipType;
  const membershipId = profileData?.user.membershipId ?? user?.membershipId;

  const { data: real } = useQuery({
    queryKey: ["collections", membershipType, membershipId],
    queryFn: () =>
      apiFetch<APIUserCollections>(
        `/api/collections/${membershipType}/${membershipId}`
      ),
    enabled: membershipType != null && !!membershipId,
  });

  const displayName =
    profileData?.user.displayName || user?.displayName || "Guardian";

  const fromReal = (
    c: Pick<APICollectionSummary, "total" | "collected"> | undefined,
    fallback: SummaryCategory
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

  const [mWeapons, mArmor, mExotics, mCosmetics] = summary.categories;
  const categories: SummaryCategory[] = [
    fromReal(real?.weapons, mWeapons),
    fromReal(real?.armor, mArmor),
    fromReal(real?.exotics, mExotics),
    mCosmetics,
  ];
  const overall = Math.round(
    categories.reduce((sum, c) => sum + c.pct, 0) / categories.length
  );

  const availableWishlist = wishlist.filter((i) => i.avail.now);

  return (
    <div className="gt-page gt-dash">
      <PageHead
        title={`Welcome back, ${displayName}`}
        sub="Your collection at a glance"
        right={<CountdownChip prefix="Weekly reset" time={weekly.resetIn} icon="clock" />}
      />

      {/* HERO COMPLETION */}
      <Panel pad={false} style={{ padding: "var(--s-5)" }}>
        <div className="gt-hero">
          <div className="gt-hero-radial">
            <RadialProgress
              value={overall}
              size="clamp(8rem,9vw,10rem)"
              color="var(--c-exotic)"
              sub="overall"
              pctSize="clamp(2rem,3vw,2.6rem)"
            />
          </div>
          <div className="gt-hero-bars">
            {categories.map((c) => (
              <button key={c.id} className="gt-hero-bar" onClick={() => go("/collections")}>
                <div className="gt-hero-bar-top">
                  <span className="gt-hero-bar-label">{c.label}</span>
                  <span className="mono gt-hero-bar-count">
                    {c.count[0].toLocaleString()}/{c.count[1].toLocaleString()}
                  </span>
                </div>
                <ProgressBar
                  value={c.pct}
                  showVal
                  valText={`${c.pct}%`}
                  color={c.id === "exotics" ? "var(--c-exotic)" : "var(--c-signal)"}
                />
              </button>
            ))}
          </div>
        </div>
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
        <div className="gt-today">
          <button className="gt-today-row" onClick={() => go("/this-week")}>
            <Icon name="bungie" size="1.2rem" style={{ color: "var(--c-exotic)" }} />
            <div className="gt-today-main">
              <div className="gt-today-text">
                <strong>Xûr is selling Hawkmoon</strong> — a Hand Cannon exotic you're missing.
              </div>
              <div className="gt-action-meta mono">The Tower · leaves in 1d 6h</div>
            </div>
            <Badge kind="missing" dot icon="bolt" />
          </button>
          <button className="gt-today-row" onClick={() => go("/collections")}>
            <Icon name="collections" size="1.2rem" style={{ color: "var(--c-rare)" }} />
            <div className="gt-today-main">
              <div className="gt-today-text">
                <strong>Featured raid: Vault of Glass</strong> — 2 weapons you don't have yet.
              </div>
              <div className="gt-action-meta mono">Pinnacle gear · resets in 2d 14h</div>
            </div>
            <Badge kind="completes-set" dot />
          </button>
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
            {weekly.milestones.slice(0, 2).map((m) => (
              <li key={m.id} className="gt-milestone">
                <div className="gt-milestone-l">
                  <div className="gt-action-meta mono">{m.label}</div>
                  <div className="gt-item-name">{m.name}</div>
                  <div className="gt-item-type">Reward: {m.reward}</div>
                </div>
                {m.missing > 0 ? (
                  <Badge kind="missing" dot>
                    {m.missing} missing
                  </Badge>
                ) : (
                  <Badge kind="complete" dot />
                )}
              </li>
            ))}
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
          <div className="gt-avail-head">
            <span className="gt-avail-num mono">{availableWishlist.length}</span>
            <span className="gt-avail-text">of your wishlisted items are obtainable right now</span>
          </div>
          <div className="gt-avail-list">
            {availableWishlist.slice(0, 3).map((i) => (
              <button
                key={i.id}
                className="gt-item gt-item--compact"
                data-rarity={i.rarity}
                onClick={() => go("/wishlist")}
              >
                <ItemTile rarity={i.rarity} type={i.type} style={{ width: "1.9rem" }} />
                <div className="gt-item-head" style={{ flex: 1 }}>
                  <div className="gt-item-name">{i.name}</div>
                  <div className="gt-item-type">{i.avail.where}</div>
                </div>
                <Badge kind="avail-now" dot />
              </button>
            ))}
          </div>
        </Panel>
      </div>

      <div className="gt-dash-actions">
        <Button variant="outline" icon="collections" onClick={() => go("/collections")}>
          View Collections
        </Button>
        <Button variant="outline" icon="wishlist" onClick={() => go("/wishlist")}>
          Manage Wishlist
        </Button>
        <DataFreshnessChip ago={summary.updatedAgo} />
      </div>
    </div>
  );
}
