package main

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

// ---------- konfiguracja ----------

// Config przechowuje ustawienia serwera ładowane z ustawienia.toml.
// Plik jest przeładowywany na gorąco; zmiana portu wymaga restartu.
type Config struct {
	Port       int
	SiteName   string
	Footer     string
	AdminEmail string
	LdapURL    string
	Domain     string
	DomainNT   string
	Users      map[string]bool // allowlist: pusta = wszyscy użytkownicy domeny
}

// loadConfig wczytuje ustawienia z pliku ustawienia.toml.
// Jeśli pliku brakuje, zwraca konfigurację domyślną.
func loadConfig() Config {
	c := Config{
		Port:       8081,
		SiteName:   "DTR Żarna",
		Footer:     "Juchniewicz Kamil",
		AdminEmail: "k.juchniewicz@zarna.pl",
		LdapURL:    "ldap://192.168.1.7",
		Domain:     "zdc.ols",
		DomainNT:   "ZDC",
		Users:      map[string]bool{},
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
			continue
		}

		switch k {
		case "port":
			if p, err := strconv.Atoi(v); err == nil {
				c.Port = p
			}
		case "site_name":
			c.SiteName = v
		case "footer":
			c.Footer = v
		case "admin_email":
			c.AdminEmail = v
		case "ldap_url":
			c.LdapURL = v
		case "domain":
			c.Domain = v
		case "domain_nt":
			c.DomainNT = v
		}
	}
	return c
}

// watchConfig obserwuje plik ustawienia.toml i przeładowuje go co 2 sekundy.
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

// canEdit zwraca true, gdy użytkownik znajduje się w sekcji [users].
// Tylko tacy użytkownicy mogą przełączać checkboxy na stronie startowej.
func canEdit(user string) bool {
	if user == "" {
		return false
	}
	mu.RLock()
	defer mu.RUnlock()
	return cfg.Users[user] || cfg.Users[strings.ToLower(user)]
}
