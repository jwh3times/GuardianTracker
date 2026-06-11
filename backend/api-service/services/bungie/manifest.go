package bungie

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
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
	log.Println("Checking for manifest updates...")
	manifest, err := m.client.GetManifest(ctx)
	if err != nil {
		return fmt.Errorf("failed to get manifest metadata: %w", err)
	}
	newVersion := manifest.Response.Version
	dbURL := manifest.Response.MobileWorldContentPaths.En
	if dbURL == "" {
		return fmt.Errorf("no English manifest database URL found")
	}
	fullURL := "https://www.bungie.net" + dbURL
	log.Printf("Downloading manifest version %s from %s", newVersion, fullURL)
	zipData, err := m.client.DownloadFile(ctx, fullURL)
	if err != nil {
		return fmt.Errorf("failed to download manifest: %w", err)
	}
	log.Printf("Downloaded %d bytes, extracting...", len(zipData))
	if err := m.extractManifest(zipData); err != nil {
		return fmt.Errorf("failed to extract manifest: %w", err)
	}
	m.mu.Lock()
	m.currentVersion = newVersion
	m.lastCheck = time.Now()
	m.mu.Unlock()
	if err := os.WriteFile(m.versionPath, []byte(newVersion), 0644); err != nil {
		log.Printf("Warning: failed to save version file: %v", err)
	}
	log.Printf("Manifest version %s installed successfully", newVersion)
	return nil
}

func (m *ManifestService) extractManifest(zipData []byte) error {
	zipReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return fmt.Errorf("failed to read zip: %w", err)
	}
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
	if err := os.Rename(tmpPath, m.dbPath); err != nil {
		os.Remove(tmpPath)
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
				log.Printf("Warning: failed to check for manifest update: %v", err)
				return nil
			}
			if needsUpdate {
				if err := m.Download(ctx); err != nil {
					log.Printf("Warning: failed to download manifest update: %v", err)
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
					log.Printf("Background manifest check failed: %v", err)
					continue
				}
				if needsUpdate {
					log.Printf("New manifest version available: %s", newVersion)
					if err := m.Download(ctx); err != nil {
						log.Printf("Background manifest download failed: %v", err)
					}
				}
			}
		}
	}()
}
