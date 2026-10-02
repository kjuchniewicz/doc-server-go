package main

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	gmhtml "github.com/yuin/goldmark/renderer/html"
)

// ---------- renderowanie Markdown i szablon strony ----------

// cbRe dopasowuje linie z checkboxami Markdown.
var cbRe = regexp.MustCompile(`^(\s*[-*+] )\[([ xX])\]`)

// mdEngine to skonfigurowany parser/renderer goldmark.
// Ze względu na to, że wstrzykujemy do źródła <input>, włączamy renderowanie
// surowego HTML (WithUnsafe) oraz rozszerzenia: tabele, przekreślenia, autolinki.
var mdEngine = goldmark.New(
	goldmark.WithExtensions(
		extension.Table,
		extension.Strikethrough,
		extension.Linkify,
		extension.TaskList,
	),
	goldmark.WithRendererOptions(gmhtml.WithUnsafe()),
)

// renderMD konwertuje Markdown na HTML przy użyciu goldmark.
func renderMD(data []byte) string {
	var buf bytes.Buffer
	if err := mdEngine.Convert(data, &buf); err != nil {
		return html.EscapeString(err.Error())
	}
	return buf.String()
}

// checkboxOpts steruje tym, czy checkboxy mają być klikalne.
type checkboxOpts struct {
	editable bool
}

// renderWithCheckboxes zamienia linie checkboxów Markdown na <input>.
// Zwraca HTML oraz liczbę linii oryginalnego dokumentu.
func renderWithCheckboxes(data []byte, opts checkboxOpts) (string, int) {
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if m := cbRe.FindStringSubmatch(line); m != nil {
			chk := ""
			if m[2] != " " {
				chk = " checked"
			}
			dis := " disabled"
			if opts.editable {
				dis = ""
			}
			input := fmt.Sprintf(
				`<input type="checkbox" class="cb" data-line="%d"%s%s>`,
				i, chk, dis,
			)
			lines[i] = m[1] + input + line[len(m[0]):]
		}
	}
	return renderMD([]byte(strings.Join(lines, "\n"))), len(lines)
}

// page generuje pełną stronę HTML: menu górne, treść i stopka.
func page(title, body, user string) string {
	mu.RLock()
	siteName := cfg.SiteName
	footerText := cfg.Footer
	adminEmail := cfg.AdminEmail
	mu.RUnlock()

	if siteName == "" {
		siteName = "DTR Żarna"
	}

	brand := `<a class="brand" href="/">` + html.EscapeString(siteName) + `</a>`

	var userNav string
	if user != "" {
		logoutBtn := `<form class="inline" method="post" action="/logout">` +
			`<button>Wyloguj</button></form>`
		userNav = fmt.Sprintf(
			`<span class="user-name">%s</span>%s`,
			html.EscapeString(user), logoutBtn,
		)
	} else {
		userNav = `<a class="btn" href="/login">Zaloguj</a>`
	}

	links := `<a href="/docs">Dokumenty</a>`
	nav := `<nav class="topnav">` + brand +
		`<div class="nav-links">` + links + `</div>` +
		`<div class="nav-user">` + userNav + `</div></nav>`

	adminLink := ""
	if adminEmail != "" {
		escaped := html.EscapeString(adminEmail)
		adminLink = `<span>Kontakt: <a href="mailto:` + escaped + `">` + escaped + `</a></span>`
		adminLink += `<span class="sep">|</span>`
	}

	escapedFooter := html.EscapeString(footerText)
	ghLink := `<a href="https://github.com/kjuchniewicz/doc-server-go"` +
		` target="_blank" rel="noopener">GitHub</a>`
	footer := `<footer class="site-footer"><span>` + escapedFooter + `</span>` +
		`<span class="sep">|</span>` + adminLink + ghLink + `</footer>`

	head := `<head><meta charset="utf-8"><title>` + html.EscapeString(title) + `</title>
<meta name="viewport" content="width=device-width,initial-scale=1">
<link rel="stylesheet" href="/static/style.css">
</head>`

	return `<!doctype html><html lang="pl">` + head +
		`<body>` + nav + `<main>` + body + `</main>` +
		footer + `</body></html>`
}

// contentVersion zwraca sumaryczną „wersję” plików start.md i legenda.md
// na potrzeby automatycznego odświeżania strony głównej.
func contentVersion() int64 {
	var v int64
	for _, f := range []string{"start.md", "legenda.md"} {
		if st, err := os.Stat(f); err == nil {
			v += st.ModTime().UnixNano()
			v += st.Size()
		}
	}
	return v
}

// version obsługuje endpoint /version używany przez stronę startowej do pollingu.
func version(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = fmt.Fprintf(w, `{"v":"%d"}`, contentVersion())
}

// previewRender obsługuje endpoint /preview – renderuje przesłany Markdown na serwerze.
func previewRender(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "wymagana metoda POST", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "błąd odczytu", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprint(w, renderMD(body))
}
