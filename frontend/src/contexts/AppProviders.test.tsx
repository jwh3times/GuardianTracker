import { describe, it, expect, beforeEach } from "vitest";
import { screen, fireEvent, renderHook } from "@testing-library/react";
import { useLocation, useNavigate } from "react-router";
import { useAuth } from "./AuthContext";
import { usePreferences } from "./PreferencesContext";
import { useFlags } from "./FlagsContext";
import { useCharacters } from "./CharacterContext";
import { useToast } from "../components/Toast";
import { renderWithProviders } from "../test/renderWithProviders";

beforeEach(() => localStorage.clear());

/**
 * The provider contract, asserted in one place for the first time.
 *
 * Before AppProviders existed the tower was rebuilt by hand in sixteen test
 * files across ten orderings, so "is this hook available here?" was answered
 * differently in every file — and `useCharacters` answered it silently.
 */
describe("provider contract", () => {
  it("every context hook throws when used outside its provider", () => {
    const cases: Array<[string, () => unknown]> = [
      ["useAuth", useAuth],
      ["usePreferences", usePreferences],
      ["useFlags", useFlags],
      ["useCharacters", useCharacters],
      ["useToast", useToast],
    ];
    for (const [name, hook] of cases) {
      expect(
        () => renderHook(() => hook()),
        `${name} must throw outside its provider`,
      ).toThrow(new RegExp(`${name} must be used within`));
    }
  });

  it("AppProviders supplies the pre-auth contexts", () => {
    function Probe() {
      useAuth();
      usePreferences();
      useToast();
      return <div>pre-auth ok</div>;
    }
    renderWithProviders(<Probe />, { authed: false });
    expect(screen.getByText("pre-auth ok")).toBeInTheDocument();
  });

  it("AuthedProviders supplies the below-the-gate contexts", () => {
    function Probe() {
      useFlags();
      useCharacters();
      return <div>authed ok</div>;
    }
    renderWithProviders(<Probe />);
    expect(screen.getByText("authed ok")).toBeInTheDocument();
  });

  // The failure this card exists to prevent: a page hoisted above the auth gate
  // silently loses Flags and Character. It must fail loudly, not render empty.
  it("the below-the-gate contexts are absent above the gate", () => {
    function FlagsProbe() {
      useFlags();
      return <div>nope</div>;
    }
    function CharacterProbe() {
      useCharacters();
      return <div>nope</div>;
    }
    expect(() =>
      renderWithProviders(<FlagsProbe />, { authed: false }),
    ).toThrow(/useFlags must be used within/);
    expect(() =>
      renderWithProviders(<CharacterProbe />, { authed: false }),
    ).toThrow(/useCharacters must be used within/);
  });

  it("seeds a signed-in user only when authed", () => {
    renderWithProviders(<div>x</div>, { authed: false });
    expect(localStorage.getItem("guardian_token")).toBeNull();

    localStorage.clear();
    renderWithProviders(<div>y</div>);
    expect(localStorage.getItem("guardian_token")).toBe("test-token");
  });
});

describe("render helper routing", () => {
  function LocationProbe() {
    const loc = useLocation();
    return <div data-testid="loc">{loc.pathname + loc.search}</div>;
  }

  it("starts at the requested route", () => {
    renderWithProviders(<LocationProbe />, { route: "/collections?node=10" });
    expect(screen.getByTestId("loc")).toHaveTextContent("/collections?node=10");
  });

  it("honours initialEntries and initialIndex for history-depth tests", () => {
    function BackButton() {
      const nav = useNavigate();
      return <button onClick={() => nav(-1)}>back</button>;
    }
    renderWithProviders(
      <>
        <LocationProbe />
        <BackButton />
      </>,
      { initialEntries: ["/elsewhere", "/collections"], initialIndex: 1 },
    );
    expect(screen.getByTestId("loc")).toHaveTextContent("/collections");
    fireEvent.click(screen.getByText("back"));
    expect(screen.getByTestId("loc")).toHaveTextContent("/elsewhere");
  });
});
