package main

import (
	"fmt"
	"html"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/gomarkdown/markdown"
	mhtml "github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

var (
	mu       sync.RWMutex
	sessions = map[string]string{} // token -> user
	locks    = map[string]*lock{}  // relPath -> blokada
	cfg      Config
)

type lock struct {
	User  string
	Since time.Time
	Beat  time.Time
}

const lockTTL = 90 * time.Second

func pruneLocks() {
	mu.Lock()
	defer mu.Unlock()
	for p, l := range locks {
		if time.Since(l.Beat) > lockTTL {
			delete(locks, p)
			log.Println("BLOKADA_EXP", l.User, p)
		}
	}
}

func lockOf(rel string) *lock {
	mu.RLock()
	defer mu.RUnlock()
	return locks[rel]
}

type Config struct {
	Port     int
	SiteName string
	Footer   string
	LdapURL  string
	Domain   string
	DomainNT string
	Users    map[string]bool // allowlist: pusta = wszyscy użytkownicy domeny
}

// ---------- konfiguracja (ustawienia.toml, hot reload) ----------

func loadConfig() Config {
	c := Config{
		Port:     8081,
		SiteName: "DTR Żarna",
		Footer:   "Juchniewicz Kamil [k.juchniewicz@zarna.pl]",
		LdapURL:  "ldap://192.168.1.7",
		Domain:   "zdc.ols",
		DomainNT: "ZDC",
		Users:    map[string]bool{},
	}
	data, err := os.ReadFile("ustawienia.toml")
	if err != nil {
		log.Println("brak ustawienia.toml, domyślny port 8081")
		return c
	}
	inUsers := false
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(strings.SplitN(line, "#", 2)[0])
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") {
			inUsers = line == "[users]"
			continue
		}
		kv := strings.SplitN(line, "=", 2)
		if len(kv) != 2 {
			continue
		}
		k := strings.TrimSpace(kv[0])
		v := strings.Trim(strings.TrimSpace(kv[1]), `"'`)
		if inUsers {
			c.Users[k] = true
		} else {
			switch k {
			case "port":
				if p, err := strconv.Atoi(v); err == nil {
					c.Port = p
				}
			case "ldap_url":
				c.LdapURL = v
			case "domain":
				c.Domain = v
			case "domain_nt":
				c.DomainNT = v
			case "site_name":
				c.SiteName = v
			case "footer":
				c.Footer = v
			}
		}
	}
	return c
}

func watchConfig() {
	var last time.Time
	for {
		if st, err := os.Stat("ustawienia.toml"); err == nil && st.ModTime() != last {
			last = st.ModTime()
			mu.Lock()
			cfg = loadConfig()
			mu.Unlock()
			log.Println("przeładowano ustawienia.toml")
		}
		time.Sleep(2 * time.Second)
	}
}

// ---------- pomocnicze ----------

// canEdit: checkboxy może zmieniać tylko użytkownik z sekcji [users] w ustawienia.toml
func canEdit(user string) bool {
	if user == "" {
		return false
	}
	mu.RLock()
	defer mu.RUnlock()
	return cfg.Users[user] || cfg.Users[strings.ToLower(user)]
}

func getUser(r *http.Request) string {
	c, err := r.Cookie("sess")
	if err != nil {
		return ""
	}
	mu.RLock()
	defer mu.RUnlock()
	return sessions[c.Value]
}

func renderMD(data []byte) string {
	ext := parser.CommonExtensions | parser.AutoHeadingIDs
	p := parser.NewWithExtensions(ext)
	renderer := mhtml.NewRenderer(mhtml.RendererOptions{
		Flags: mhtml.CommonFlags | mhtml.HrefTargetBlank,
	})
	return string(markdown.ToHTML(data, p, renderer))
}

var cbRe = regexp.MustCompile(`^(\s*[-*+] )\[([ xX])\]`)

// przetwarza markdown: linie-checkbox zamienia na <input>, reszta normalnie
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

func page(title, body, user string) string {
	mu.RLock()
	siteName := cfg.SiteName
	footerText := cfg.Footer
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
	footer := `<footer class="site-footer"><span>` + html.EscapeString(footerText) + `</span><span class="sep">|</span><a href="https://github.com/kjuchniewicz/doc-server-go" target="_blank" rel="noopener">GitHub</a></footer>`
	return `<!doctype html><html lang="pl"><head><meta charset="utf-8"><title>` + html.EscapeString(title) + `</title>
<meta name="viewport" content="width=device-width,initial-scale=1"><style>
body{font-family:Segoe UI,Arial,sans-serif;max-width:1100px;min-height:100vh;margin:0 auto;padding:0 2rem 1rem;background:#1b1d22;color:#d8dce3;display:flex;flex-direction:column}
a{color:#5fb87a}
.topnav{display:flex;align-items:center;justify-content:space-between;gap:1rem;background:#15171b;border-bottom:2px solid #2d6a4f;padding:.75rem 1.5rem;margin-bottom:1rem}
.brand{font-size:1.25rem;font-weight:600;color:#8fc7ff;text-decoration:none}
.nav-links{display:flex;gap:.5rem}
.nav-links a,.nav-user a{padding:.35rem .8rem;border-radius:4px;text-decoration:none;color:#d8dce3}
.nav-links a:hover,.nav-user a:hover{background:#2a2d33}
.btn{display:inline-block;background:#2d6a4f;color:#fff;padding:.35rem .8rem;border-radius:4px;text-decoration:none}
.btn:hover{background:#367c5c}
.nav-user{display:flex;align-items:center;gap:.75rem}
.user-name{font-weight:600;color:#8fc7ff}
form.inline{display:inline}
table{border-collapse:collapse;width:100%}td,th{border:1px solid #3a3f47;padding:.35rem .55rem}th{background:#26292f}
tr:nth-child(even) td{background:#202329}
button{padding:.4rem 1rem;cursor:pointer;background:#2d6a4f;color:#fff;border:0;border-radius:4px}
input{background:#15171b;color:#d8dce3;border:1px solid #3a3f47;padding:.35rem}
code{background:#2a2d33;padding:0 .2em}h1,h2,h3{color:#8fc7ff}
.group{color:#8fc7ff;margin:1.2rem 0 .4rem;border-bottom:1px solid #333}
.lock{background:#4d3f1a;border:1px solid #8a6d1f;padding:.5rem;border-radius:4px}
.free{background:#1d3a24;border:1px solid #2f6b3a;padding:.5rem;border-radius:4px}
textarea{width:100%;height:60vh;font-family:Consolas,monospace;font-size:14px;background:#15171b;color:#d8dce3;border:1px solid #3a3f47}
.EasyMDEContainer .CodeMirror,.EasyMDEContainer .editor-statusbar,.editor-preview{background:#1b1d22;color:#d8dce3;border-color:#3a3f47}
.CodeMirror-cursor{border-color:#d8dce3}
.editor-toolbar{border:1px solid #3a3f47;border-radius:4px 4px 0 0}
.editor-toolbar button{color:#d8dce3 !important;background:transparent}
.editor-toolbar button:hover,.editor-toolbar button.active{background:#2d3138 !important;border-color:#4a5058 !important;color:#fff !important}
.editor-toolbar i.separator{border-color:#3a3f47 !important}
.CodeMirror-fullscreen,.editor-preview-full,.editor-preview-side.editor-preview-active-side{background:#1b1d22 !important;color:#d8dce3 !important}
.cm-s-default .cm-header{color:#8fc7ff}.cm-s-default .cm-strong{color:#ffd479}
.cm-s-default .cm-em{color:#c792ea}.cm-s-default .cm-link,.cm-s-default .cm-url{color:#5fb87a}
.cm-s-default .cm-quote{color:#9aa3b0}.cm-s-default .cm-comment{color:#7f848e}
.cm-s-default .cm-string{color:#a5d6a7}.cm-s-default .cm-hr{color:#5a6270}
.CodeMirror-selected{background:#3a4250 !important}.editor-statusbar{color:#9aa3b0}
.group.lvl1{font-size:1.1em;color:#7fb0e8}
.group.lvl2{font-size:.95em;color:#6f9fd0}
.docs-table{table-layout:auto}
.docs-table td:first-child{width:100%}
.docs-table td.status,.docs-table td.department{width:1%;white-space:nowrap}
.docs-table th{text-align:left;white-space:nowrap}
.note{color:#9aa3b0;font-size:.9rem}
main{flex:1}
.site-footer{margin-top:2rem;padding:1rem 0;border-top:1px solid #333;color:#9aa3b0;font-size:.85rem;text-align:center}
.site-footer a{color:#7fb0e8}
.site-footer .sep{margin:0 .6rem}
</style></head><body>` + nav + `<main>` + body + `</main>` + footer + `</body></html>`
}

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

// ---------- strona startowa ----------

func pageStart(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	var b strings.Builder
	b.WriteString("<h1>Dokumentacja DTR</h1>")

	if data, err := os.ReadFile("start.md"); err == nil {
		htmlStr, _ := renderWithCheckboxes(data, canEdit(user))
		b.WriteString(htmlStr)
		b.WriteString(`<p class="note">Plik: start.md – edytowany bezpośrednio na dysku, strona odświeża się sama po zmianie.</p>`)
	}
	if data, err := os.ReadFile("legenda.md"); err == nil {
		b.WriteString("<hr>" + renderMD(data))
	}
	b.WriteString(`<hr><h2>Jak korzystać</h2><ol>
<li><b>Przeglądanie</b> – <a href="/docs">Dokumenty</a> listuje pliki .md z folderu <code>dokumenty/</code> pogrupowane wg podfolderów (do 3 poziomów).</li>
<li><b>Checkboxy na tej stronie</b> – może zaznaczać tylko zalogowany użytkownik domenowy wymieniony w sekcji <code>[users]</code> pliku <code>ustawienia.toml</code>. Kliknięcie zapisuje zmianę w <code>start.md</code>.</li>
<li><b>Konfiguracja</b> – <code>ustawienia.toml</code> przeładowuje się automatycznie po zapisie pliku. Zalogować może się każdy użytkownik domeny; <code>[users]</code> określa tylko edytorów checkboxów. Zmiana portu wymaga restartu; przy zajętym porcie serwer próbuje kolejnych (maks. +4).</li>
<li><b>start.md / legenda.md</b> – edytuj bezpośrednio w plikach; strona odświeża się po zmianie (legenda.md jest opcjonalny).</li>
</ol>
<script>
let v='` + strconv.FormatInt(contentVersion(), 10) + `';
setInterval(async()=>{try{const j=await(await fetch('/version',{cache:'no-store'})).json();if(j.v!==v)location.reload();}catch(e){console.error('Sprawdzanie zmian:',e);}},2000);
document.querySelectorAll('input.cb').forEach(c=>c.addEventListener('change',async()=>{c.disabled=true;await fetch('/toggle?line='+c.dataset.line,{method:'POST'});location.reload();}));
</script>`)
	fmt.Fprint(w, page("Start", b.String(), user))
}

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

func version(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	fmt.Fprintf(w, `{"v":"%d"}`, contentVersion())
}

// ---------- checkbox toggle ----------

func toggle(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	if !canEdit(user) {
		http.Error(w, "brak uprawnień do edycji checkboxów", http.StatusForbidden)
		return
	}
	n, _ := strconv.Atoi(r.URL.Query().Get("line"))
	data, err := os.ReadFile("start.md")
	if err != nil {
		http.Error(w, "brak start.md", 404)
		return
	}
	lines := strings.Split(string(data), "\n")
	if n < 0 || n >= len(lines) || !cbRe.MatchString(lines[n]) {
		http.Error(w, "zła linia", 400)
		return
	}
	lines[n] = cbRe.ReplaceAllStringFunc(lines[n], func(s string) string {
		if strings.Contains(s, "[ ]") {
			return strings.Replace(s, "[ ]", "[x]", 1)
		}
		return regexp.MustCompile(`\[[xX]\]`).ReplaceAllString(s, "[ ]")
	})
	os.WriteFile("start.md", []byte(strings.Join(lines, "\n")), 0644)
	log.Println("checkbox zmieniony:", user, "linia", n)
	w.WriteHeader(204)
}

// ---------- dokumenty ----------

func pageDocs(w http.ResponseWriter, r *http.Request) {
	type sec struct {
		name  string
		files []string
	}
	secs := map[string]*sec{}
	var order []string
	var allFiles []string
	filepath.Walk("dokumenty", func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
			return nil
		}
		rel, _ := filepath.Rel("dokumenty", p)
		rel = filepath.ToSlash(rel)
		dir := path.Dir(rel)
		parts := strings.Split(dir, "/")
		if len(parts) > 3 {
			parts = parts[:3]
		}
		section := strings.Join(parts, " / ")
		if dir == "." {
			section = "(bez kategorii)"
		}
		if secs[section] == nil {
			secs[section] = &sec{name: section}
			order = append(order, section)
		}
		secs[section].files = append(secs[section].files, rel)
		allFiles = append(allFiles, rel)
		return nil
	})
	sort.Strings(order)
	var b strings.Builder
	byDepartment := r.URL.Query().Get("sort") == "department"
	b.WriteString("<h1>Dokumenty</h1>")
	if byDepartment {
		b.WriteString(`<p><a href="/docs"><button>Grupuj według folderów</button></a></p>`)
	} else {
		b.WriteString(`<p><a href="/docs?sort=department"><button>Grupuj według działu odpowiedzialnego</button></a></p>`)
	}
	pruneLocks()
	if byDepartment {
		departments := map[string][]string{}
		var names []string
		for _, f := range allFiles {
			_, department := fileMeta(path.Base(f))
			if department == "" {
				department = "Bez przypisanego działu"
			}
			if _, ok := departments[department]; !ok {
				names = append(names, department)
			}
			departments[department] = append(departments[department], f)
		}
		sort.Strings(names)
		for _, department := range names {
			sort.Strings(departments[department])
			b.WriteString(`<h2 class="group lvl0">` + html.EscapeString(department) + `</h2>`)
			b.WriteString(renderDocsTable(departments[department], 0, false))
		}
		fmt.Fprint(w, page("Dokumenty", b.String(), getUser(r)))
		return
	}
	emitted := map[string]bool{}
	for _, s := range order {
		if s == "(bez kategorii)" {
			b.WriteString(`<h2 class="group lvl0">(bez kategorii)</h2>`)
		} else {
			parts := strings.Split(s, " / ")
			// nagłówki folderów nadrzędnych (gdy nie mają własnej sekcji)
			for i := 0; i < len(parts)-1; i++ {
				anc := strings.Join(parts[:i+1], " / ")
				if !emitted[anc] && secs[anc] == nil {
					emitted[anc] = true
					b.WriteString(fmt.Sprintf(`<h2 class="group lvl%d" style="margin-left:%dpx">%s</h2>`,
						i, i*28, html.EscapeString(parts[i])))
				}
			}
			depth := len(parts) - 1
			b.WriteString(fmt.Sprintf(`<h2 class="group lvl%d" style="margin-left:%dpx">%s</h2>`,
				depth, depth*28, html.EscapeString(parts[depth])))
		}
		depth := 0
		if s != "(bez kategorii)" {
			depth = len(strings.Split(s, " / ")) - 1
		}
		indent := depth * 28
		b.WriteString(fmt.Sprintf(`<table class="docs-table" style="margin-left:%dpx;width:calc(100%% - %dpx)"><thead><tr><th>Dokument</th><th>Status</th><th>Dział odpowiedzialny</th></tr></thead><tbody>`, indent, indent))
		for _, f := range secs[s].files {
			st := `<span class="free">wolny</span>`
			if l := lockOf(f); l != nil {
				st = fmt.Sprintf(`<span class="lock">edytuje: <b>%s</b></span>`, html.EscapeString(l.User))
			}
			display, department := fileMeta(path.Base(f))
			b.WriteString(`<tr><td><a href="/view?f=` + urlq(f) + `">` + html.EscapeString(display) + `</a></td><td class="status">` + st + `</td><td class="department">` + html.EscapeString(department) + `</td></tr>`)
		}
		b.WriteString("</tbody>")
		b.WriteString("</table>")
	}
	fmt.Fprint(w, page("Dokumenty", b.String(), getUser(r)))
}

func renderDocsTable(files []string, indent int, showDepartment bool) string {
	thirdHeader := "Folder"
	if showDepartment {
		thirdHeader = "Dział odpowiedzialny"
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf(`<table class="docs-table" style="margin-left:%dpx;width:calc(100%% - %dpx)"><thead><tr><th>Dokument</th><th>Status</th><th>%s</th></tr></thead><tbody>`, indent, indent, thirdHeader))
	for _, f := range files {
		st := `<span class="free">wolny</span>`
		if l := lockOf(f); l != nil {
			st = fmt.Sprintf(`<span class="lock">edytuje: <b>%s</b></span>`, html.EscapeString(l.User))
		}
		display, department := fileMeta(path.Base(f))
		third := department
		if !showDepartment {
			third = path.Dir(f)
		}
		b.WriteString(`<tr><td><a href="/view?f=` + urlq(f) + `">` + html.EscapeString(display) + `</a></td><td class="status">` + st + `</td><td class="department">` + html.EscapeString(third) + `</td></tr>`)
	}
	b.WriteString("</tbody></table>")
	return b.String()
}

// fileMeta odczytuje dział z ostatniego " - " przed rozszerzeniem.
// Np. "01_Montaz - Konstrukcja mechaniczna.md".
func fileMeta(filename string) (displayName, department string) {
	ext := filepath.Ext(filename)
	base := strings.TrimSuffix(filename, ext)
	if i := strings.LastIndex(base, " - "); i >= 0 {
		return base[:i] + ext, strings.TrimSpace(base[i+3:])
	}
	return filename, ""
}

func urlq(s string) string {
	return url.QueryEscape(s)
}

func pageView(w http.ResponseWriter, r *http.Request) {
	rel := r.URL.Query().Get("f")
	p, ok := safeDoc(rel)
	if !ok {
		http.Error(w, "niedozwolona ścieżka", 403)
		return
	}
	data, err := os.ReadFile(p)
	if err != nil {
		http.Error(w, "nie ma takiego pliku", 404)
		return
	}
	user := getUser(r)
	pruneLocks()
	l := lockOf(rel)
	banner := `<p id="lockinfo" class="free"></p>`
	if l != nil {
		banner = `<p id="lockinfo" class="lock"></p>`
	}
	btn := ""
	if l != nil && l.User == user {
		btn = `<form class="inline" method="get" action="/edit"><input type="hidden" name="f" value="` + html.EscapeString(rel) + `"><button>Wróć do edycji</button></form>
<form class="inline" method="get" action="/release"><input type="hidden" name="f" value="` + html.EscapeString(rel) + `"><button>Zwolnij blokadę</button></form>`
	} else if l == nil && user != "" {
		btn = `<form class="inline" method="get" action="/edit"><input type="hidden" name="f" value="` + html.EscapeString(rel) + `"><button>Edytuj</button></form>`
	} else if user == "" && l == nil {
		btn = `<p><a href="/login">Zaloguj się</a>, aby edytować.</p>`
	}
	body := `<h1>` + html.EscapeString(rel) + `</h1>` + banner + btn + `<hr>` + renderMD(data) + `
<script>
async function pingLock(){
  const j=await(await fetch('/lockstate?f=` + urlq(rel) + `')).json();
  const li=document.getElementById('lockinfo');
  if(j.user){li.className='lock';li.innerHTML='Plik aktualnie edytuje <b>'+j.user+'</b> (od '+new Date(j.since).toLocaleTimeString('pl-PL')+').';}
  else{li.className='free';li.textContent='Plik wolny – można edytować.';}
}
pingLock();setInterval(pingLock,10000);
</script>`
	fmt.Fprint(w, page(rel, body, user))
}

// ---------- edycja z blokadą ----------

func lockState(w http.ResponseWriter, r *http.Request) {
	pruneLocks()
	rel := r.URL.Query().Get("f")
	if l := lockOf(rel); l != nil {
		fmt.Fprintf(w, `{"user":%q,"since":%d}`, l.User, l.Since.UnixMilli())
		return
	}
	fmt.Fprint(w, `{}`)
}

func pageEdit(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	if user == "" {
		http.Redirect(w, r, "/login", 302)
		return
	}
	rel := r.URL.Query().Get("f")
	p, ok := safeDoc(rel)
	if !ok {
		http.Error(w, "niedozwolona ścieżka", 403)
		return
	}
	data, err := os.ReadFile(p)
	if err != nil {
		http.Error(w, "nie ma takiego pliku", 404)
		return
	}
	pruneLocks()
	mu.Lock()
	if l := locks[rel]; l != nil && l.User != user {
		mu.Unlock()
		http.Redirect(w, r, "/view?f="+urlq(rel), 302)
		return
	}
	if l := locks[rel]; l != nil {
		l.Beat = time.Now()
	} else {
		locks[rel] = &lock{User: user, Since: time.Now(), Beat: time.Now()}
	}
	mu.Unlock()
	log.Println("EDYCJA", user, rel)
	body := `<h1>Edycja: ` + html.EscapeString(rel) + `</h1>
<link rel="stylesheet" href="/vendor/easymde/easymde.min.css">
<script src="/vendor/easymde/easymde.min.js"></script>
<form method="post" action="/save?f=` + urlq(rel) + `">
<textarea name="c" id="ed">` + html.EscapeString(string(data)) + `</textarea>
<p><button type="submit">Zapisz</button>
<a href="/release?f=` + urlq(rel) + `">Anuluj (zwolnij blokadę)</a></p></form>
<script>
try{new EasyMDE({element:document.getElementById('ed'),spellChecker:false,sideBySideFullscreen:false});}
catch(e){console.error(e);}
setInterval(()=>fetch('/beat?f=` + urlq(rel) + `',{method:'POST'}),30000);
</script>`
	fmt.Fprint(w, page("Edycja "+rel, body, user))
}

func beat(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	rel := r.URL.Query().Get("f")
	mu.Lock()
	if l := locks[rel]; l != nil && l.User == user {
		l.Beat = time.Now()
	}
	mu.Unlock()
	w.WriteHeader(204)
}

func save(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	if user == "" {
		http.Redirect(w, r, "/login", 302)
		return
	}
	rel := r.URL.Query().Get("f")
	p, ok := safeDoc(rel)
	if !ok {
		http.Error(w, "niedozwolona ścieżka", 403)
		return
	}
	pruneLocks()
	if l := lockOf(rel); l != nil && l.User != user {
		http.Error(w, "plik edytuje ktoś inny", 409)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "błąd formularza", 400)
		return
	}
	if err := os.WriteFile(p, []byte(r.Form.Get("c")), 0644); err != nil {
		http.Error(w, "błąd zapisu", 500)
		return
	}
	mu.Lock()
	delete(locks, rel)
	mu.Unlock()
	log.Println("ZAPIS", user, rel)
	http.Redirect(w, r, "/view?f="+urlq(rel), 302)
}

func release(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	rel := r.URL.Query().Get("f")
	mu.Lock()
	if l := locks[rel]; l != nil && l.User == user {
		delete(locks, rel)
		log.Println("BLOKADA_OFF", user, rel)
	}
	mu.Unlock()
	http.Redirect(w, r, "/view?f="+urlq(rel), 302)
}

// ---------- logowanie ----------

func pageLogin(w http.ResponseWriter, r *http.Request) {
	err := ""
	if r.URL.Query().Get("err") != "" {
		err = `<p style="color:#e08080">Błędny login/hasło domenowe, brak kontaktu z kontrolerem domeny lub brak uprawnienia.</p>`
	}
	fmt.Fprint(w, page("Logowanie", `<h1>Logowanie</h1>
<p>Użyj loginu i hasła domenowego.</p>
<form method="post" action="/login"><p>Login: <input name="u" required autofocus></p>
<p>Hasło: <input name="p" type="password" required></p><button>Zaloguj</button></form>`+err, ""))
}

func doLogin(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	u, p := r.Form.Get("u"), r.Form.Get("p")
	mu.RLock()
	allowlist := cfg.Users
	ldapURL, domain, domainNT := cfg.LdapURL, cfg.Domain, cfg.DomainNT
	mu.RUnlock()
	_ = allowlist // [users] = edytorzy checkboxów, nie ogranicza logowania
	if !ldapAuth(ldapURL, domain, domainNT, u, p) {
		log.Println("LOGIN_FAIL", u, r.RemoteAddr)
		http.Redirect(w, r, "/login?err=1", 302)
		return
	}
	tok := fmt.Sprintf("%x", time.Now().UnixNano()) + fmt.Sprintf("%x", os.Getpid())
	mu.Lock()
	sessions[tok] = u
	mu.Unlock()
	log.Println("LOGIN", u, r.RemoteAddr)
	http.SetCookie(w, &http.Cookie{Name: "sess", Value: tok, Path: "/", HttpOnly: true})
	http.Redirect(w, r, "/", 302)
}

// ldapAuth – bind na kontrolerze domeny: najpierw user@domena, potem DOMENA\user
func ldapAuth(ldapURL, domain, domainNT, user, pass string) bool {
	if user == "" || pass == "" || strings.ContainsAny(user, `*()\\/,;`) {
		return false
	}
	l, err := ldap.DialURL(ldapURL, ldap.DialWithDialer(&net.Dialer{Timeout: 5 * time.Second}))
	if err != nil {
		log.Println("LDAP connect error:", err)
		return false
	}
	defer l.Close()
	l.SetTimeout(5 * time.Second)
	if err = l.Bind(fmt.Sprintf("%s@%s", user, domain), pass); err == nil {
		return true
	}
	return l.Bind(fmt.Sprintf(`%s\%s`, domainNT, user), pass) == nil
}

func doLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie("sess"); err == nil {
		mu.Lock()
		delete(sessions, c.Value)
		mu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{Name: "sess", Value: "", Path: "/", MaxAge: -1})
	http.Redirect(w, r, "/", 302)
}

// ---------- main ----------

func main() {
	cfg = loadConfig()
	go watchConfig()
	http.HandleFunc("/", pageStart)
	http.HandleFunc("/version", version)
	http.HandleFunc("/toggle", toggle)
	http.HandleFunc("/docs", pageDocs)
	http.HandleFunc("/view", pageView)
	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			doLogin(w, r)
		} else {
			pageLogin(w, r)
		}
	})
	http.HandleFunc("/logout", doLogout)
	http.HandleFunc("/edit", pageEdit)
	http.HandleFunc("/save", save)
	http.HandleFunc("/release", release)
	http.HandleFunc("/beat", beat)
	http.HandleFunc("/lockstate", lockState)
	http.Handle("/vendor/", http.StripPrefix("/vendor/", http.FileServer(http.Dir("static"))))
	// próbuj port, potem port+1 .. port+4
	for i := 0; i < 5; i++ {
		port := cfg.Port + i
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			log.Printf("port %d zajęty, próbuję %d", port, port+1)
			continue
		}
		log.Printf("doc-server-go na http://0.0.0.0:%d", port)
		log.Fatal(http.Serve(ln, nil))
	}
	log.Fatal("nie znaleziono wolnego portu w zakresie 5 prób")
}
