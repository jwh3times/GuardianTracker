import { useEffect, useRef, useState } from "react";
import { useVirtualizer } from "@tanstack/react-virtual";
import type { GTItem } from "../../types/design";
import { CosmeticTile } from "./CosmeticTile";

// TILE_PX must match `.gt-cosmetic-tile { width }` in kit.css.
const TILE_PX = 88;
const GAP = 12;
const ROW_HEIGHT = 124; // tile (88) + caption + gap; rows are uniform height.

export function CosmeticsGrid({
  items,
  onOpen,
  id,
  role,
  "aria-labelledby": ariaLabelledBy,
}: {
  items: GTItem[];
  onOpen: (item: GTItem) => void;
  id?: string;
  role?: string;
  "aria-labelledby"?: string;
}) {
  const parentRef = useRef<HTMLDivElement>(null);
  const [cols, setCols] = useState(1);

  useEffect(() => {
    const el = parentRef.current;
    if (!el) return;
    const measure = () =>
      setCols(
        Math.max(1, Math.floor((el.clientWidth + GAP) / (TILE_PX + GAP))),
      );
    measure();
    const ro = new ResizeObserver(measure);
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

  const rowCount = Math.ceil(items.length / cols);
  const rows = useVirtualizer({
    count: rowCount,
    getScrollElement: () => parentRef.current,
    estimateSize: () => ROW_HEIGHT,
    overscan: 4,
  });

  return (
    <div
      ref={parentRef}
      className="gt-tilegrid-scroll"
      data-testid="cosmetics-grid"
      id={id}
      role={role}
      aria-labelledby={ariaLabelledBy}
    >
      <div style={{ height: rows.getTotalSize(), position: "relative" }}>
        {rows.getVirtualItems().map((row) => {
          const start = row.index * cols;
          return (
            <div
              key={row.key}
              className="gt-tilegrid-row"
              style={{
                position: "absolute",
                top: 0,
                left: 0,
                width: "100%",
                height: ROW_HEIGHT,
                transform: `translateY(${row.start}px)`,
                display: "grid",
                gridTemplateColumns: `repeat(${cols}, ${TILE_PX}px)`,
                justifyContent: "center",
                gap: GAP,
              }}
            >
              {items.slice(start, start + cols).map((it) => (
                <CosmeticTile key={it.id} item={it} onOpen={onOpen} />
              ))}
            </div>
          );
        })}
      </div>
    </div>
  );
}
