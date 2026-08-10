package workers

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func fileSHA256(path string) (string, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("leer artefacto %s: %w", path, err)
	}
	hash := sha256.Sum256(contents)
	return hex.EncodeToString(hash[:]), nil
}

func artifactID(kind, fichaID, itemCode, sha string) string {
	hash := sha256.Sum256([]byte(strings.Join([]string{kind, fichaID, itemCode, sha}, "|")))
	return kind + "-" + hex.EncodeToString(hash[:8])
}

func safeArtifactPath(root, requested, fallback string) (string, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolver carpeta de artefactos: %w", err)
	}
	path := requested
	if path == "" {
		path = filepath.Join(root, fallback)
	} else if !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolver ruta de artefacto: %w", err)
	}
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("la ruta del artefacto debe permanecer dentro de la carpeta local")
	}
	if filepath.Ext(path) == "" {
		path += filepath.Ext(fallback)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", fmt.Errorf("crear carpeta de artefacto: %w", err)
	}
	// Resolve the existing parent after creating it so a pre-existing symlink
	// cannot redirect an artifact outside the configured local root.
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("resolver carpeta real de artefactos: %w", err)
	}
	realParent, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil {
		return "", fmt.Errorf("resolver carpeta real de salida: %w", err)
	}
	relative, err = filepath.Rel(realRoot, realParent)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", errors.New("la ruta del artefacto no puede atravesar un enlace simbólico fuera de la carpeta local")
	}
	return path, nil
}
