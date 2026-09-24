package backup

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PendingResetFile is the marker that requests a clean local workspace on
// the next start. Only the explicit in-app reset ("Restablecer datos")
// writes it; installs and updates never do.
const PendingResetFile = ".reset-pending"

// resetTargets are the user-data entries removed by a reset. backups/ is
// never removed: it is the only way back after a reset.
var resetTargets = []string{
	"zajuna.db", "zajuna.db-wal", "zajuna.db-shm",
	"config.json", "evidences", "reports", "exports",
	pendingRestoreDir, appliedRestoreFile,
	// Copies left by a restore that was applied but never committed.
	"zajuna.db.restore-old", "config.json.restore-old", "evidences.restore-old",
	"reports.restore-old", "exports.restore-old",
}

// StageReset records that the local data must be wiped on the next start.
func StageReset(dataDir string) error {
	if strings.TrimSpace(dataDir) == "" {
		return errors.New("la carpeta de datos es obligatoria")
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return fmt.Errorf("prepare data dir for reset: %w", err)
	}
	stamp := time.Now().UTC().Format(time.RFC3339) + "\n"
	if err := os.WriteFile(filepath.Join(dataDir, PendingResetFile), []byte(stamp), 0o600); err != nil {
		return fmt.Errorf("stage reset: %w", err)
	}
	return nil
}

// ResetPending reports whether a reset is staged.
func ResetPending(dataDir string) bool {
	_, err := os.Stat(filepath.Join(dataDir, PendingResetFile))
	return err == nil
}

// LegacyInstallerResetMarker is the marker content that installers up to
// 0.1.3 wrote on every install. It was not requested by the user, so it is
// discarded instead of wiping the workspace.
const LegacyInstallerResetMarker = "full"

// AppVersionFile records which app version last opened the local data.
const AppVersionFile = ".app-version"

// RecordVersion stores the running version next to the data and returns the
// previous one when it changed. Data from other versions is kept: schema
// changes are handled by the SQLite migrations. Development builds ("dev" or
// empty) never touch the marker.
func RecordVersion(dataDir, version string) (string, error) {
	version = strings.TrimSpace(version)
	if version == "" || version == "dev" {
		return "", nil
	}
	markerPath := filepath.Join(dataDir, AppVersionFile)
	contents, err := os.ReadFile(markerPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("read app version marker: %w", err)
	}
	previous := strings.TrimSpace(string(contents))
	if previous == version {
		return "", nil
	}
	if err := os.WriteFile(markerPath, []byte(version+"\n"), 0o600); err != nil {
		return previous, fmt.Errorf("write app version marker: %w", err)
	}
	return previous, nil
}

// wipe removes the user data. The database is mandatory: if it cannot be
// removed the error is returned so the caller keeps the marker and retries.
func wipe(dataDir string) error {
	var failures []string
	for _, name := range resetTargets {
		if err := removeWithRetry(filepath.Join(dataDir, name)); err != nil {
			if strings.HasPrefix(name, "zajuna.db") {
				return fmt.Errorf("no se pudo borrar la base local para restablecer los datos: %w", err)
			}
			failures = append(failures, name)
		}
	}
	if len(failures) > 0 {
		// The database is gone, so the workspace is already clean for the UI;
		// orphaned files are collected by the evidence GC after migration.
		return &partialWipeError{names: failures}
	}
	return nil
}

type partialWipeError struct{ names []string }

func (e *partialWipeError) Error() string {
	return "datos restablecidos, pero quedaron archivos en uso: " + strings.Join(e.names, ", ")
}

// ErrLegacyResetDiscarded reports that an installer-written marker was
// removed without touching the data.
var ErrLegacyResetDiscarded = errors.New("se descartó la orden de borrado automático de un instalador anterior; los datos se conservan")

// ApplyPendingReset wipes the staged user data before SQLite is opened. It
// returns false when no reset is pending. If the database cannot be removed
// (for example, a leftover process still holds it on Windows) the marker is
// kept so the next start retries, and the caller keeps using the old data.
func ApplyPendingReset(dataDir string) (bool, error) {
	if !ResetPending(dataDir) {
		return false, nil
	}
	markerPath := filepath.Join(dataDir, PendingResetFile)
	contents, _ := os.ReadFile(markerPath)
	if strings.TrimSpace(string(contents)) == LegacyInstallerResetMarker {
		if err := os.Remove(markerPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return false, fmt.Errorf("remove legacy reset marker: %w", err)
		}
		return false, ErrLegacyResetDiscarded
	}
	wipeErr := wipe(dataDir)
	var partial *partialWipeError
	if wipeErr != nil && !errors.As(wipeErr, &partial) {
		return false, wipeErr
	}
	if err := os.Remove(markerPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return true, fmt.Errorf("remove reset marker: %w", err)
	}
	return true, wipeErr
}

func removeWithRetry(target string) error {
	var err error
	for attempt := 0; attempt < 10; attempt++ {
		err = os.RemoveAll(target)
		if err == nil {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return err
}
