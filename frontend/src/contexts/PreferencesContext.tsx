import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
  ReactNode,
} from "react";
import { apiFetch } from "../lib/api";

export type CardStyle = "framed" | "compact";
export type Personalize = "off" | "normal";

interface Preferences {
  /** Item-card density on Collections: full framed cards or condensed rows. */
  cardStyle: CardStyle;
  /** Whether the "for you" personalization badges (Missing / Avail-now) show. */
  personalize: Personalize;
}

interface PreferencesContextType extends Preferences {
  setCardStyle: (v: CardStyle) => void;
  setPersonalize: (v: Personalize) => void;
}

const DEFAULTS: Preferences = { cardStyle: "framed", personalize: "normal" };
const STORAGE_KEY = "guardian_prefs";

const PreferencesContext = createContext<PreferencesContextType | undefined>(
  undefined
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
  const [prefs, setPrefs] = useState<Preferences>(load);

  useEffect(() => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(prefs));
    } catch {
      /* ignore persistence failures */
    }
  }, [prefs]);

  useEffect(() => {
    // On mount, sync from API if authenticated
    const token = localStorage.getItem("guardian_token");
    if (!token) return;
    apiFetch<{ cardStyle: "framed" | "compact"; personalize: boolean }>("/api/preferences")
      .then((remote) => {
        setPrefs({
          cardStyle: remote.cardStyle,
          personalize: remote.personalize ? "normal" : "off",
        });
      })
      .catch(() => {
        // API unavailable — keep localStorage value
      });
  }, []); // once on mount

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

  return (
    <PreferencesContext.Provider
      value={{ ...prefs, setCardStyle, setPersonalize }}
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
