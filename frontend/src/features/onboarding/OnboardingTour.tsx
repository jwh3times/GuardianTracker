import { useEffect, useMemo, useReducer, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useLocation, useNavigate } from "react-router-dom";
import { Button } from "../../components/primitives";
import { Icon } from "../../components/Icon";
import { useAuth } from "../../contexts/AuthContext";
import { usePreferences } from "../../contexts/PreferencesContext";
import { collectionsQuery } from "../../lib/queries";
import { initialTourState, TOUR_ROUTES, tourReducer } from "./tourState";

const STEPS = [
  {
    title: "Your Guardian at a glance",
    body: "Dashboard turns your account-wide collection into a practical starting point for today.",
    target: "dashboard",
  },
  {
    title: "Plan before reset",
    body: "This Week ranks live activities and vendor signals, and qualifies data that Bungie does not expose reliably.",
    target: "this-week",
  },
  {
    title: "Find the gaps",
    body: "Collections shows what you own, what is missing, and which tracked items are available right now.",
    target: "collections",
  },
] as const;

export function OnboardingTour() {
  const { user } = useAuth();
  const { onboardedAt, preferencesReady, completeOnboarding } =
    usePreferences();
  const [state, dispatch] = useReducer(tourReducer, initialTourState);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();

  const query = collectionsQuery(
    user?.membershipType,
    user?.membershipId,
    false,
  );
  const { data: collections } = useQuery({
    ...query,
    enabled: query.enabled && preferencesReady && onboardedAt === null,
  });

  const progress = useMemo(() => {
    const categories = collections
      ? Object.values(collections.summary).filter(
          (category) => category.total > category.collected,
        )
      : [];
    return {
      categories: categories.length,
      missing: categories.reduce(
        (total, category) => total + category.total - category.collected,
        0,
      ),
    };
  }, [collections]);

  useEffect(() => {
    if (state.phase !== "tour") return;
    const route = TOUR_ROUTES[state.step];
    if (location.pathname !== route) navigate(route);
  }, [location.pathname, navigate, state]);

  useEffect(() => {
    if (state.phase !== "tour") return;
    const selector = `[data-onboarding-target="${STEPS[state.step].target}"]`;
    let active: Element | null = null;
    const activate = () => {
      const next = document.querySelector(selector);
      if (next === active) return;
      active?.classList.remove("gt-tour-target-active");
      active = next;
      active?.classList.add("gt-tour-target-active");
    };
    activate();
    const observer = new MutationObserver(activate);
    observer.observe(document.body, { childList: true, subtree: true });
    return () => {
      observer.disconnect();
      active?.classList.remove("gt-tour-target-active");
    };
  }, [state]);

  if (!preferencesReady || onboardedAt !== null) return null;

  const finish = async () => {
    setSaving(true);
    setSaveError(false);
    try {
      await completeOnboarding();
      dispatch({ type: "skip" });
    } catch {
      setSaveError(true);
    } finally {
      setSaving(false);
    }
  };

  if (state.phase === "complete") return null;

  if (state.phase === "welcome") {
    return (
      <div className="gt-tour-scrim" role="presentation">
        <section
          className="gt-tour-card gt-tour-welcome"
          role="dialog"
          aria-modal="true"
          aria-labelledby="gt-tour-title"
        >
          <Icon name="sparkle" size="2rem" />
          <p className="gt-eyebrow mono">Welcome to Guardian Tracker</p>
          <h2 id="gt-tour-title">Your collection, with a plan</h2>
          <p>
            Guardian Tracker reads your Bungie profile and manifest data. It
            never changes your inventory or loadouts, and live data is always
            shown with its freshness or reset window.
          </p>
          {progress.missing > 0 && (
            <p className="gt-tour-progress mono">
              Your first snapshot found {progress.missing.toLocaleString()} gaps
              across {progress.categories} collection categories.
            </p>
          )}
          {saveError && (
            <p className="gt-tour-error" role="alert">
              We could not save that choice. Please try again.
            </p>
          )}
          <div className="gt-tour-actions">
            <Button
              variant="ghost"
              onClick={() => void finish()}
              disabled={saving}
            >
              Skip tour
            </Button>
            <Button
              variant="primary"
              autoFocus
              onClick={() => dispatch({ type: "start" })}
            >
              Show me around
            </Button>
          </div>
        </section>
      </div>
    );
  }

  const step = STEPS[state.step];
  const last = state.step === STEPS.length - 1;
  return (
    <div className="gt-tour-layer">
      <section
        className="gt-tour-card gt-tour-step"
        role="dialog"
        aria-modal="true"
        aria-labelledby="gt-tour-step-title"
      >
        <span className="gt-eyebrow mono">
          Step {state.step + 1} of {STEPS.length}
        </span>
        <h2 id="gt-tour-step-title">{step.title}</h2>
        <p>{step.body}</p>
        {saveError && (
          <p className="gt-tour-error" role="alert">
            We could not save completion. Please try again.
          </p>
        )}
        <div className="gt-tour-actions">
          <Button
            variant="ghost"
            onClick={() => void finish()}
            disabled={saving}
          >
            Skip
          </Button>
          <Button
            variant="primary"
            autoFocus
            disabled={saving}
            onClick={() => {
              if (last) void finish();
              else dispatch({ type: "next" });
            }}
          >
            {last ? "Finish" : "Next"}
          </Button>
        </div>
      </section>
    </div>
  );
}
