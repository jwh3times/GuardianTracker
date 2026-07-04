import { useNavigate } from "react-router-dom";
import { Panel } from "../../components/composite";
import { Icon } from "../../components/Icon";

const STEPS = [
  {
    icon: "collections",
    title: "See what you're missing",
    body: "Collections compares your Bungie profile against every obtainable item.",
    path: "/collections",
    cta: "Open Collections",
  },
  {
    icon: "week",
    title: "Plan your week",
    body: "This Week ranks the best things to do before reset.",
    path: "/this-week",
    cta: "Open This Week",
  },
  {
    icon: "wishlist",
    title: "Track your chase",
    body: "Wishlist an item and we'll flag it when it's available.",
    path: "/wishlist",
    cta: "Open Wishlist",
  },
] as const;

/** Dismissible first-run panel (ROADMAP §2 onboarding v1). */
export function GetStartedPanel({ onDismiss }: { onDismiss: () => void }) {
  const navigate = useNavigate();
  return (
    <Panel
      title="New here? Start with these"
      icon="sparkle"
      accent="var(--c-signal)"
      right={
        <button className="gt-link" onClick={onDismiss}>
          Got it — hide this
        </button>
      }
    >
      <div className="gt-getstarted">
        {STEPS.map((s) => (
          <button
            key={s.path}
            className="gt-getstarted-step"
            onClick={() => navigate(s.path)}
          >
            <Icon
              name={s.icon}
              size="1.2rem"
              style={{ color: "var(--c-signal)" }}
            />
            <span className="gt-getstarted-main">
              <span className="gt-item-name">{s.title}</span>
              <span className="gt-item-type">{s.body}</span>
            </span>
            <span className="gt-link">
              {s.cta} <Icon name="chevron" size="0.8rem" />
            </span>
          </button>
        ))}
      </div>
    </Panel>
  );
}
