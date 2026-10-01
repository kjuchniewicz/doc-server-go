package main

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// ---------- narzędzia ----------

// safeDoc weryfikuje, czy relatywna ścieżka wskazuje na plik .md
// znajdujący się wewnątrz folderu dokumenty.
// Zwraca bezpieczną ścieżkę oraz true, gdy ścieżka jest dozwolona.
func safeDoc(rel string) (string, bool) {
	clean := filepath.Clean("/" + rel)
	p := filepath.Join("dokumenty", clean)
	abs, _ := filepath.Abs(p)
	root, _ := filepath.Abs("dokumenty")
	if !strings.HasPrefix(abs, root) || !strings.HasSuffix(strings.ToLower(abs), ".md") {
		return "", false
	}
	return p, true
}

// urlq zwraca bezpiecznie zakodowany parametr URL.
func urlq(s string) string {
	return url.QueryEscape(s)
}

// fileMeta rozbija nazwę pliku na nazwę wyświetlaną i dział odpowiedzialny.
// Dział jest odczytywany z tekstu po ostatnim " - " przed rozszerzeniem.
// Przykład: "01_Montaz - Konstrukcja mechaniczna.md".
func fileMeta(filename string) (displayName, department string) {
	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)
	if i := strings.LastIndex(base, " - "); i >= 0 {
		return base[:i] + ext, strings.TrimSpace(base[i+3:])
	}
	return filename, ""
}

// readFileOrEmpty zwraca zawartość pliku lub pusty ciąg, gdy plik nie istnieje.
func readFileOrEmpty(name string) string {
	data, err := os.ReadFile(name)
	if err != nil {
		return ""
	}
	return string(data)
}
