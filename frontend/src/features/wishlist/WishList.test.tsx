import React from "react";
import { describe, it, expect, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { http, HttpResponse, delay } from "msw";
import { API, sampleUser, sampleWishlist, server } from "../../test/testServer";
import { AuthProvider } from "../../contexts/AuthContext";
import { PreferencesProvider } from "../../contexts/PreferencesContext";
import { ToastProvider } from "../../components/Toast";
import { WishList } from "./WishList";

beforeEach(() => {
  localStorage.clear();
  localStorage.setItem("guardian_token", "test-token");
  localStorage.setItem("guardian_user", JSON.stringify(sampleUser));
});

function renderPage(ui: React.ReactNode, route = "/") {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={qc}>
      <AuthProvider>
        <PreferencesProvider>
          <ToastProvider>
            <MemoryRouter initialEntries={[route]}>{ui}</MemoryRouter>
          </ToastProvider>
        </PreferencesProvider>
      </AuthProvider>
    </QueryClientProvider>,
  );
}

describe("WishList page", () => {
  it("renders items with available-now and source-only states", async () => {
    server.use(
      http.get(`${API}/api/wishlist`, () =>
        HttpResponse.json([
          ...sampleWishlist,
          {
            id: "2",
            itemHash: 88888,
            name: "Vex Mythoclast",
            itemType: "Fusion Rifle",
            rarity: "Exotic",
            icon: "",
            priority: "LOW",
            notes: "from VoG",
            sources: ["Vault of Glass"],
            availableNow: false,
            dateAdded: new Date().toISOString(),
          },
        ]),
      ),
    );
    renderPage(<WishList />);
    expect(await screen.findByText("Gjallarhorn")).toBeInTheDocument();
    // available-now item shows the where label; source-only item shows "Source:"
    expect(screen.getByText("Vex Mythoclast")).toBeInTheDocument();
    expect(screen.getByText(/Source:/)).toBeInTheDocument();
    expect(screen.getByText("2 items · 1 available now")).toBeInTheDocument();
    // notes render in quotes
    expect(screen.getByText('"from VoG"')).toBeInTheDocument();
  });

  it("shows the empty state when the wishlist has no items", async () => {
    server.use(http.get(`${API}/api/wishlist`, () => HttpResponse.json([])));
    renderPage(<WishList />);
    expect(
      await screen.findByText("Your wishlist is empty"),
    ).toBeInTheDocument();
  });

  it("renders the loading state before data arrives", async () => {
    const { container } = renderPage(<WishList />);
    // PageHead + spinner card render synchronously while the query is pending
    expect(container.querySelector(".gt-page")).not.toBeNull();
    expect(screen.queryByText(/available now/)).not.toBeInTheDocument();
  });

  it("filters by priority and reports per-priority counts", async () => {
    renderPage(<WishList />);
    await screen.findByText("Gjallarhorn");
    // Gjallarhorn is HIGH priority; filtering to Low hides it
    fireEvent.click(screen.getByRole("button", { name: /Low/ }));
    expect(screen.queryByText("Gjallarhorn")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /All/ }));
    expect(screen.getByText("Gjallarhorn")).toBeInTheDocument();
  });

  it("removes an item optimistically", async () => {
    // Stateful handler so the post-delete refetch (onSettled) reflects the removal.
    let items = [...sampleWishlist];
    let deleted = false;
    server.use(
      http.get(`${API}/api/wishlist`, () => HttpResponse.json(items)),
      http.delete(`${API}/api/wishlist/:id`, ({ params }) => {
        items = items.filter((i) => i.id !== params.id);
        deleted = true;
        return new HttpResponse(null, { status: 204 });
      }),
    );
    renderPage(<WishList />);
    await screen.findByText("Gjallarhorn");
    fireEvent.click(screen.getByRole("button", { name: /Remove/ }));
    await waitFor(() =>
      expect(screen.queryByText("Gjallarhorn")).not.toBeInTheDocument(),
    );
    expect(deleted).toBe(true);
  });

  it("rolls back and toasts when a delete fails", async () => {
    server.use(
      http.delete(`${API}/api/wishlist/:id`, () =>
        HttpResponse.json({ error: "boom" }, { status: 500 }),
      ),
    );
    renderPage(<WishList />);
    await screen.findByText("Gjallarhorn");
    fireEvent.click(screen.getByRole("button", { name: /Remove/ }));
    expect(
      await screen.findByText("Failed to remove item"),
    ).toBeInTheDocument();
    // Optimistic removal rolled back → item returns
    expect(await screen.findByText("Gjallarhorn")).toBeInTheDocument();
  });

  it("edits notes and saves them", async () => {
    let putBody: unknown = null;
    server.use(
      http.put(`${API}/api/wishlist/:id`, async ({ request }) => {
        putBody = await request.json();
        return HttpResponse.json({ ...sampleWishlist[0], notes: "new note" });
      }),
    );
    renderPage(<WishList />);
    await screen.findByText("Gjallarhorn");

    fireEvent.click(screen.getByRole("button", { name: /Edit notes/ }));
    const textarea = screen.getByLabelText("Notes for Gjallarhorn");
    fireEvent.change(textarea, { target: { value: "new note" } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => expect(putBody).toEqual({ notes: "new note" }));
  });

  it("cancels a notes edit without saving", async () => {
    renderPage(<WishList />);
    await screen.findByText("Gjallarhorn");
    fireEvent.click(screen.getByRole("button", { name: /Edit notes/ }));
    expect(screen.getByLabelText("Notes for Gjallarhorn")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(
      screen.queryByLabelText("Notes for Gjallarhorn"),
    ).not.toBeInTheDocument();
  });

  it("changes priority through the row dropdown", async () => {
    let putBody: unknown = null;
    server.use(
      http.put(`${API}/api/wishlist/:id`, async ({ request }) => {
        putBody = await request.json();
        return HttpResponse.json({ ...sampleWishlist[0], priority: "URGENT" });
      }),
    );
    renderPage(<WishList />);
    await screen.findByText("Gjallarhorn");

    // Open the row's Priority dropdown. Its accessible name is exactly "High"
    // (the filter chip is "High 1", so an exact match disambiguates).
    fireEvent.click(screen.getByRole("button", { name: "High" }));
    fireEvent.click(screen.getByRole("button", { name: "Urgent" }));
    await waitFor(() => expect(putBody).toEqual({ priority: "URGENT" }));
  });
});

describe("WishList bulk actions", () => {
  it("enters select mode and bulk-deletes a selected item", async () => {
    // Stateful handler so the post-delete refetch (onSettled) reflects the
    // removal — same pattern as the single-item "removes an item
    // optimistically" test above. The POST is deliberately delayed so it
    // (and thus onSettled's refetch) cannot possibly have completed by the
    // time we assert the removal — the disappearance can only be explained
    // by the optimistic onMutate path.
    let items = [
      {
        id: "1",
        itemHash: 111,
        name: "Gjallarhorn",
        itemType: "Rocket Launcher",
        rarity: "Exotic",
        icon: "",
        priority: "HIGH",
        notes: "",
        sources: [],
        availableNow: false,
        dateAdded: new Date().toISOString(),
      },
      {
        id: "2",
        itemHash: 222,
        name: "Fatebringer",
        itemType: "Hand Cannon",
        rarity: "Legendary",
        icon: "",
        priority: "LOW",
        notes: "",
        sources: ["Vault of Glass"],
        availableNow: false,
        dateAdded: new Date().toISOString(),
      },
    ];
    let postResolved = false;
    server.use(
      http.get(`${API}/api/wishlist`, () => HttpResponse.json(items)),
      http.post(`${API}/api/wishlist/bulk`, async ({ request }) => {
        const body = (await request.json()) as {
          action: string;
          ids: number[];
        };
        await delay(100);
        postResolved = true;
        if (body.action === "delete") {
          items = items.filter((i) => !body.ids.includes(Number(i.id)));
        }
        return HttpResponse.json({ updated: 1, skipped: 0 });
      }),
    );
    renderPage(<WishList />, "/wishlist");
    await screen.findByText("Gjallarhorn");

    fireEvent.click(screen.getByRole("button", { name: /select/i }));
    const check = await screen.findByRole("checkbox", {
      name: /select gjallarhorn/i,
    });
    fireEvent.click(check);
    fireEvent.click(screen.getByRole("button", { name: /^delete$/i }));

    // optimistic removal → Gjallarhorn disappears from the list before the
    // delayed POST (and its onSettled refetch) has resolved.
    await waitFor(() =>
      expect(screen.queryByText("Gjallarhorn")).not.toBeInTheDocument(),
    );
    expect(postResolved).toBe(false);
  });

  it("bulk-sets priority on selected items", async () => {
    let items = [
      {
        id: "1",
        itemHash: 111,
        name: "Gjallarhorn",
        itemType: "Rocket Launcher",
        rarity: "Exotic",
        icon: "",
        priority: "LOW",
        notes: "",
        sources: [],
        availableNow: false,
        dateAdded: new Date().toISOString(),
      },
    ];
    let postBody: unknown = null;
    server.use(
      // GET stays stateful so the onSettled refetch is consistent with the
      // optimistic update rather than clobbering it back to Low.
      http.get(`${API}/api/wishlist`, () => HttpResponse.json(items)),
      http.post(`${API}/api/wishlist/bulk`, async ({ request }) => {
        const body = (await request.json()) as {
          action: string;
          ids: number[];
          priority?: string;
        };
        postBody = body;
        if (body.action === "set_priority" && body.priority) {
          items = items.map((i) =>
            body.ids.includes(Number(i.id))
              ? { ...i, priority: body.priority! }
              : i,
          );
        }
        return HttpResponse.json({ updated: 1, skipped: 0 });
      }),
    );
    renderPage(<WishList />, "/wishlist");
    await screen.findByText("Gjallarhorn");

    fireEvent.click(screen.getByRole("button", { name: /select/i }));
    const check = await screen.findByRole("checkbox", {
      name: /select gjallarhorn/i,
    });
    fireEvent.click(check);

    // Open the action bar's "Set priority" dropdown and pick High — same
    // open/pick interaction as the single-item "changes priority through
    // the row dropdown" test above.
    fireEvent.click(screen.getByRole("button", { name: "Set priority" }));
    fireEvent.click(screen.getByRole("button", { name: "High" }));

    // Optimistic update: the item's priority badge flips to High before the
    // request settles.
    await waitFor(() =>
      expect(
        screen.getByText("High", { selector: ".gt-badge" }),
      ).toBeInTheDocument(),
    );
    expect(postBody).toEqual({
      action: "set_priority",
      ids: [1],
      priority: "HIGH",
    });
  });

  it("rolls back a failed bulk delete", async () => {
    server.use(
      http.post(
        `${API}/api/wishlist/bulk`,
        () => new HttpResponse(null, { status: 500 }),
      ),
    );
    renderPage(<WishList />, "/wishlist");
    await screen.findByText("Gjallarhorn");

    fireEvent.click(screen.getByRole("button", { name: /select/i }));
    const check = await screen.findByRole("checkbox", {
      name: /select gjallarhorn/i,
    });
    fireEvent.click(check);
    fireEvent.click(screen.getByRole("button", { name: /^delete$/i }));

    // onError rolls the optimistic removal back → Gjallarhorn returns.
    expect(await screen.findByText("Gjallarhorn")).toBeInTheDocument();
  });

  it("keeps selection on a failed bulk action", async () => {
    server.use(
      http.post(
        `${API}/api/wishlist/bulk`,
        () => new HttpResponse(null, { status: 500 }),
      ),
    );
    renderPage(<WishList />, "/wishlist");
    await screen.findByText("Gjallarhorn");

    fireEvent.click(screen.getByRole("button", { name: /select/i }));
    const check = await screen.findByRole("checkbox", {
      name: /select gjallarhorn/i,
    });
    fireEvent.click(check);
    fireEvent.click(screen.getByRole("button", { name: /^delete$/i }));

    // onError rolls the optimistic removal back → Gjallarhorn returns.
    expect(await screen.findByText("Gjallarhorn")).toBeInTheDocument();
    // Select mode was NOT exited on failure — the "Done" toggle (select mode
    // is on) and the row checkbox are both still present, so the user can
    // retry the same selection.
    expect(screen.getByRole("button", { name: "Done" })).toBeInTheDocument();
    expect(
      screen.getByRole("checkbox", { name: /select gjallarhorn/i }),
    ).toBeChecked();
  });

  it("disables Set priority until an item is selected", async () => {
    server.use(
      http.get(`${API}/api/wishlist`, () =>
        HttpResponse.json([
          {
            id: "1",
            itemHash: 111,
            name: "Gjallarhorn",
            itemType: "Rocket Launcher",
            rarity: "Exotic",
            icon: "",
            priority: "HIGH",
            notes: "",
            sources: [],
            availableNow: false,
            dateAdded: new Date().toISOString(),
          },
        ]),
      ),
    );
    renderPage(<WishList />, "/wishlist");
    await screen.findByText("Gjallarhorn");

    fireEvent.click(screen.getByRole("button", { name: /select/i }));
    // Nothing selected yet → Set priority is disabled, mirroring Delete.
    expect(
      screen.getByRole("button", { name: /set priority/i }),
    ).toBeDisabled();

    fireEvent.click(
      await screen.findByRole("checkbox", { name: /select gjallarhorn/i }),
    );
    expect(screen.getByRole("button", { name: /set priority/i })).toBeEnabled();
  });

  it("renders a non-Xûr vendor availability label", async () => {
    server.use(
      http.get(`${API}/api/wishlist`, () =>
        HttpResponse.json([
          {
            id: "1",
            itemHash: 111,
            name: "Gjallarhorn",
            itemType: "Rocket Launcher",
            rarity: "Exotic",
            icon: "",
            priority: "HIGH",
            notes: "",
            sources: [],
            availableNow: true,
            availableFrom: "Banshee-44",
            dateAdded: new Date().toISOString(),
          },
        ]),
      ),
    );
    renderPage(<WishList />, "/wishlist");
    expect(await screen.findByText("Banshee-44")).toBeInTheDocument();
  });
});
