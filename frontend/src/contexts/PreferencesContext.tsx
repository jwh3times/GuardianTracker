import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
  ReactNode,
} from "react";
import { apiFetch } from "../lib/api";
import { useAuth } from "./AuthContext";

export type CardStyle = "framed" | "compact";
export type Personalize = "off" | "normal";

interface Preferences {
  /** Item-card density on Collections: full framed cards or condensed rows. */
  cardStyle: CardStyle;
  /** Whether the "for you" personalization badges (Missing / Avail-now) show. */
  personalize: Personalize;
}

interface PreferencesContextType extends Preferences {
  /** Undefined when the user's preferences could not be resolved. */
  onboardedAt: string | null | undefined;
  preferencesReady: boolean;
  setCardStyle: (v: CardStyle) => void;
  setPersonalize: (v: Personalize) => void;
  completeOnboarding: () => Promise<void>;
}

const DEFAULTS: Preferences = { cardStyle: "framed", personalize: "normal" };
const STORAGE_KEY = "guardian_prefs";

const PreferencesContext = createContext<PreferencesContextType | undefined>(
  undefined,
);

function load(): Preferences {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return DEFAULTS;
    const parsed = JSON.parse(raw) as Partial<Preferences>;
    return {
      cardStyle: parsed.cardStyle === "compact" ? "compact" : "framed",
      personalize: parsed.personalize === "off" ? "off" : "normal",
    };
  } catch {
    return DEFAULTS;
  }
}

export const PreferencesProvider: React.FC<{ children: ReactNode }> = ({
  children,
}) => {
  const { isAuthenticated, user } = useAuth();
  const [prefs, setPrefs] = useState<Preferences>(load);
  const [onboardedAt, setOnboardedAt] = useState<string | null | undefined>(
    undefined,
  );
  const [syncedMembershipId, setSyncedMembershipId] = useState<string | null>(
    null,
  );

  useEffect(() => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(prefs));
    } catch {
      /* ignore persistence failures */
    }
  }, [prefs]);

  useEffect(() => {
    if (!isAuthenticated || !user?.membershipId) {
      return;
    }

    let cancelled = false;
    const membershipId = user.membershipId;
    apiFetch<{
      cardStyle: "framed" | "compact";
      personalize: boolean;
      onboardedAt: string | null;
    }>("/api/preferences")
      .then((remote) => {
        if (cancelled) return;
        setPrefs({
          cardStyle: remote.cardStyle,
          personalize: remote.personalize ? "normal" : "off",
        });
        setOnboardedAt(remote.onboardedAt);
      })
      .catch(() => {
        // API unavailable — keep localStorage value
      })
      .finally(() => {
        if (!cancelled) setSyncedMembershipId(membershipId);
      });

    return () => {
      cancelled = true;
    };
  }, [isAuthenticated, user?.membershipId]);

  const setCardStyle = useCallback((cardStyle: CardStyle) => {
    setPrefs((p) => {
      const next = { ...p, cardStyle };
      apiFetch("/api/preferences", {
        method: "PUT",
        body: JSON.stringify({ cardStyle }),
      }).catch(() => {}); // silent
      return next;
    });
  }, []);

  const setPersonalize = useCallback((personalize: Personalize) => {
    setPrefs((p) => {
      const next = { ...p, personalize };
      apiFetch("/api/preferences", {
        method: "PUT",
        body: JSON.stringify({ personalize: personalize === "normal" }),
      }).catch(() => {}); // silent
      return next;
    });
  }, []);

  const completeOnboarding = useCallback(async () => {
    const remote = await apiFetch<{
      cardStyle: "framed" | "compact";
      personalize: boolean;
      onboardedAt: string;
    }>("/api/preferences", {
      method: "PUT",
      body: JSON.stringify({ onboardingComplete: true }),
    });
    setOnboardedAt(remote.onboardedAt);
  }, []);

  const preferencesReady =
    !isAuthenticated ||
    (!!user?.membershipId && syncedMembershipId === user.membershipId);

  return (
    <PreferencesContext.Provider
      value={{
        ...prefs,
        onboardedAt,
        preferencesReady,
        setCardStyle,
        setPersonalize,
        completeOnboarding,
      }}
    >
      {children}
    </PreferencesContext.Provider>
  );
};

export function usePreferences(): PreferencesContextType {
  const ctx = useContext(PreferencesContext);
  if (!ctx) {
    throw new Error("usePreferences must be used within a PreferencesProvider");
  }
  return ctx;
}
