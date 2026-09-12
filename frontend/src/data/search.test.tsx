import { useState } from "react";
import { describe, it, expect } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { http, HttpResponse } from "msw";
import { server, API } from "../test/testServer";
import { renderWithProviders } from "../test/renderWithProviders";
import { useItemSearch } from "./search";

/**
 * Contract tests for the Search data-access module (ADR 0020) — key identity
 * per term, the minimum-length gate, the transport's encoding and limit,
 * projection to the domain `SearchResult`, and the thirty-second reuse window.
 */

function SearchProbe({ term, id }: { term: string; id: string }) {
  const { results, isLoading, isError } = useItemSearch(term);
  return (
    <div data-testid={id}>
      {isLoading
        ? "loading"
        : isError
          ? "failed"
          : results
              .map((r) => `${r.id}|${r.name}|${r.icon}|${r.type}|${r.rarity}`)
              .join(",") || "empty"}
    </div>
  );
}

function hit(hash: number, name: string, rarity = "Exotic") {
  return { hash, name, icon: `/${hash}.png`, type: "Hand Cannon", rarity };
}

/** Records every search request's query string and echoes the term back. */
function recordSearches() {
  const sent: { q: string | null; limit: string | null }[] = [];
  server.use(
    http.get(`${API}/api/items/search`, ({ request }) => {
      const params = new URL(request.url).searchParams;
      const q = params.get("q");
      sent.push({ q, limit: params.get("limit") });
      return HttpResponse.json([hit(q?.length ?? 0, `Result for ${q}`)]);
    }),
  );
  return sent;
}

describe("search query identity", () => {
  it("shares one request per term and keys each term separately", async () => {
    const sent = recordSearches();

    renderWithProviders(
      <>
        <SearchProbe term="hawk" id="a" />
        <SearchProbe term="hawk" id="b" />
        <SearchProbe term="gjall" id="c" />
      </>,
    );

    await waitFor(() =>
      expect(screen.getByTestId("c")).toHaveTextContent("Result for gjall"),
    );
    await waitFor(() =>
      expect(screen.getByTestId("a")).toHaveTextContent("Result for hawk"),
    );
    expect(screen.getByTestId("b")).toHaveTextContent("Result for hawk");
    expect(sent.map((s) => s.q).sort()).toEqual(["gjall", "hawk"]);
  });

  it("encodes the term and asks for twenty results", async () => {
    const sent = recordSearches();

    renderWithProviders(<SearchProbe term="a&b c" id="a" />);

    await waitFor(() => expect(sent).toEqual([{ q: "a&b c", limit: "20" }]));
  });

  it("sends nothing for a term shorter than two characters", async () => {
    const sent = recordSearches();

    renderWithProviders(
      <>
        <SearchProbe term="" id="empty" />
        <SearchProbe term="h" id="one" />
        <SearchProbe term="ha" id="two" />
      </>,
    );

    // A gated query is pending but not loading, so the rendered text cannot
    // carry this — the request log is what discriminates. Waiting on the
    // two-character probe proves every probe had its chance to fire.
    await waitFor(() =>
      expect(screen.getByTestId("two")).toHaveTextContent("Result for ha"),
    );
    expect(sent.map((s) => s.q)).toEqual(["ha"]);
  });

  it("reuses a recent term's results rather than asking again", async () => {
    const sent = recordSearches();

    function Toggle() {
      const [open, setOpen] = useState(true);
      return (
        <>
          <button onClick={() => setOpen((v) => !v)}>toggle</button>
          {open && <SearchProbe term="hawk" id="a" />}
        </>
      );
    }

    renderWithProviders(<Toggle />);
    await waitFor(() =>
      expect(screen.getByTestId("a")).toHaveTextContent("Result for hawk"),
    );

    await userEvent.click(screen.getByText("toggle"));
    await userEvent.click(screen.getByText("toggle"));

    expect(screen.getByTestId("a")).toHaveTextContent("Result for hawk");
    expect(sent).toHaveLength(1);
  });
});

describe("projection", () => {
  it("projects wire results to domain results", async () => {
    server.use(
      http.get(`${API}/api/items/search`, () =>
        HttpResponse.json([
          hit(555, "Gjallarhorn", "Exotic"),
          hit(556, "Fatebringer", "Legendary"),
          hit(557, "Oddity", "Mythic"),
        ]),
      ),
    );

    renderWithProviders(<SearchProbe term="gjall" id="a" />);

    await waitFor(() =>
      expect(screen.getByTestId("a")).toHaveTextContent(
        [
          "555|Gjallarhorn|/555.png|Hand Cannon|exotic",
          "556|Fatebringer|/556.png|Hand Cannon|legendary",
          "557|Oddity|/557.png|Hand Cannon|legendary",
        ].join(","),
      ),
    );
  });

  it("surfaces a failed search as an error", async () => {
    server.use(
      http.get(`${API}/api/items/search`, () =>
        HttpResponse.json({ error: "boom" }, { status: 500 }),
      ),
    );

    renderWithProviders(<SearchProbe term="gjall" id="a" />);

    await waitFor(() =>
      expect(screen.getByTestId("a")).toHaveTextContent("failed"),
    );
  });
});
