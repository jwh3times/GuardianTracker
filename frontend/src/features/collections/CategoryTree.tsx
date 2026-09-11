import React, { useEffect, useState } from "react";
import type { ReactNode } from "react";
import { Icon } from "../../components/Icon";
import type { TreeNode } from "../../types/design";

type CSS = React.CSSProperties & Record<`--${string}`, string | number>;

export function CategoryTree({
  nodes,
  activeId,
  onSelect,
  expand,
}: {
  nodes: TreeNode[];
  activeId: string;
  onSelect: (id: string) => void;
  expand?: string[];
}) {
  const [open, setOpen] = useState<Set<string>>(() => new Set());
  const toggle = (id: string) =>
    setOpen((s) => {
      const n = new Set(s);
      if (n.has(id)) n.delete(id);
      else n.add(id);
      return n;
    });

  // Controlled-seed reveal: when a deep-link resolves an ancestor path, merge
  // those ids into the open set so the selected node is visible. Existing
  // manually-opened ids are preserved. The `expand` prop is the external system
  // being synchronized into local open state here, so the in-effect setState is
  // intentional.
  useEffect(() => {
    if (!expand || expand.length === 0) return;
    setOpen((s) => {
      const n = new Set(s);
      for (const id of expand) n.add(id);
      return n;
    });
  }, [expand]);

  const renderNode = (node: TreeNode, depth: number): ReactNode => {
    const hasChildren = !!node.children && node.children.length > 0;
    const isOpen = open.has(node.id);
    return (
      <div
        key={node.id}
        className="gt-tree-group"
        role="treeitem"
        aria-expanded={hasChildren ? isOpen : undefined}
      >
        <div
          className="gt-tree-row gt-tree-row--parent"
          data-active={activeId === node.id}
          style={{ paddingLeft: `calc(${depth} * var(--s-3))` }}
        >
          {hasChildren ? (
            <button
              className="gt-tree-caret"
              onClick={() => toggle(node.id)}
              aria-label={`${isOpen ? "Collapse" : "Expand"} ${node.label}`}
              data-open={isOpen}
            >
              <Icon name="chevron" size="0.8rem" />
            </button>
          ) : (
            <span
              className="gt-tree-caret gt-tree-caret--leaf"
              aria-hidden="true"
            />
          )}
          <button className="gt-tree-main" onClick={() => onSelect(node.id)}>
            <span className="gt-tree-label">{node.label}</span>
            <span className="gt-tree-pct mono">
              {node.count[0]}/{node.count[1]}
            </span>
          </button>
        </div>
        <div
          className="gt-tree-bar"
          style={{ paddingLeft: `calc(${depth} * var(--s-3))` }}
        >
          <div
            className="gt-tree-bar-fill"
            style={{ "--val": node.pct + "%" } as CSS}
          />
        </div>
        {hasChildren && isOpen && (
          <div className="gt-tree-children" role="group">
            {node.children!.map((c) => renderNode(c, depth + 1))}
          </div>
        )}
      </div>
    );
  };

  return (
    <nav className="gt-tree" role="tree">
      {nodes.map((n) => renderNode(n, 0))}
    </nav>
  );
}
