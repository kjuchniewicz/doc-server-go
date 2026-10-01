package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
)

// ---------- uwierzytelnianie i sesje ----------

// getUser zwraca login zalogowanego użytkownika na podstawie cookie sess.
func getUser(r *http.Request) string {
	c, err := r.Cookie("sess")
	if err != nil {
		return ""
	}
	mu.RLock()
	defer mu.RUnlock()
	return sessions[c.Value]
}

// pageLogin wyświetla formularz logowania.
func pageLogin(w http.ResponseWriter, r *http.Request) {
	err := ""
	if r.URL.Query().Get("err") != "" {
		err = `<p style="color:#e08080">Błędny login/hasło domenowe, brak kontaktu z kontrolerem domeny lub brak uprawnienia.</p>`
	}
	body := `<h1>Logowanie</h1>
<p>Użyj loginu i hasła domenowego.</p>
<form method="post" action="/login"><p>Login: <input name="u" required autofocus></p>
<p>Hasło: <input name="p" type="password" required></p><button>Zaloguj</button></form>` + err
	fmt.Fprint(w, page("Logowanie", body, ""))
}

// doLogin obsługuje wysłanie formularza logowania.
func doLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/login?err=1", http.StatusFound)
		return
	}
	u, p := r.Form.Get("u"), r.Form.Get("p")

	mu.RLock()
	ldapURL, domain, domainNT := cfg.LdapURL, cfg.Domain, cfg.DomainNT
	mu.RUnlock()

	if !ldapAuth(ldapURL, domain, domainNT, u, p) {
		log.Println("LOGIN_FAIL", u, r.RemoteAddr)
		http.Redirect(w, r, "/login?err=1", http.StatusFound)
		return
	}

	tok := fmt.Sprintf("%x", time.Now().UnixNano()) + fmt.Sprintf("%x", os.Getpid())
	mu.Lock()
	sessions[tok] = u
	mu.Unlock()
	log.Println("LOGIN", u, r.RemoteAddr)

	http.SetCookie(w, &http.Cookie{Name: "sess", Value: tok, Path: "/", HttpOnly: true})
	http.Redirect(w, r, "/", http.StatusFound)
}

// ldapAuth próbuje zalogować użytkownika przez LDAP.
// Najpierw user@domena, a w razie potrzeby DOMENA\user.
func ldapAuth(ldapURL, domain, domainNT, user, pass string) bool {
	if user == "" || pass == "" || strings.ContainsAny(user, `*()\/,;`) {
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

// loginHandler rozdziela metodę GET (formularz) i POST (autoryzacja).
func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		doLogin(w, r)
		return
	}
	pageLogin(w, r)
}

// doLogout usuwa sesję i czyści cookie.
func doLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie("sess"); err == nil {
		mu.Lock()
		delete(sessions, c.Value)
		mu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{Name: "sess", Value: "", Path: "/", MaxAge: -1})
	http.Redirect(w, r, "/", http.StatusFound)
}
