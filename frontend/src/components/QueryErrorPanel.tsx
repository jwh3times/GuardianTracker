import { Button, EmptyState } from "./primitives";
import { errorState } from "../lib/errorState";

const BUNGIE_PRIVACY_URL = "https://www.bungie.net/7/en/User/Account/Privacy";

interface QueryErrorPanelProps {
  /** The `error` from a failed `useQuery`. `undefined` yields the generic copy. */
  error: unknown;
  /** Wire to the query's `refetch`. */
  onRetry: () => void;
}

/**
 * The failure panel for a data fetch: coded copy from `errorState`, the Bungie
 * privacy link when the failure was a privacy restriction, and a Retry button.
 *
 * Every view that loads data from a Bungie-backed endpoint renders this on
 * failure, so a failed fetch never reads as "you have nothing" — which is
 * exactly what WishList, Admin, This Week and header search used to do.
 * Keeping it in one module also keeps the copy and the privacy affordance
 * consistent across pages.
 *
 * Renders a `gt-card`, so it suits a page or panel slot. The header search
 * dropdown deliberately uses its own one-line message instead — a card inside
 * that menu would break its layout.
 */
export function QueryErrorPanel({ error, onRetry }: QueryErrorPanelProps) {
  const es = errorState(error);
  return (
    <div className="gt-card">
      <EmptyState
        icon={es.icon}
        color="var(--c-text-3)"
        title={es.title}
        body={es.body}
        action={
          <div style={{ display: "flex", gap: "var(--s-2)" }}>
            {es.privacyLink && (
              <a
                href={BUNGIE_PRIVACY_URL}
                target="_blank"
                rel="noopener noreferrer"
              >
                <Button variant="outline" sm icon="external">
                  Bungie privacy settings
                </Button>
              </a>
            )}
            <Button variant="outline" sm onClick={onRetry}>
              Retry
            </Button>
          </div>
        }
      />
    </div>
  );
}
