import { describe, it, expect } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { server, API, sampleAdminFlags } from "../test/testServer";
import { renderWithProviders } from "../test/renderWithProviders";
import {
  useAdminFlags,
  useAdminUsers,
  useAuditLog,
  useSetMemberRole,
  useUpdateFlag,
} from "./admin";

/**
 * Contract tests for the Admin data-access module (ADR 0020) — key identity
 * for the roster, the flag configuration and the per-type audit log; the audit
 * transport and its tab gate; projection to domain types; and both mutations'
 * request, invalidation and callback behavior.
 */

function user(id: string, overrides: Record<string, unknown> = {}) {
  return {
    id,
    displayName: `Member ${id}`,
    membershipId: `m-${id}`,
    platform: "steam",
    role: "beta",
    lastActive: "2026-09-01T00:00:00Z",
    ...overrides,
  };
}

function UsersProbe() {
  const { users, isLoading, isError, retry } = useAdminUsers();
  return (
    <div>
      <button onClick={retry}>retry users</button>
      <div data-testid="users">
        {isLoading
          ? "loading"
          : isError
            ? "failed"
            : users
                .map(
                  (u) =>
                    `${u.id}|${u.displayName}|${u.membershipId}|${u.platform}|${u.role}|${u.lastActive}`,
                )
                .join(",")}
      </div>
    </div>
  );
}

function FlagsProbe() {
  const { flags, isLoading, isError, retry } = useAdminFlags();
  return (
    <div>
      <button onClick={retry}>retry flags</button>
      <div data-testid="flags">
        {isLoading
          ? "loading"
          : isError
            ? "failed"
            : flags
                .map(
                  (f) =>
                    `${f.key}|${f.name}|${f.description}|${f.category}|${f.minTier}|${f.enabled}|${f.updatedAt}`,
                )
                .join(",")}
      </div>
    </div>
  );
}

function AuditProbe({
  type,
  enabled,
  id,
}: {
  type: string;
  enabled?: boolean;
  id: string;
}) {
  const { entries, isLoading } = useAuditLog(type, { enabled });
  return (
    <div data-testid={id}>
      {isLoading
        ? "loading"
        : entries
            .map(
              (e) =>
                `${e.id}|${e.eventType}|${e.outcome}|${e.actor.displayName}/${e.actor.membershipId}|${
                  e.target
                    ? `${e.target.displayName}/${e.target.membershipId}`
                    : "no-target"
                }|${e.ip ?? "no-ip"}|${JSON.stringify(e.details)}|${e.createdAt}`,
            )
            .join(",") || "empty"}
    </div>
  );
}

describe("admin users", () => {
  it("projects the roster and shares one request between readers", async () => {
    let requests = 0;
    server.use(
      http.get(`${API}/api/admin/users`, () => {
        requests += 1;
        return HttpResponse.json([user("1", { role: "admin" }), user("2")]);
      }),
    );

    renderWithProviders(
      <>
        <UsersProbe />
        <UsersProbe />
      </>,
    );

    await waitFor(() =>
      expect(screen.getAllByTestId("users")[0]).toHaveTextContent(
        "1|Member 1|m-1|steam|admin|2026-09-01T00:00:00Z,2|Member 2|m-2|steam|beta|2026-09-01T00:00:00Z",
      ),
    );
    expect(requests).toBe(1);
  });

  it("surfaces a failure and refetches on retry", async () => {
    let attempt = 0;
    server.use(
      http.get(`${API}/api/admin/users`, () => {
        attempt += 1;
        return attempt === 1
          ? HttpResponse.json({ error: "boom" }, { status: 500 })
          : HttpResponse.json([user("1")]);
      }),
    );

    renderWithProviders(<UsersProbe />);
    await waitFor(() =>
      expect(screen.getByTestId("users")).toHaveTextContent("failed"),
    );

    await userEvent.click(screen.getByText("retry users"));

    await waitFor(() =>
      expect(screen.getByTestId("users")).toHaveTextContent("Member 1"),
    );
  });
});

describe("admin flags", () => {
  it("projects the flag configuration, typing the gate tier", async () => {
    let requests = 0;
    server.use(
      http.get(`${API}/api/admin/flags`, () => {
        requests += 1;
        return HttpResponse.json([
          sampleAdminFlags[1],
          { ...sampleAdminFlags[0], minTier: "legend" },
        ]);
      }),
    );

    renderWithProviders(
      <>
        <FlagsProbe />
        <FlagsProbe />
      </>,
    );

    const [a, b] = sampleAdminFlags;
    await waitFor(() =>
      expect(screen.getAllByTestId("flags")[0]).toHaveTextContent(
        `${b.key}|${b.name}|${b.description}|${b.category}|alpha|false|${b.updatedAt},` +
          `${a.key}|${a.name}|${a.description}|${a.category}|standard|true|${a.updatedAt}`,
      ),
    );
    expect(requests).toBe(1);
  });

  it("surfaces a failure and refetches on retry", async () => {
    let attempt = 0;
    server.use(
      http.get(`${API}/api/admin/flags`, () => {
        attempt += 1;
        return attempt === 1
          ? HttpResponse.json({ error: "boom" }, { status: 500 })
          : HttpResponse.json(sampleAdminFlags);
      }),
    );

    renderWithProviders(<FlagsProbe />);
    await waitFor(() =>
      expect(screen.getByTestId("flags")).toHaveTextContent("failed"),
    );

    await userEvent.click(screen.getByText("retry flags"));

    await waitFor(() =>
      expect(screen.getByTestId("flags")).toHaveTextContent("god-roll"),
    );
  });
});

/** Records each audit request's query string and echoes the type back. */
function recordAudit() {
  const sent: { type: string | null; limit: string | null }[] = [];
  server.use(
    http.get(`${API}/api/admin/audit`, ({ request }) => {
      const params = new URL(request.url).searchParams;
      const type = params.get("type");
      sent.push({ type, limit: params.get("limit") });
      return HttpResponse.json({
        entries: [
          {
            id: `e-${type ?? "all"}`,
            eventType: type ?? "login.success",
            outcome: "success",
            actor: { membershipId: "mid-1", displayName: "Tester" },
            details: {},
            createdAt: "2026-09-01T00:00:00Z",
          },
        ],
        nextCursor: "",
      });
    }),
  );
  return sent;
}

describe("audit log", () => {
  it("keys each event type separately and shares one request per type", async () => {
    const sent = recordAudit();

    renderWithProviders(
      <>
        <AuditProbe type="" id="all-a" />
        <AuditProbe type="" id="all-b" />
        <AuditProbe type="role." id="roles" />
      </>,
    );

    await waitFor(() =>
      expect(screen.getByTestId("roles")).toHaveTextContent("e-role.|role."),
    );
    await waitFor(() =>
      expect(screen.getByTestId("all-a")).toHaveTextContent("e-all|"),
    );
    expect(screen.getByTestId("all-b")).toHaveTextContent("e-all|");
    expect(new Set(sent.map((s) => s.type))).toEqual(new Set([null, "role."]));
    expect(sent).toHaveLength(2);
  });

  it("asks for a hundred events and encodes the type", async () => {
    const sent = recordAudit();

    renderWithProviders(<AuditProbe type="a&b c" id="a" />);

    await waitFor(() =>
      expect(sent).toEqual([{ type: "a&b c", limit: "100" }]),
    );
  });

  it("requests nothing while the audit tab is closed", async () => {
    const sent = recordAudit();

    renderWithProviders(
      <>
        <AuditProbe type="logout." enabled={false} id="closed" />
        <AuditProbe type="refresh." id="open" />
      </>,
    );

    // A gated query is pending but not loading, so the request log is what
    // discriminates. Waiting on the open probe proves both had their chance.
    await waitFor(() =>
      expect(screen.getByTestId("open")).toHaveTextContent("e-refresh."),
    );
    expect(sent.map((s) => s.type)).toEqual(["refresh."]);
  });

  it("projects each entry, with and without a target and ip", async () => {
    server.use(
      http.get(`${API}/api/admin/audit`, () =>
        HttpResponse.json({
          entries: [
            {
              id: "1",
              eventType: "role.change.admin",
              outcome: "failure",
              actor: { membershipId: "mid-a", displayName: "Admin" },
              target: { membershipId: "mid-t", displayName: "Target" },
              ip: "203.0.113.7",
              userAgent: "ignored",
              details: { from: "beta", to: "alpha" },
              createdAt: "2026-09-02T00:00:00Z",
            },
            {
              id: "2",
              eventType: "login.success",
              outcome: "success",
              actor: { membershipId: "mid-b", displayName: "" },
              details: {},
              createdAt: "2026-09-03T00:00:00Z",
            },
          ],
          nextCursor: "",
        }),
      ),
    );

    renderWithProviders(<AuditProbe type="" id="a" />);

    await waitFor(() =>
      expect(screen.getByTestId("a")).toHaveTextContent(
        '1|role.change.admin|failure|Admin/mid-a|Target/mid-t|203.0.113.7|{"from":"beta","to":"alpha"}|2026-09-02T00:00:00Z,' +
          "2|login.success|success|/mid-b|no-target|no-ip|{}|2026-09-03T00:00:00Z",
      ),
    );
  });
});

function SetRoleProbe({ log }: { log: string[] }) {
  const { setRole } = useSetMemberRole({
    onSuccess: (vars) => log.push(`ok:${vars.id}:${vars.role}`),
    onError: (_e, vars) => log.push(`err:${vars.id}`),
  });
  return (
    <button onClick={() => setRole({ id: "2", role: "alpha" })}>
      set role
    </button>
  );
}

describe("set member role", () => {
  it("sends the new role, refreshes the roster and reports success", async () => {
    let body: unknown = null;
    let path = "";
    let gets = 0;
    server.use(
      http.get(`${API}/api/admin/users`, () => {
        gets += 1;
        return HttpResponse.json([
          user("2", { role: gets > 1 ? "alpha" : "beta" }),
        ]);
      }),
      http.put(
        `${API}/api/admin/users/:id/role`,
        async ({ request, params }) => {
          path = String(params.id);
          body = await request.json();
          return HttpResponse.json({
            id: "2",
            role: "alpha",
            previousRole: "beta",
          });
        },
      ),
    );
    const log: string[] = [];

    renderWithProviders(
      <>
        <UsersProbe />
        <SetRoleProbe log={log} />
      </>,
    );
    await waitFor(() =>
      expect(screen.getByTestId("users")).toHaveTextContent("|beta|"),
    );

    await userEvent.click(screen.getByText("set role"));

    // The roster refetch reaching the screen is what proves the invalidation
    // matched the real entry, not just that a call was made.
    await waitFor(() =>
      expect(screen.getByTestId("users")).toHaveTextContent("|alpha|"),
    );
    expect(path).toBe("2");
    expect(body).toEqual({ role: "alpha" });
    expect(log).toEqual(["ok:2:alpha"]);
  });

  it("reports a failure and leaves the roster alone", async () => {
    let gets = 0;
    server.use(
      http.get(`${API}/api/admin/users`, () => {
        gets += 1;
        return HttpResponse.json([user("2")]);
      }),
      http.put(`${API}/api/admin/users/:id/role`, () =>
        HttpResponse.json({ error: "nope" }, { status: 403 }),
      ),
    );
    const log: string[] = [];

    renderWithProviders(
      <>
        <UsersProbe />
        <SetRoleProbe log={log} />
      </>,
    );
    await waitFor(() => expect(gets).toBe(1));

    await userEvent.click(screen.getByText("set role"));

    await waitFor(() => expect(log).toEqual(["err:2"]));
    await new Promise((resolve) => setTimeout(resolve, 50));
    expect(gets).toBe(1);
  });
});

function UpdateFlagProbe({ log }: { log: string[] }) {
  const { updateFlag, isPending } = useUpdateFlag({
    onSuccess: (vars) => log.push(`ok:${vars.key}`),
    onError: (_e, vars) => log.push(`err:${vars.key}`),
  });
  return (
    <>
      <button
        onClick={() =>
          updateFlag({
            key: "god-roll",
            patch: { enabled: true, minTier: "beta" },
          })
        }
      >
        update flag
      </button>
      <div data-testid="pending">{String(isPending)}</div>
    </>
  );
}

describe("update flag", () => {
  it("sends the patch, refreshes the flag list and reports success", async () => {
    let body: unknown = null;
    let path = "";
    let gets = 0;
    server.use(
      http.get(`${API}/api/admin/flags`, () => {
        gets += 1;
        return HttpResponse.json([
          { ...sampleAdminFlags[1], enabled: gets > 1 },
        ]);
      }),
      http.put(`${API}/api/admin/flags/:key`, async ({ request, params }) => {
        path = String(params.key);
        body = await request.json();
        return HttpResponse.json({ ...sampleAdminFlags[1], enabled: true });
      }),
    );
    const log: string[] = [];

    renderWithProviders(
      <>
        <FlagsProbe />
        <UpdateFlagProbe log={log} />
      </>,
    );
    await waitFor(() =>
      expect(screen.getByTestId("flags")).toHaveTextContent("|alpha|false|"),
    );

    await userEvent.click(screen.getByText("update flag"));

    await waitFor(() =>
      expect(screen.getByTestId("flags")).toHaveTextContent("|alpha|true|"),
    );
    expect(path).toBe("god-roll");
    expect(body).toEqual({ enabled: true, minTier: "beta" });
    expect(log).toEqual(["ok:god-roll"]);
    expect(screen.getByTestId("pending")).toHaveTextContent("false");
  });

  it("reports a failure", async () => {
    server.use(
      http.put(`${API}/api/admin/flags/:key`, () =>
        HttpResponse.json({ error: "nope" }, { status: 500 }),
      ),
    );
    const log: string[] = [];

    renderWithProviders(<UpdateFlagProbe log={log} />);

    await userEvent.click(screen.getByText("update flag"));

    await waitFor(() => expect(log).toEqual(["err:god-roll"]));
  });
});
