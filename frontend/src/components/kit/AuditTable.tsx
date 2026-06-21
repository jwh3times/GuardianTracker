import React from "react";
import { Badge } from "./primitives";
import { relTime } from "../../lib/adapters";
import type { APIAuditEntry } from "../../types/api";

const EVENT_LABEL: Record<string, string> = {
  "login.success": "Login",
  "login.failure": "Login failed",
  "logout.session": "Logout",
  "logout.all": "Logout (all devices)",
  "refresh.reuse": "Token reuse",
  "refresh.failure": "Refresh failed",
  "role.change.admin": "Role change (admin)",
  "role.optin": "Role opt-in",
  "flag.update": "Flag update",
};

function label(type: string): string {
  return EVENT_LABEL[type] ?? type;
}

export function AuditTable({
  entries,
  loading,
}: {
  entries: APIAuditEntry[];
  loading: boolean;
}) {
  if (loading && entries.length === 0) {
    return <div className="gt-audit-empty mono">Loading audit events…</div>;
  }
  if (entries.length === 0) {
    return <div className="gt-audit-empty mono">No audit events match.</div>;
  }
  return (
    <div className="gt-card gt-pad0">
      <div className="gt-audit-row gt-audit-row--head">
        <span className="gt-userrow-h">Event</span>
        <span className="gt-userrow-h">Actor</span>
        <span className="gt-userrow-h">Details</span>
        <span className="gt-userrow-h" style={{ textAlign: "right" }}>
          When
        </span>
      </div>
      {entries.map((e) => (
        <div className="gt-audit-row" key={e.id}>
          <span>
            <Badge kind={e.outcome === "failure" ? "urgent" : "complete"}>
              {label(e.eventType)}
            </Badge>
          </span>
          <span className="gt-audit-actor">
            {e.actor.displayName || e.actor.membershipId || "—"}
            {e.target ? (
              <span className="gt-audit-target mono">
                {" → "}
                {e.target.displayName || e.target.membershipId}
              </span>
            ) : null}
          </span>
          <span className="gt-audit-details mono">
            {Object.entries(e.details).map(([k, v]) => (
              <span className="gt-audit-pill" key={k}>
                {k}: {JSON.stringify(v)}
              </span>
            ))}
            {e.ip ? <span className="gt-audit-pill">ip: {e.ip}</span> : null}
          </span>
          <span className="gt-audit-when mono" style={{ textAlign: "right" }}>
            {relTime(e.createdAt)}
          </span>
        </div>
      ))}
    </div>
  );
}
