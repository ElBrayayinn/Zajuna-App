package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"context"
	"fmt"
	"github.com/mxschmitt/playwright-go"
	"github.com/zajuna-app/core/internal/capture"
	"github.com/zajuna-app/core/internal/storage/sqlite"
	"github.com/zajuna-app/core/internal/zajuna"
	"net/http/httptest"
)

func TestNVDAScreenReaderPass(t *testing.T) {
	if os.Getenv("ZAJUNA_RUN_NVDA") != "1" {
		t.Skip("NVDA pass disabled")
	}

	repoRoot, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	nvdaExe := os.Getenv("ZAJUNA_NVDA_EXE")
	if nvdaExe == "" {
		nvdaExe = filepath.Join(repoRoot, "tmp", "nvda-portable", "nvda.exe")
	}
	if _, err := os.Stat(nvdaExe); err != nil {
		t.Fatalf("NVDA no está en %s: %v", nvdaExe, err)
	}

	configDir := filepath.Join(repoRoot, "tmp", "nvda-config")
	logPath := filepath.Join(repoRoot, "tmp", "nvda-pass.log")
	_ = os.Remove(logPath)
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}

	_ = exec.Command(nvdaExe, "--quit").Run()
	time.Sleep(800 * time.Millisecond)

	nvda := exec.Command(nvdaExe,
		"--disable-addons",
		"--lang=es",
		"--config-path="+configDir,
		"--log-file="+logPath,
		"--log-level=12",
	)
	nvda.Dir = filepath.Dir(nvdaExe)
	if err := nvda.Start(); err != nil {
		t.Fatalf("no se pudo arrancar NVDA: %v", err)
	}
	defer func() {
		_ = exec.Command(nvdaExe, "--quit").Run()
		if nvda.Process != nil {
			_ = nvda.Process.Kill()
		}
	}()

	deadline := time.Now().Add(25 * time.Second)
	for {
		check := exec.Command(nvdaExe, "--check-running")
		if err := check.Run(); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("NVDA no llegó a estar en ejecución")
		}
		time.Sleep(400 * time.Millisecond)
	}
	time.Sleep(1500 * time.Millisecond)

	dataDir := t.TempDir()
	store, err := sqlite.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := writeConfig(dataDir, appConfig{SetupComplete: true, ZajunaUsername: "qa-user", CredentialsStored: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpsertFichas(context.Background(), []zajuna.Ficha{{ExternalID: "qa-ficha", Name: "Ficha de prueba", CourseID: "qa-course"}}); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(newRouterWithServices(dataDir, &memoryCredentialStore{}, nil, store, nil))
	defer server.Close()

	runtime := capture.Resolve("")
	pw, err := runtime.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer pw.Stop()
	headed := false
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: &headed,
		Args:     []string{"--disable-lcd-text"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer browser.Close()
	browserContext, err := browser.NewContext()
	if err != nil {
		t.Fatal(err)
	}
	defer browserContext.Close()
	page, err := browserContext.NewPage()
	if err != nil {
		t.Fatal(err)
	}
	if err := page.SetViewportSize(1280, 800); err != nil {
		t.Fatal(err)
	}

	routes := []string{
		"/resumen",
		"/fichas",
		"/checklist",
		"/actividades",
		"/evidencias",
		"/trabajos",
		"/reportes",
		"/configuracion",
		"/diagnostico",
	}

	for _, route := range routes {
		if _, err := page.Goto(server.URL + route); err != nil {
			t.Fatalf("navigate %s: %v", route, err)
		}
		if err := page.Locator("#dashboard-main").WaitFor(playwright.LocatorWaitForOptions{State: playwright.WaitForSelectorStateVisible}); err != nil {
			t.Fatalf("%s main: %v", route, err)
		}
		page.WaitForTimeout(600)
		if err := page.Keyboard().Press("Tab"); err != nil {
			t.Fatalf("%s Tab: %v", route, err)
		}
		page.WaitForTimeout(700)
		if err := page.Keyboard().Press("h"); err != nil {
			t.Fatalf("%s heading: %v", route, err)
		}
		page.WaitForTimeout(500)
		t.Logf("NVDA visited %s", route)
	}

	page.WaitForTimeout(1200)
	spoken, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("no se pudo leer el log de NVDA %s: %v", logPath, err)
	}
	text := string(spoken)
	t.Logf("NVDA log bytes=%d path=%s", len(spoken), logPath)

	needles := []string{"Saltar al contenido", "principal", "navegación"}
	found := 0
	lower := strings.ToLower(text)
	for _, needle := range needles {
		if strings.Contains(strings.ToLower(text), strings.ToLower(needle)) || strings.Contains(lower, strings.ToLower(needle)) {
			found++
		}
	}
	if !strings.Contains(text, "Speaking") && !strings.Contains(lower, "hablando") && !strings.Contains(lower, "saltar") {
		t.Fatalf("el log de NVDA no contiene voz útil (%d bytes). Revisar que Chromium headed sea visible para NVDA.\n%s", len(spoken), clip(text, 2500))
	}
	t.Logf("NVDA log bytes=%d needles=%d/%d", len(spoken), found, len(needles))
	fmt.Fprintf(os.Stderr, "NVDA excerpt:\n%s\n", clip(text, 1800))
}

func clip(value string, n int) string {
	if len(value) <= n {
		return value
	}
	return value[len(value)-n:]
}
