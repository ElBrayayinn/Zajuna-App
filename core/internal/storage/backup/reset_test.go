package backup

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func seedUserData(t *testing.T, dataDir string) {
	t.Helper()
	for _, dir := range []string{"evidences/checklist", "reports", "backups"} {
		if err := os.MkdirAll(filepath.Join(dataDir, filepath.FromSlash(dir)), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	for _, file := range []string{"zajuna.db", "zajuna.db-wal", "config.json", "evidences/checklist/slot-1.png", "backups/old.zip"} {
		if err := os.WriteFile(filepath.Join(dataDir, filepath.FromSlash(file)), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func exists(dataDir, name string) bool {
	_, err := os.Stat(filepath.Join(dataDir, filepath.FromSlash(name)))
	return err == nil
}

func TestApplyPendingResetKeepsBackupsForInAppReset(t *testing.T) {
	dataDir := t.TempDir()
	seedUserData(t, dataDir)
	if err := StageReset(dataDir); err != nil {
		t.Fatal(err)
	}
	applied, err := ApplyPendingReset(dataDir)
	if err != nil || !applied {
		t.Fatalf("ApplyPendingReset = %v, %v", applied, err)
	}
	for _, name := range []string{"zajuna.db", "zajuna.db-wal", "config.json", "evidences", "reports", PendingResetFile} {
		if exists(dataDir, name) {
			t.Fatalf("%s should have been removed", name)
		}
	}
	if !exists(dataDir, "backups/old.zip") {
		t.Fatal("in-app reset must keep backups")
	}
	if applied, _ := ApplyPendingReset(dataDir); applied {
		t.Fatal("second call must be a no-op")
	}
}

func TestApplyPendingResetDiscardsLegacyInstallerMarker(t *testing.T) {
	dataDir := t.TempDir()
	seedUserData(t, dataDir)
	if err := os.WriteFile(filepath.Join(dataDir, PendingResetFile), []byte(LegacyInstallerResetMarker), 0o600); err != nil {
		t.Fatal(err)
	}
	applied, err := ApplyPendingReset(dataDir)
	if applied || !errors.Is(err, ErrLegacyResetDiscarded) {
		t.Fatalf("ApplyPendingReset = %v, %v; want discarded legacy marker", applied, err)
	}
	for _, name := range []string{"zajuna.db", "config.json", "evidences/checklist/slot-1.png", "backups/old.zip"} {
		if !exists(dataDir, name) {
			t.Fatalf("%s must survive an installer-written marker", name)
		}
	}
	if exists(dataDir, PendingResetFile) {
		t.Fatal("the legacy marker must be removed so it is not retried")
	}
}

func TestRecordVersionKeepsDataAcrossVersions(t *testing.T) {
	dataDir := t.TempDir()
	// Data from a release that never wrote the version marker (<= 0.1.2).
	seedUserData(t, dataDir)
	previous, err := RecordVersion(dataDir, "0.1.3")
	if err != nil || previous != "" {
		t.Fatalf("RecordVersion = %q, %v", previous, err)
	}
	if previous, err = RecordVersion(dataDir, "0.1.3"); err != nil || previous != "" {
		t.Fatalf("same version must not report a change: %q, %v", previous, err)
	}
	if previous, err = RecordVersion(dataDir, "0.1.4"); err != nil || previous != "0.1.3" {
		t.Fatalf("upgrade must report the previous version: %q, %v", previous, err)
	}
	for _, name := range []string{"zajuna.db", "zajuna.db-wal", "config.json", "evidences/checklist/slot-1.png", "backups/old.zip"} {
		if !exists(dataDir, name) {
			t.Fatalf("%s must survive a version change", name)
		}
	}
	contents, _ := os.ReadFile(filepath.Join(dataDir, AppVersionFile))
	if string(contents) != "0.1.4\n" {
		t.Fatalf("version marker = %q", contents)
	}
}

func TestRecordVersionIgnoresDevBuilds(t *testing.T) {
	devDir := t.TempDir()
	seedUserData(t, devDir)
	if previous, err := RecordVersion(devDir, "dev"); err != nil || previous != "" || exists(devDir, AppVersionFile) {
		t.Fatalf("development builds must not touch the marker: %q, %v", previous, err)
	}
}
