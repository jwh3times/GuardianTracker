package bungie

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"guardian-tracker/api-service/observability"
)

// ManifestService handles downloading and updating the Bungie manifest database.
type ManifestService struct {
	client        *Client
	dbPath        string
	versionPath   string
	checkInterval time.Duration

	mu             sync.RWMutex
	currentVersion string
	lastCheck      time.Time

	hooksMu      sync.Mutex
	participants []SwapParticipant
	observers    []ManifestObserver
}

// SwapParticipant is implemented by a module that holds an OS-level handle on
// the manifest database file. Every such module MUST register, or the swap's
// os.Rename runs under its open handle: the rename fails outright on Windows,
// and on Linux the module keeps reading a deleted inode.
//
// Implementations may block in CloseForSwap to drain in-flight work — that is
// the point of the call — but must return promptly. There is deliberately no
// timeout: proceeding to the rename with a handle still open is the bug this
// interface exists to prevent, so a slow participant must be fixed at the
// participant.
//
// Reopen means "reopen against whatever manifest is now live", not "a new
// version was installed". It runs on the rollback path too, where the rename
// failed and the old file is still in place. Anything that should happen only
// when the version actually changed belongs in ManifestObserver.
type SwapParticipant interface {
	CloseForSwap()
	Reopen() error
}

// ManifestObserver is implemented by a module holding manifest-derived state —
// cached definitions, a derived index, a resolved tree. OnVersionChanged fires
// only when a new manifest was actually installed, so implementations can
// invalidate and rebuild unconditionally without checking for a rollback.
//
// It runs after every SwapParticipant has reopened, so an implementation may
// query the manifest. Rebuilds that are slow should be launched asynchronously
// by the implementation itself; the notifier calls observers synchronously and
// does not wait on anything they spawn.
type ManifestObserver interface {
	OnVersionChanged(version string) error
}

// NewManifestService creates a new manifest service.
func NewManifestService(client *Client, dbPath string, checkInterval time.Duration) *ManifestService {
	versionPath := strings.TrimSuffix(dbPath, filepath.Ext(dbPath)) + "_version.txt"
	ms := &ManifestService{
		client:        client,
		dbPath:        dbPath,
		versionPath:   versionPath,
		checkInterval: checkInterval,
	}
	if data, err := os.ReadFile(versionPath); err == nil {
		ms.currentVersion = strings.TrimSpace(string(data))
	}
	return ms
}

// RegisterParticipant enrolls a module in the manifest file swap. Participants
// are closed in registration order before the rename and reopened in the same
// order after it, so register manifest.Provider first: observers may query the
// manifest, and lazily-rebuilding consumers reach it through the Provider.
//
// Must be called before the first download can fire.
func (m *ManifestService) RegisterParticipant(p SwapParticipant) {
	if p == nil {
		return
	}
	m.hooksMu.Lock()
	defer m.hooksMu.Unlock()
	m.participants = append(m.participants, p)
}

// RegisterObserver enrolls a module to be told when a new manifest version has
// been installed. Must be called before the first download can fire.
func (m *ManifestService) RegisterObserver(o ManifestObserver) {
	if o == nil {
		return
	}
	m.hooksMu.Lock()
	defer m.hooksMu.Unlock()
	m.observers = append(m.observers, o)
}

func (m *ManifestService) snapshotParticipants() []SwapParticipant {
	m.hooksMu.Lock()
	defer m.hooksMu.Unlock()
	return append([]SwapParticipant{}, m.participants...)
}

// closeParticipants releases every open handle on the manifest file. Runs
// immediately before os.Rename.
func (m *ManifestService) closeParticipants() {
	for _, p := range m.snapshotParticipants() {
		p.CloseForSwap()
	}
}

// reopenParticipants reconnects every participant to whatever manifest file is
// now live. Runs on both the success and the rollback path — a participant left
// closed would serve ErrNotReady forever, so a failure here is logged and the
// loop continues rather than stranding the participants behind it.
func (m *ManifestService) reopenParticipants() {
	for _, p := range m.snapshotParticipants() {
		if err := p.Reopen(); err != nil {
			slog.Warn("manifest swap participant reopen failed",
				slog.String("participant", fmt.Sprintf("%T", p)),
				observability.Err(err),
			)
		}
	}
}

// notifyObservers announces a genuinely new manifest version. Never runs on the
// rollback path. A failing observer is logged and the rest still run.
func (m *ManifestService) notifyObservers(version string) {
	m.hooksMu.Lock()
	observers := append([]ManifestObserver{}, m.observers...)
	m.hooksMu.Unlock()
	for _, o := range observers {
		if err := o.OnVersionChanged(version); err != nil {
			slog.Warn("manifest version-change observer failed",
				slog.String("observer", fmt.Sprintf("%T", o)),
				slog.String("manifest_version", version),
				observability.Err(err),
			)
		}
	}
}

func (m *ManifestService) GetCurrentVersion() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentVersion
}

// Version returns the currently installed manifest version string.
// Returns empty string if no manifest has been downloaded yet.
func (m *ManifestService) Version() string {
	return m.GetCurrentVersion()
}

func (m *ManifestService) GetDBPath() string { return m.dbPath }

func (m *ManifestService) IsReady() bool {
	_, err := os.Stat(m.dbPath)
	return err == nil
}

func (m *ManifestService) CheckForUpdate(ctx context.Context) (bool, string, error) {
	manifest, err := m.client.GetManifest(ctx)
	if err != nil {
		return false, "", fmt.Errorf("failed to get manifest metadata: %w", err)
	}
	newVersion := manifest.Response.Version
	m.mu.RLock()
	current := m.currentVersion
	m.mu.RUnlock()
	return current == "" || current != newVersion, newVersion, nil
}

func (m *ManifestService) Download(ctx context.Context) error {
	logger := observability.Logger(ctx)
	logger.InfoContext(ctx, "checking for manifest updates")
	manifest, err := m.client.GetManifest(ctx)
	if err != nil {
		return fmt.Errorf("failed to get manifest metadata: %w", err)
	}
	newVersion := manifest.Response.Version
	dbURL := manifest.Response.MobileWorldContentPaths.En
	if dbURL == "" {
		return fmt.Errorf("no English manifest database URL found")
	}
	fullURL := m.client.cdnBaseURL + dbURL
	logger.LogAttrs(ctx, slog.LevelInfo, "downloading manifest",
		slog.String("manifest_version", newVersion),
	)
	zipPath := m.dbPath + ".zip.tmp"
	if err := m.client.DownloadFileToPath(ctx, fullURL, zipPath); err != nil {
		return fmt.Errorf("failed to download manifest: %w", err)
	}
	defer os.Remove(zipPath)
	if err := m.extractManifest(zipPath); err != nil {
		return fmt.Errorf("failed to extract manifest: %w", err)
	}
	m.mu.Lock()
	m.currentVersion = newVersion
	m.lastCheck = time.Now()
	m.mu.Unlock()
	if err := os.WriteFile(m.versionPath, []byte(newVersion), 0644); err != nil {
		logger.LogAttrs(ctx, slog.LevelWarn, "manifest version file save failed",
			slog.String("manifest_version", newVersion),
			observability.Err(err),
		)
	}
	logger.LogAttrs(ctx, slog.LevelInfo, "manifest installed",
		slog.String("manifest_version", newVersion),
	)
	// Reopen first so observers can query the new manifest, then announce it.
	m.reopenParticipants()
	m.notifyObservers(newVersion)
	return nil
}

func (m *ManifestService) extractManifest(zipPath string) error {
	zipReader, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to read zip: %w", err)
	}
	defer zipReader.Close()
	var dbFile *zip.File
	for _, f := range zipReader.File {
		if strings.HasSuffix(f.Name, ".content") {
			dbFile = f
			break
		}
	}
	if dbFile == nil {
		return fmt.Errorf("no .content file found in manifest zip")
	}
	rc, err := dbFile.Open()
	if err != nil {
		return fmt.Errorf("failed to open file in zip: %w", err)
	}
	defer rc.Close()
	if err := os.MkdirAll(filepath.Dir(m.dbPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	tmpPath := m.dbPath + ".tmp"
	outFile, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	_, err = io.Copy(outFile, rc)
	outFile.Close()
	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to write database: %w", err)
	}
	// Close open handles before replacing the file: on Windows the rename fails
	// under open handles; on Linux readers would keep the deleted inode.
	m.closeParticipants()
	if err := os.Rename(tmpPath, m.dbPath); err != nil {
		os.Remove(tmpPath)
		// Rollback: participants are closed but the old database is still in
		// place, so reopen against it and keep serving the previous version.
		// Observers are deliberately NOT notified — the version did not change,
		// and invalidating caches or rebuilding indexes here would be pure waste.
		m.reopenParticipants()
		return fmt.Errorf("failed to move database: %w", err)
	}
	return nil
}

// EnsureReady downloads the manifest if it doesn't exist, or updates it if stale.
func (m *ManifestService) EnsureReady(ctx context.Context) error {
	if m.IsReady() {
		m.mu.RLock()
		shouldCheck := time.Since(m.lastCheck) > m.checkInterval
		m.mu.RUnlock()
		if shouldCheck {
			needsUpdate, _, err := m.CheckForUpdate(ctx)
			if err != nil {
				observability.Logger(ctx).LogAttrs(ctx, slog.LevelWarn, "manifest update check failed; keeping installed manifest",
					observability.Err(err),
				)
				return nil
			}
			if needsUpdate {
				if err := m.Download(ctx); err != nil {
					observability.Logger(ctx).LogAttrs(ctx, slog.LevelWarn, "manifest update download failed; keeping installed manifest",
						slog.String("installed_manifest_version", m.GetCurrentVersion()),
						observability.Err(err),
					)
					return nil
				}
			}
		}
		return nil
	}
	return m.Download(ctx)
}

// StartBackgroundUpdater starts a goroutine that periodically checks for manifest updates.
func (m *ManifestService) StartBackgroundUpdater(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(m.checkInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				needsUpdate, newVersion, err := m.CheckForUpdate(ctx)
				if err != nil {
					observability.Logger(ctx).LogAttrs(ctx, slog.LevelWarn, "background manifest update check failed",
						observability.Err(err),
					)
					continue
				}
				if needsUpdate {
					observability.Logger(ctx).LogAttrs(ctx, slog.LevelInfo, "new manifest version available",
						slog.String("manifest_version", newVersion),
					)
					if err := m.Download(ctx); err != nil {
						observability.Logger(ctx).LogAttrs(ctx, slog.LevelWarn, "background manifest download failed",
							slog.String("manifest_version", newVersion),
							observability.Err(err),
						)
					}
				}
			}
		}
	}()
}
