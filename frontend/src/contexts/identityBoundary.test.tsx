import React, { StrictMode, useState, useEffect } from "react";
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { QueryClient, useQueryClient, useQuery } from "@tanstack/react-query";
import { http, HttpResponse } from "msw";
import { AppProviders, AuthedProviders } from "./AppProviders";
import { useAuth } from "./AuthContext";
import { usePreferences } from "./PreferencesContext";
import { useCharacters } from "./CharacterContext";
import { useIdentityMutation } from "./IdentityMutation";
import { seedBrowserSession } from "../test/browserSession";
import { browserSessionClient } from "../lib/browserSessionBrowser";
import { API, sampleUser, server } from "../test/testServer";

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (error: Error) => void;
  const promise = new Promise<T>((yes, no) => {
    resolve = yes;
    reject = no;
  });
  return { promise, resolve, reject };
}
const otherUser = {
  ...sampleUser,
  id: "other",
  membershipId: "other",
  displayName: "Other Guardian",
};
function client() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
}

function Probe({
  clients,
  renders,
}: {
  clients: QueryClient[];
  renders?: string[];
}) {
  const cache = useQueryClient();
  const { user } = useAuth();
  const preferences = usePreferences();
  const [draft, setDraft] = useState("");
  if (!clients.includes(cache)) clients.push(cache);
  renders?.push(
    `${user?.membershipId}:${cache.getQueryData<string>(["private"])}:${preferences.cardStyle}:${draft}`,
  );
  return (
    <>
      <div>{user?.displayName ?? "anonymous"}</div>
      <div data-testid="private">
        {cache.getQueryData<string>(["private"]) ?? "empty"}
      </div>
      <div data-testid="style">{preferences.cardStyle}</div>
      <input
        aria-label="Draft"
        value={draft}
        onChange={(event) => setDraft(event.target.value)}
      />
    </>
  );
}

describe("application identity boundaries", () => {
  it.each([
    otherUser,
    { ...sampleUser, membershipType: sampleUser.membershipType + 1 },
  ])(
    "replaces the cache and provider subtree before rendering a different membership ($membershipId/$membershipType)",
    (nextUser) => {
      seedBrowserSession();
      const initial = client();
      initial.setQueryData(["private"], "A private rows");
      localStorage.setItem("gt_done:reset-time", '["A-action"]');
      localStorage.setItem("gt.collections.filters", '{"sort":"name"}');
      localStorage.setItem(
        "guardian_prefs",
        JSON.stringify({ cardStyle: "compact", personalize: "off" }),
      );
      server.use(
        http.get(`${API}/api/preferences`, () => new Promise(() => {})),
      );
      const clients: QueryClient[] = [];
      const renders: string[] = [];
      render(
        <AppProviders client={initial}>
          <Probe clients={clients} renders={renders} />
        </AppProviders>,
      );
      fireEvent.change(screen.getByLabelText("Draft"), {
        target: { value: "A draft" },
      });
      const start = renders.length;
      act(() => seedBrowserSession(nextUser));
      expect(screen.getByTestId("private")).toHaveTextContent("empty");
      expect(screen.getByTestId("style")).toHaveTextContent("framed");
      expect(screen.getByLabelText("Draft")).toHaveValue("");
      expect(initial.getQueryCache().getAll()).toHaveLength(0);
      expect(clients).toHaveLength(2);
      expect(localStorage.getItem("gt_done:reset-time")).toBeNull();
      expect(localStorage.getItem("gt.collections.filters")).toBe(
        '{"sort":"name"}',
      );
      expect(clients[1].getDefaultOptions()).toEqual(
        initial.getDefaultOptions(),
      );
      expect(
        renders
          .slice(start)
          .every(
            (value) =>
              !value.includes("A private rows") &&
              !value.includes("A draft") &&
              !value.includes("compact"),
          ),
      ).toBe(true);
    },
  );

  it("keeps the cache and mounted draft on same-membership refresh under StrictMode", () => {
    seedBrowserSession();
    const initial = client();
    initial.setQueryData(["private"], "retained");
    localStorage.setItem("gt_done:reset-time", '["A-action"]');
    const clients: QueryClient[] = [];
    render(
      <StrictMode>
        <AppProviders client={initial}>
          <Probe clients={clients} />
        </AppProviders>
      </StrictMode>,
    );
    fireEvent.change(screen.getByLabelText("Draft"), {
      target: { value: "keep my draft" },
    });
    act(() =>
      seedBrowserSession(
        { ...sampleUser, displayName: "Updated name" },
        "fresh-token",
      ),
    );
    expect(screen.getByText("Updated name")).toBeInTheDocument();
    expect(screen.getByLabelText("Draft")).toHaveValue("keep my draft");
    expect(screen.getByTestId("private")).toHaveTextContent("retained");
    expect(clients).toEqual([initial]);
    expect(localStorage.getItem("gt_done:reset-time")).toBe('["A-action"]');
  });

  it("clears on logout and does not reuse an earlier cache when the same member logs in again", async () => {
    seedBrowserSession();
    const initial = client();
    initial.setQueryData(["private"], "departed");
    const clients: QueryClient[] = [];
    render(
      <AppProviders client={initial}>
        <Probe clients={clients} />
      </AppProviders>,
    );
    await act(() => browserSessionClient.end("current"));
    expect(screen.getByText("anonymous")).toBeInTheDocument();
    expect(initial.getQueryCache().getAll()).toHaveLength(0);
    act(() => seedBrowserSession());
    expect(screen.getByTestId("private")).toHaveTextContent("empty");
    expect(clients).toHaveLength(3);
  });

  it("cancels a late query and ignores a departing mutation rollback", async () => {
    seedBrowserSession();
    const initial = client();
    initial.setQueryData(["private"], "A rows");
    const read = deferred<string>();
    const write = deferred<void>();
    const rollback = vi.fn<() => void>();
    let capturedSignal: AbortSignal | undefined;
    const clients: QueryClient[] = [];
    function PendingWork() {
      const cache = useQueryClient();
      const { user } = useAuth();
      useQuery({
        queryKey: ["slow"],
        queryFn: ({ signal }) => {
          capturedSignal = signal;
          return read.promise;
        },
        enabled: user?.membershipId === sampleUser.membershipId,
      });
      const mutation = useIdentityMutation({
        mutationFn: () => write.promise,
        onMutate: () => {
          const previous = cache.getQueryData(["private"]);
          cache.setQueryData(["private"], "optimistic A");
          return previous;
        },
        onError: (_error, _variables, previous) => {
          rollback();
          cache.setQueryData(["private"], previous);
        },
      });
      return (
        <>
          <Probe clients={clients} />
          <button onClick={() => mutation.mutate()}>Mutate</button>
        </>
      );
    }
    render(
      <AppProviders client={initial}>
        <PendingWork />
      </AppProviders>,
    );
    fireEvent.click(screen.getByText("Mutate"));
    await waitFor(() =>
      expect(initial.getQueryData(["private"])).toBe("optimistic A"),
    );
    act(() => seedBrowserSession(otherUser));
    expect(capturedSignal?.aborted).toBe(true);
    await act(async () => {
      read.resolve("late A result");
      write.reject(new Error("late failure"));
      await read.promise;
    });
    expect(rollback).not.toHaveBeenCalled();
    expect(clients[1].getQueryData(["private"])).toBeUndefined();
    expect(clients[1].getQueryData(["slow"])).toBeUndefined();
    expect(initial.getQueryCache().getAll()).toHaveLength(0);
  });

  it("does not send a mutation whose optimistic preparation crosses a boundary", async () => {
    seedBrowserSession();
    const initial = client();
    const prepared = deferred<void>();
    const send = vi.fn<() => Promise<void>>(() => Promise.resolve());
    const started = vi.fn<() => void>();
    function MutationProbe() {
      const mutation = useIdentityMutation({
        mutationFn: send,
        onMutate: async () => {
          started();
          await prepared.promise;
        },
      });
      return <button onClick={() => mutation.mutate()}>Mutate</button>;
    }
    render(
      <AppProviders client={initial}>
        <MutationProbe />
      </AppProviders>,
    );
    fireEvent.click(screen.getByText("Mutate"));
    await waitFor(() => expect(started).toHaveBeenCalled());
    act(() => seedBrowserSession(otherUser));
    await act(async () => {
      prepared.resolve();
      await prepared.promise;
    });
    expect(send).not.toHaveBeenCalled();
  });
  it("does not reuse a departed client when the entire app was unmounted during replacement", () => {
    seedBrowserSession();
    const initial = client();
    initial.setQueryData(["private"], "A private rows");
    const first = render(
      <AppProviders client={initial}>
        <Probe clients={[]} />
      </AppProviders>,
    );
    first.unmount();
    act(() => seedBrowserSession(otherUser));
    const clients: QueryClient[] = [];
    render(
      <AppProviders client={initial}>
        <Probe clients={clients} />
      </AppProviders>,
    );
    expect(screen.getByTestId("private")).toHaveTextContent("empty");
    expect(clients[0]).not.toBe(initial);
    expect(initial.getQueryCache().getAll()).toHaveLength(0);
  });

  it("replaces the identity even when reconnect storage cleanup throws", () => {
    seedBrowserSession();
    const initial = client();
    initial.setQueryData(["private"], "A private rows");
    const clients: QueryClient[] = [];
    render(
      <AppProviders client={initial}>
        <Probe clients={clients} />
      </AppProviders>,
    );
    const removal = vi
      .spyOn(Storage.prototype, "removeItem")
      .mockImplementation(() => {
        throw new Error("Storage unavailable");
      });
    try {
      act(() => seedBrowserSession(otherUser));
      expect(screen.getByText("Other Guardian")).toBeInTheDocument();
      expect(screen.getByTestId("private")).toHaveTextContent("empty");
      expect(clients).toHaveLength(2);
    } finally {
      removal.mockRestore();
    }
  });
  it("resets character and onboarding state and ignores late preferences and old setters", async () => {
    seedBrowserSession(sampleUser, "token-A");
    const initial = client();
    initial.setQueryData(
      ["characters", sampleUser.membershipType, sampleUser.membershipId],
      [
        {
          characterId: "A-character",
          classType: 1,
          className: "Hunter",
          raceName: "Human",
          light: 2000,
        },
      ],
    );
    const oldRead = deferred<Response>();
    let writes = 0;
    server.use(
      http.get(`${API}/api/preferences`, ({ request }) =>
        request.headers.get("Authorization") === "Bearer token-A"
          ? oldRead.promise
          : HttpResponse.json({
              cardStyle: "framed",
              personalize: true,
              onboardedAt: null,
            }),
      ),
      http.put(`${API}/api/preferences`, () => {
        writes += 1;
        return HttpResponse.json({});
      }),
      http.get(`${API}/api/characters/:type/:id`, () => HttpResponse.json([])),
    );
    let oldSetter: ((style: "compact") => void) | undefined;
    function StateProbe() {
      const { user } = useAuth();
      const preferences = usePreferences();
      const { activeCharacter } = useCharacters();
      useEffect(() => {
        if (user?.membershipId === sampleUser.membershipId)
          oldSetter = preferences.setCardStyle;
      }, [user?.membershipId, preferences.setCardStyle]);
      return (
        <div data-testid="state">
          {activeCharacter?.id ?? "no-character"}/{preferences.cardStyle}/
          {String(preferences.onboardedAt)}/
          {String(preferences.preferencesReady)}
        </div>
      );
    }
    render(
      <AppProviders client={initial}>
        <AuthedProviders>
          <StateProbe />
        </AuthedProviders>
      </AppProviders>,
    );
    expect(screen.getByTestId("state")).toHaveTextContent("A-character");
    act(() => seedBrowserSession(otherUser, "token-B"));
    await waitFor(() =>
      expect(screen.getByTestId("state")).toHaveTextContent(
        "no-character/framed/null/true",
      ),
    );
    await act(async () => {
      oldSetter?.("compact");
      oldRead.resolve(
        HttpResponse.json({
          cardStyle: "compact",
          personalize: false,
          onboardedAt: "A-completed",
        }),
      );
      await oldRead.promise;
    });
    expect(screen.getByTestId("state")).toHaveTextContent(
      "no-character/framed/null/true",
    );
    expect(writes).toBe(0);
  });
});
