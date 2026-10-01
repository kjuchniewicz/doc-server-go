package main

import (
	"fmt"
	"html"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/gomarkdown/markdown"
	mhtml "github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

// ---------- renderowanie Markdown i szablon strony ----------

// cbRe dopasowuje linie z checkboxami Markdown.
var cbRe = regexp.MustCompile(`^(\s*[-*+] )\[([ xX])\]`)

// renderMD konwertuje Markdown na HTML.
func renderMD(data []byte) string {
	ext := parser.CommonExtensions | parser.AutoHeadingIDs
	p := parser.NewWithExtensions(ext)
	renderer := mhtml.NewRenderer(mhtml.RendererOptions{
		Flags: mhtml.CommonFlags | mhtml.HrefTargetBlank,
	})
	return string(markdown.ToHTML(data, p, renderer))
}

// renderWithCheckboxes zamienia linie checkboxów na <input>.
// Zwraca HTML oraz liczbę linii oryginalnego dokumentu.
func renderWithCheckboxes(data []byte, editable bool) (string, int) {
	lines := strings.Split(string(data), "\n")
	for i, l := range lines {
		if m := cbRe.FindStringSubmatch(l); m != nil {
			dis := " disabled"
			chk := ""
			if editable {
				dis = ""
			}
			if m[2] != " " {
				chk = " checked"
			}
			lines[i] = m[1] + fmt.Sprintf(`<input type="checkbox" class="cb" data-line="%d"%s%s>`, i, chk, dis) + l[len(m[0]):]
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
		userNav = fmt.Sprintf(`<span class="user-name">%s</span><form class="inline" method="post" action="/logout"><button>Wyloguj</button></form>`, html.EscapeString(user))
	} else {
		userNav = `<a class="btn" href="/login">Zaloguj</a>`
	}

	nav := `<nav class="topnav">` + brand + `<div class="nav-links"><a href="/">Start</a><a href="/docs">Dokumenty</a></div><div class="nav-user">` + userNav + `</div></nav>`

	adminLink := ""
	if adminEmail != "" {
		adminLink = `<span>Kontakt: <a href="mailto:` + html.EscapeString(adminEmail) + `">` + html.EscapeString(adminEmail) + `</a></span><span class="sep">|</span>`
	}

	footer := `<footer class="site-footer"><span>` + html.EscapeString(footerText) + `</span><span class="sep">|</span>` + adminLink + `<a href="https://github.com/kjuchniewicz/doc-server-go" target="_blank" rel="noopener">GitHub</a></footer>`

	return `<!doctype html><html lang="pl"><head><meta charset="utf-8"><title>` + html.EscapeString(title) + `</title>
<meta name="viewport" content="width=device-width,initial-scale=1">
<link rel="stylesheet" href="/static/style.css">
</head><body>` + nav + `<main>` + body + `</main>` + footer + `</body></html>`
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
func version(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	fmt.Fprintf(w, `{"v":"%d"}`, contentVersion())
}
