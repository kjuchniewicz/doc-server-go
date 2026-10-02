package main

import (
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ---------- obsługa stron ----------

// pageStart renderuje stronę główną: start.md + legenda.md + instrukcja obsługi.
func pageStart(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	var b strings.Builder
	b.WriteString("<h1>Dokumentacja DTR</h1>")

	if data, err := os.ReadFile("start.md"); err == nil {
		htmlStr, _ := renderWithCheckboxes(data, checkboxOpts{editable: canEdit(user)})
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

// toggle przełącza stan checkboxa w pliku start.md (tylko dla użytkowników z [users]).
func toggle(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	if !canEdit(user) {
		http.Error(w, "brak uprawnień do edycji checkboxów", http.StatusForbidden)
		return
	}
	n, _ := strconv.Atoi(r.URL.Query().Get("line"))
	data, err := os.ReadFile("start.md")
	if err != nil {
		http.Error(w, "brak start.md", http.StatusNotFound)
		return
	}
	lines := strings.Split(string(data), "\n")
	if n < 0 || n >= len(lines) || !cbRe.MatchString(lines[n]) {
		http.Error(w, "zła linia", http.StatusBadRequest)
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
	w.WriteHeader(http.StatusNoContent)
}

// pageDocs wyświetla listę dokumentów pogrupowanych według folderów lub działów.
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
			// Nagłówki folderów nadrzędnych, jeśli nie mają własnych plików.
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

// renderDocsTable generuje tabelę dokumentów z kolumną statusu i dodatkową kolumną.
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

// pageView renderuje podgląd pojedynczego dokumentu Markdown.
func pageView(w http.ResponseWriter, r *http.Request) {
	rel := r.URL.Query().Get("f")
	p, ok := safeDoc(rel)
	if !ok {
		http.Error(w, "niedozwolona ścieżka", http.StatusForbidden)
		return
	}
	data, err := os.ReadFile(p)
	if err != nil {
		http.Error(w, "nie ma takiego pliku", http.StatusNotFound)
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
	switch {
	case l != nil && l.User == user:
		btn = `<form class="inline" method="get" action="/edit"><input type="hidden" name="f" value="` + html.EscapeString(rel) + `"><button>Wróć do edycji</button></form>
<form class="inline" method="get" action="/release"><input type="hidden" name="f" value="` + html.EscapeString(rel) + `"><button>Zwolnij blokadę</button></form>`
	case l == nil && user != "":
		btn = `<form class="inline" method="get" action="/edit"><input type="hidden" name="f" value="` + html.EscapeString(rel) + `"><button>Edytuj</button></form>`
	case user == "" && l == nil:
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

// pageEdit wyświetla edytor EasyMDE i zakłada blokadę edycji pliku.
func pageEdit(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	if user == "" {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	rel := r.URL.Query().Get("f")
	p, ok := safeDoc(rel)
	if !ok {
		http.Error(w, "niedozwolona ścieżka", http.StatusForbidden)
		return
	}
	data, err := os.ReadFile(p)
	if err != nil {
		http.Error(w, "nie ma takiego pliku", http.StatusNotFound)
		return
	}

	pruneLocks()
	mu.Lock()
	if l := locks[rel]; l != nil && l.User != user {
		mu.Unlock()
		http.Redirect(w, r, "/view?f="+urlq(rel), http.StatusFound)
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
<form id="editform" method="post" action="/save?f=` + urlq(rel) + `">
<textarea id="rawmd" style="display:none">` + html.EscapeString(string(data)) + `</textarea>
<input type="hidden" name="c" id="mdcontent">
<div id="editor"></div>
<p><button type="submit">Zapisz</button>
<a href="/release?f=` + urlq(rel) + `">Anuluj (zwolnij blokadę)</a></p>
</form>
<link rel="stylesheet" href="/static/toastui/toastui-editor.min.css">
<script src="/static/toastui/toastui-editor-all.min.js"></script>
<script>
const editor = new toastui.Editor({
  el: document.getElementById('editor'),
  initialEditType: 'wysiwyg',
  previewStyle: 'vertical',
  height: '60vh',
  initialValue: document.getElementById('rawmd').value,
  usageStatistics: false,
  hideModeSwitch: true
});
document.getElementById('editform').addEventListener('submit', function(e){
  document.getElementById('mdcontent').value = editor.getMarkdown();
});
setInterval(()=>fetch('/beat?f=` + urlq(rel) + `',{method:'POST'}),30000);
</script>`
	fmt.Fprint(w, page("Edycja "+rel, body, user))
}

// save zapisuje zmiany w pliku i zwalnia blokadę.
func save(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	if user == "" {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	rel := r.URL.Query().Get("f")
	p, ok := safeDoc(rel)
	if !ok {
		http.Error(w, "niedozwolona ścieżka", http.StatusForbidden)
		return
	}

	pruneLocks()
	if l := lockOf(rel); l != nil && l.User != user {
		http.Error(w, "plik edytuje ktoś inny", http.StatusConflict)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "błąd formularza", http.StatusBadRequest)
		return
	}
	if err := os.WriteFile(p, []byte(r.Form.Get("c")), 0644); err != nil {
		http.Error(w, "błąd zapisu", http.StatusInternalServerError)
		return
	}

	mu.Lock()
	delete(locks, rel)
	mu.Unlock()
	log.Println("ZAPIS", user, rel)
	http.Redirect(w, r, "/view?f="+urlq(rel), http.StatusFound)
}
