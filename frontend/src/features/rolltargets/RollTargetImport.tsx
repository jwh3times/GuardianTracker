import { useRef, useState } from "react";
import { Button, Textarea } from "../../components/primitives";
import { useImportRollTargets } from "../../data/rolltargets";
import type {
  RollTargetImportLine,
  RollTargetImportReport,
} from "../../types/design";

/**
 * DIM-format import: a file picker and a paste box, both add-only, both
 * landing on the same `useImportRollTargets` mutation. Renders the resulting
 * report inline — counts first, then every non-imported line grouped by
 * outcome with a plain-language reason, per CONTEXT.md's "DIM import
 * report". Lines whose outcome is "skipped" (comments/headers) are not
 * listed; imported lines are reflected only in the count.
 */

/** Stable iteration order for the counts line, matching how a player reads
 * it: what worked, then why the rest didn't. */
const COUNT_ORDER = [
  "imported",
  "unresolved perk",
  "already saved",
  "unknown weapon",
  "unsupported",
  "malformed",
  "failed",
];

const COUNT_LABEL: Record<string, string> = {
  imported: "imported",
  "unresolved perk": "unresolved",
  "already saved": "already saved",
  "unknown weapon": "unknown weapon",
  unsupported: "unsupported",
  malformed: "malformed",
  failed: "failed",
};

const GROUP_LABEL: Record<string, string> = {
  "unresolved perk": "Unresolved perk",
  "already saved": "Already saved",
  "unknown weapon": "Unknown weapon",
  unsupported: "Unsupported line",
  malformed: "Malformed line",
  failed: "Failed to save",
};

/** Plain-language reason for one non-imported line, using the structured
 * `unresolved.reason` for wording when the line has one. */
function describeLine(line: RollTargetImportLine): string {
  if (line.outcome === "unresolved perk" && line.unresolved) {
    const label = line.unresolved.name
      ? `perk "${line.unresolved.name}"`
      : `perk ${line.unresolved.hash}`;
    switch (line.unresolved.reason) {
      case "not-in-pool":
        return `This weapon can't roll ${label}.`;
      case "ambiguous":
        return `${label} matches more than one perk, so it wasn't guessed.`;
      case "not-a-weapon-perk":
        return `${label} isn't a weapon perk.`;
      default:
        break;
    }
  }
  return line.detail || "This line did not import.";
}

function groupNonImportedLines(
  report: RollTargetImportReport,
): [string, RollTargetImportLine[]][] {
  const groups = new Map<string, RollTargetImportLine[]>();
  for (const line of report.lines) {
    if (line.outcome === "skipped" || line.outcome === "imported") continue;
    const list = groups.get(line.outcome) ?? [];
    list.push(line);
    groups.set(line.outcome, list);
  }
  return Array.from(groups.entries()).sort(
    (a, b) => COUNT_ORDER.indexOf(a[0]) - COUNT_ORDER.indexOf(b[0]),
  );
}

export function RollTargetImport({ onImported }: { onImported?: () => void }) {
  const [text, setText] = useState("");
  const [report, setReport] = useState<RollTargetImportReport | null>(null);
  const [readError, setReadError] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const { importDIM, isPending } = useImportRollTargets({
    onSuccess: (result) => {
      setReport(result);
      onImported?.();
    },
    onError: () => {
      setReadError("Import failed. Check the file and try again.");
    },
  });

  const runImport = (value: string) => {
    if (!value.trim()) return;
    setReadError(null);
    setReport(null);
    importDIM(value);
  };

  const handleFile = async (file: File) => {
    setReadError(null);
    try {
      const content = await file.text();
      setText(content);
      runImport(content);
    } catch {
      setReadError("Could not read that file.");
    }
    if (fileInputRef.current) fileInputRef.current.value = "";
  };

  const stats = report
    ? COUNT_ORDER.filter((outcome) => (report.counts[outcome] ?? 0) > 0)
        .map((outcome) => `${report.counts[outcome]} ${COUNT_LABEL[outcome]}`)
        .join(" · ") || "Nothing to import."
    : "";

  const lineGroups = report ? groupNonImportedLines(report) : [];

  return (
    <section className="gt-rt-import gt-card" aria-labelledby="rt-import-title">
      <h2 id="rt-import-title" className="gt-section-title">
        Import from DIM
      </h2>
      <p className="gt-action-meta">
        Import a DIM-format wish list of roll targets. Import only adds —
        nothing already saved is changed or removed.
      </p>
      <div className="gt-rt-import-controls">
        <label className="gt-rt-file-label">
          <span>Choose a file</span>
          <input
            ref={fileInputRef}
            type="file"
            accept=".txt,.dim,text/plain"
            aria-label="Import a DIM-format file"
            onChange={(e) => {
              const file = e.target.files?.[0];
              if (file) void handleFile(file);
            }}
          />
        </label>
        <Textarea
          value={text}
          onChange={setText}
          rows={3}
          placeholder="…or paste a DIM-format wish list here"
          ariaLabel="Paste a DIM-format wish list"
        />
        <Button
          variant="primary"
          sm
          disabled={!text.trim() || isPending}
          onClick={() => runImport(text)}
        >
          {isPending ? "Importing…" : "Import"}
        </Button>
      </div>

      {readError && (
        <p className="gt-rt-import-error" role="alert">
          {readError}
        </p>
      )}

      {report && (
        <div className="gt-rt-import-report" role="status">
          <p className="gt-rt-import-stats mono">{stats}</p>
          {lineGroups.length > 0 && (
            <div className="gt-rt-import-lines">
              {lineGroups.map(([outcome, lines]) => (
                <div key={outcome} className="gt-rt-import-group">
                  <h3>
                    {GROUP_LABEL[outcome] ?? outcome} ({lines.length})
                  </h3>
                  <ul>
                    {lines.map((line) => (
                      <li key={line.line}>
                        Line {line.line}: {describeLine(line)}
                      </li>
                    ))}
                  </ul>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </section>
  );
}
