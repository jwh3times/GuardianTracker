import React, { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Button, EmptyState } from "../../../components/primitives";
import { Dropdown, PageHead, SealCard } from "../../../components/composite";
import { useAuth } from "../../../contexts/AuthContext";
import { apiFetch } from "../../../lib/api";
import { errorState } from "../../../lib/errorState";
import { LoadingSpinner } from "../../../components/LoadingSpinner";
import type { Seal } from "../../../types/design";
import type { APIRecordsEnvelope } from "../../../types/api";

type Sort = "closest" | "name";

export function Triumphs() {
  const { user } = useAuth();
  const membershipType = user?.membershipType;
  const membershipId = user?.membershipId;

  const {
    data: sealsData,
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: ["seals", membershipType, membershipId],
    queryFn: () =>
      apiFetch<APIRecordsEnvelope<Seal>>(
        `/api/seals/${membershipType}/${membershipId}`,
      ),
    enabled: membershipType != null && !!membershipId,
  });
  const seals = useMemo(() => sealsData?.items ?? [], [sealsData]);

  const [sort, setSort] = useState<Sort>("closest");
  // undefined = never interacted (auto-open first); null = user explicitly closed
  const [openId, setOpenId] = useState<string | null | undefined>(undefined);
  const effectiveOpenId =
    openId === undefined ? (seals[0]?.id ?? null) : openId;

  const sorted = useMemo(() => {
    const l = seals.slice();
    if (sort === "closest") {
      l.sort(
        (a, b) => (b.pct >= 100 ? -1 : b.pct) - (a.pct >= 100 ? -1 : a.pct),
      );
    } else if (sort === "name") {
      l.sort((a, b) => a.name.localeCompare(b.name));
    }
    return l;
  }, [seals, sort]);

  const gilded = seals.filter((s) => s.gilded > 0).length;

  if (isLoading) {
    return (
      <div className="gt-page">
        <div className="gt-page-loading">
          <LoadingSpinner size="lg" />
        </div>
      </div>
    );
  }

  if (isError) {
    const es = errorState(error);
    return (
      <div className="gt-page">
        <PageHead
          title="Triumphs & Seals"
          sub={<span className="mono">Seal completion</span>}
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
                    void refetch();
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

  return (
    <div className="gt-page">
      <PageHead
        title="Triumphs & Seals"
        sub={
          <span className="mono">
            {seals.length} seals · {gilded} gilded
          </span>
        }
        right={
          <Dropdown
            label="Sort: Closest to done"
            value={
              { closest: "Sort: Closest to done", name: "Sort: Name" }[sort]
            }
            noClear
            options={[
              { v: "closest", l: "Sort: Closest to done" },
              { v: "name", l: "Sort: Name" },
            ]}
            onPick={(v) => v && setSort(v as Sort)}
          />
        }
      />

      <div className="gt-seal-grid">
        {sorted.map((s) => (
          <SealCard
            key={s.id}
            seal={s}
            expanded={effectiveOpenId === s.id}
            onToggle={() => setOpenId(effectiveOpenId === s.id ? null : s.id)}
          />
        ))}
      </div>
    </div>
  );
}
