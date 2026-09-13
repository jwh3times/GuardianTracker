import { describe, it, expect } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { QueryClient } from "@tanstack/react-query";
import { server, API, sampleCollections } from "../test/testServer";
import { renderWithProviders } from "../test/renderWithProviders";
import { useCollections } from "./collections";
import {
  useMembershipRefresh,
  useReloadAfterReconnect,
} from "./membershipRefresh";
import { useWishlist } from "./wishlist";

/**
 * Contract tests for the membership-refresh module (ADR 0020): the fan-out a
 * refresh performs, the completeness guard that fails when a membership-scoped
 * data module is added without being wired in (the failure ADR 0018 cares
 * about), and the wider reload after a Bungie reconnect.
 */

function RefreshProbe() {
  const { refresh, isRefreshing } = useMembershipRefresh();
  return (
    <div>
      <button onClick={refresh}>refresh</button>
      <div data-testid="pending">{isRefreshing ? "yes" : "no"}</div>
    </div>
  );
}

/**
 * A QueryClient that records the key families it was asked to invalidate.
 * Asserting on invalidation shows the whole fan-out at once; each resource's
 * own contract test proves its root key reaches a real cache entry.
 */
function trackingClient() {
  const client = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
  const invalidated: string[] = [];
  const original = client.invalidateQueries.bind(client);
  client.invalidateQueries = ((filters?: { queryKey?: readonly unknown[] }) => {
    const head = filters?.queryKey?.[0];
    if (typeof head === "string") invalidated.push(head);
    return original(filters as never);
  }) as typeof client.invalidateQueries;
  return Object.assign(client, { invalidated });
}

describe("membership refresh fan-out", () => {
  it("invalidates every membership-scoped resource ADR 0018 names", async () => {
    server.use(
      http.post(`${API}/api/collections/:type/:id/refresh`, () =>
        HttpResponse.json({ refreshed: true }),
      ),
    );
    const client = trackingClient();
    renderWithProviders(<RefreshProbe />, { client });

    await userEvent.click(screen.getByText("refresh"));

    await waitFor(() => expect(client.invalidated.length).toBeGreaterThan(0));
    // Dropping any of these silently leaves a stale page after a refresh.
    expect([...client.invalidated].sort()).toEqual([
      "catalysts",
      "characters",
      "collections",
      "crafting",
      "seals",
      "weekly",
    ]);
  });

  it("does not invalidate anything when the refresh fails", async () => {
    server.use(
      http.post(`${API}/api/collections/:type/:id/refresh`, () =>
        HttpResponse.json({ message: "nope" }, { status: 500 }),
      ),
    );
    const client = trackingClient();
    renderWithProviders(<RefreshProbe />, { client });

    await userEvent.click(screen.getByText("refresh"));

    await waitFor(() =>
      expect(screen.getByTestId("pending")).toHaveTextContent("no"),
    );
    expect(client.invalidated).toEqual([]);
  });
});

/*
 * Completeness. The membership-scoped set is an explicit list in one file
 * (ADR 0020 rejected an import-time registry), so what guards it is this test:
 * every data module that reads the signed-in membership must be refreshed
 * through its own invalidation entry point, or be exempt by name with a reason.
 */
const DATA_MODULES = import.meta.glob<string>(
  ["./*.ts", "./*.tsx", "!./*.test.ts", "!./*.test.tsx"],
  { query: "?raw", import: "default", eager: true },
);

const REFRESH_MODULE = "./membershipRefresh.ts";

/** Modules that read the membership but must never be refreshed, and why. */
const REFRESH_EXEMPTIONS: Record<string, string> = {
  "./preferences.ts":
    "A Bungie data refresh cannot change a Guardian Tracker setting (ADR 0021).",
};

function readsMembership(source: string): boolean {
  return /\buseAuth\b|\bbrowserSessionClient\b/.test(source);
}

function specifierOf(path: string): string {
  return path.replace(/\.tsx?$/, "");
}

function escapeRegExp(text: string): string {
  return text.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

const refreshSource = DATA_MODULES[REFRESH_MODULE] ?? "";
const membershipScoped = Object.keys(DATA_MODULES)
  .filter(
    (path) =>
      path !== REFRESH_MODULE && readsMembership(DATA_MODULES[path] ?? ""),
  )
  .sort();
const mustBeRefreshed = membershipScoped.filter(
  (path) => !(path in REFRESH_EXEMPTIONS),
);

describe("membership refresh completeness", () => {
  it("detects the membership-scoped modules, so the checks below guard something", () => {
    expect(membershipScoped).toEqual(
      expect.arrayContaining([
        "./catalysts.ts",
        "./characters.ts",
        "./collections.ts",
        "./crafting.ts",
        "./preferences.ts",
        "./seals.ts",
        "./weekly.ts",
      ]),
    );
  });

  it.each(mustBeRefreshed)(
    "refreshes %s through its own invalidation entry point",
    (path) => {
      const source = DATA_MODULES[path] ?? "";
      const entry =
        /export function (invalidate[A-Z]\w*)\(/.exec(source)?.[1] ?? "(none)";
      expect(
        entry,
        `${path} reads the membership but exports no invalidate<Resource> entry point`,
      ).not.toBe("(none)");
      expect(refreshSource).toMatch(
        new RegExp(
          `import \\{ ${entry} \\} from "${escapeRegExp(specifierOf(path))}";`,
        ),
      );
      expect(refreshSource).toMatch(new RegExp(`\\b${entry}\\(client\\);`));
    },
  );

  it.each(Object.entries(REFRESH_EXEMPTIONS))(
    "records %s as a deliberate exemption",
    (path) => {
      expect(
        DATA_MODULES[path],
        `${path} no longer exists; remove its exemption`,
      ).toBeDefined();
      expect(readsMembership(DATA_MODULES[path] ?? "")).toBe(true);
      expect(refreshSource).not.toContain(`from "${specifierOf(path)}"`);
    },
  );
});

function ReconnectProbe() {
  const reload = useReloadAfterReconnect();
  const { view } = useCollections();
  const { entries } = useWishlist();
  return (
    <div>
      <button onClick={() => void reload()}>reload</button>
      <div data-testid="loaded">
        {view ? "collections" : "-"}|{entries.length}
      </div>
    </div>
  );
}

describe("reload after reconnect", () => {
  it("re-fetches every cached resource, not only membership-scoped ones", async () => {
    let collectionsGets = 0;
    let wishlistGets = 0;
    server.use(
      http.get(`${API}/api/collections/:type/:id`, () => {
        collectionsGets += 1;
        return HttpResponse.json(sampleCollections);
      }),
      http.get(`${API}/api/wishlist`, () => {
        wishlistGets += 1;
        return HttpResponse.json([]);
      }),
    );

    renderWithProviders(<ReconnectProbe />);
    await waitFor(() => expect(collectionsGets).toBe(1));
    await waitFor(() => expect(wishlistGets).toBe(1));

    await userEvent.click(screen.getByText("reload"));

    // The wish list is not membership-refreshed; only reloading everything
    // re-asks for it.
    await waitFor(() => expect(wishlistGets).toBe(2));
    await waitFor(() => expect(collectionsGets).toBe(2));
  });
});
