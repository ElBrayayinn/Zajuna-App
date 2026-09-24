package backup

import (
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

func TestApplyPendingResetFullMarkerRemovesBackups(t *testing.T) {
	dataDir := t.TempDir()
	seedUserData(t, dataDir)
	if err := os.WriteFile(filepath.Join(dataDir, PendingResetFile), []byte(FullResetMarker), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyPendingReset(dataDir); err != nil {
		t.Fatal(err)
	}
	if exists(dataDir, "backups") || exists(dataDir, "zajuna.db") {
		t.Fatal("installer reset must remove everything, backups included")
	}
}

func TestEnforceVersionKeepsDataAcrossVersions(t *testing.T) {
	dataDir := t.TempDir()
	seedUserData(t, dataDir)
	wiped, err := EnforceVersion(dataDir, "0.1.3")
	if err != nil || wiped {
		t.Fatalf("EnforceVersion = %v, %v", wiped, err)
	}
	if !exists(dataDir, "zajuna.db") || !exists(dataDir, "evidences") || !exists(dataDir, "backups") {
		t.Fatal("upgrades must keep local data")
	}
	if wiped, err := EnforceVersion(dataDir, "0.1.3"); err != nil || wiped {
		t.Fatalf("same version must not wipe: %v, %v", wiped, err)
	}
	if wiped, err := EnforceVersion(dataDir, "0.1.4"); err != nil || wiped {
		t.Fatalf("new version must not wipe: %v, %v", wiped, err)
	}
	if !exists(dataDir, "zajuna.db") {
		t.Fatal("data of the previous version must be kept")
	}
}

func TestEnforceVersionFreshInstallAndDevBuilds(t *testing.T) {
	dataDir := t.TempDir()
	if wiped, err := EnforceVersion(dataDir, "0.1.3"); err != nil || wiped {
		t.Fatalf("empty dir must not report a wipe: %v, %v", wiped, err)
	}
	if !exists(dataDir, AppVersionFile) {
		t.Fatal("version marker must be written on first start")
	}
	devDir := t.TempDir()
	seedUserData(t, devDir)
	if wiped, _ := EnforceVersion(devDir, "dev"); wiped || !exists(devDir, "zajuna.db") {
		t.Fatal("development builds must never wipe")
	}
}
