import React from "react";
import { render, type RenderResult } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { QueryClient } from "@tanstack/react-query";
import { AppProviders, AuthedProviders } from "../contexts/AppProviders";
import { seedBrowserSession } from "./browserSession";

export interface RenderWithProvidersOptions {
  /** Sugar for `initialEntries: [route]`. Ignored when initialEntries is given. */
  route?: string;
  /** Use when a test needs history depth — e.g. asserting a REPLACE vs a PUSH. */
  initialEntries?: string[];
  initialIndex?: number;
  /**
   * Mount the below-the-gate providers (Flags, Character) and seed a signed-in
   * user in localStorage. Defaults to true: every page in the app renders under
   * `ProtectedLayout`, so the honest default is the authenticated tower. Pass
   * false for pages that render above the gate, like Login.
   */
  authed?: boolean;
  /** Override the QueryClient. The default disables retries so failures surface. */
  client?: QueryClient;
}

/**
 * Render a page under the same context tower the app uses.
 *
 * Before this existed, sixteen test files hand-rolled the tower in ten distinct
 * orderings, and most omitted `FlagsProvider` and/or `CharacterProvider`. That
 * drift is invisible until a page adds a `useFlag` or `useCharacters` call, at
 * which point unrelated test files start failing — or worse, don't.
 *
 * Pass a `<Routes>` element as `ui` when a test needs real route matching; it is
 * just children, so nothing extra is required here.
 */
export function renderWithProviders(
  ui: React.ReactNode,
  {
    route = "/",
    initialEntries,
    initialIndex,
    authed = true,
    client,
  }: RenderWithProvidersOptions = {},
): RenderResult {
  if (authed) {
    seedBrowserSession();
  }

  const queryClient =
    client ??
    new QueryClient({
      defaultOptions: {
        queries: { retry: false },
        mutations: { retry: false },
      },
    });

  const entries = initialEntries ?? [route];
  const body = authed ? <AuthedProviders>{ui}</AuthedProviders> : ui;

  return render(
    <MemoryRouter initialEntries={entries} initialIndex={initialIndex}>
      <AppProviders client={queryClient}>{body}</AppProviders>
    </MemoryRouter>,
  );
}
