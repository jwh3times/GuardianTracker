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

const alwaysCurrent = () => true;
export const PreferencesProvider: React.FC<{
  children: ReactNode;
  resetLocal?: boolean;
  isCurrent?: () => boolean;
}> = ({ children, resetLocal = false, isCurrent = alwaysCurrent }) => {
  const { isAuthenticated, user } = useAuth();
  const [prefs, setPrefs] = useState<Preferences>(() =>
    resetLocal ? DEFAULTS : load(),
  );
  const [onboardedAt, setOnboardedAt] = useState<string | null | undefined>(
    undefined,
  );
  const [syncedMembershipId, setSyncedMembershipId] = useState<string | null>(
    null,
  );

  useEffect(() => {
    if (!isCurrent()) return;
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(prefs));
    } catch {
      /* ignore persistence failures */
    }
  }, [prefs, isCurrent]);

  useEffect(() => {
    if (!isAuthenticated || !user?.membershipId) {
      return;
    }

    let cancelled = false;
    const membershipId = JSON.stringify([
      user.membershipType,
      user.membershipId,
    ]);
    apiFetch<{
      cardStyle: "framed" | "compact";
      personalize: boolean;
      onboardedAt: string | null;
    }>("/api/preferences")
      .then((remote) => {
        if (cancelled || !isCurrent()) return;
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
        if (!cancelled && isCurrent()) setSyncedMembershipId(membershipId);
      });

    return () => {
      cancelled = true;
    };
  }, [isAuthenticated, user?.membershipId, user?.membershipType, isCurrent]);

  const setCardStyle = useCallback(
    (cardStyle: CardStyle) => {
      if (!isCurrent()) return;
      setPrefs((p) => ({ ...p, cardStyle }));
      apiFetch("/api/preferences", {
        method: "PUT",
        body: JSON.stringify({ cardStyle }),
      }).catch(() => {});
    },
    [isCurrent],
  );

  const setPersonalize = useCallback(
    (personalize: Personalize) => {
      if (!isCurrent()) return;
      setPrefs((p) => ({ ...p, personalize }));
      apiFetch("/api/preferences", {
        method: "PUT",
        body: JSON.stringify({ personalize: personalize === "normal" }),
      }).catch(() => {});
    },
    [isCurrent],
  );

  const completeOnboarding = useCallback(async () => {
    if (!isCurrent()) return;
    const remote = await apiFetch<{
      cardStyle: "framed" | "compact";
      personalize: boolean;
      onboardedAt: string;
    }>("/api/preferences", {
      method: "PUT",
      body: JSON.stringify({ onboardingComplete: true }),
    });
    if (isCurrent()) setOnboardedAt(remote.onboardedAt);
  }, [isCurrent]);

  const preferencesReady =
    !isAuthenticated ||
    (!!user?.membershipId &&
      syncedMembershipId ===
        JSON.stringify([user.membershipType, user.membershipId]));

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
