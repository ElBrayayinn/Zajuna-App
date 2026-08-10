package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mxschmitt/playwright-go"
)

func main() {
	root := os.Getenv("ZAJUNA_PLAYWRIGHT_DIR")
	if root == "" {
		root = filepath.Join("bin", "playwright")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "no se pudo resolver el runtime de Playwright: %v\n", err)
		os.Exit(1)
	}
	driverDir := filepath.Join(root, "driver")
	browsersDir := filepath.Join(root, "browsers")
	if err := os.MkdirAll(root, 0o700); err != nil {
		fmt.Fprintf(os.Stderr, "no se pudo crear el runtime de Playwright: %v\n", err)
		os.Exit(1)
	}
	if err := os.Setenv("PLAYWRIGHT_BROWSERS_PATH", browsersDir); err != nil {
		fmt.Fprintf(os.Stderr, "no se pudo configurar la ruta de Chromium: %v\n", err)
		os.Exit(1)
	}
	if err := playwright.Install(&playwright.RunOptions{
		DriverDirectory: driverDir,
		Browsers:        []string{"chromium"},
		Verbose:         true,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "no se pudo instalar Chromium local: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Runtime Chromium instalado en %s\n", root)
}
