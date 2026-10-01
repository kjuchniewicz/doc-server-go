package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

// ---------- blokady edycji ----------

const lockTTL = 90 * time.Second

// pruneLocks usuwa przeterminowane blokady (brak heartbeatu dłużej niż lockTTL).
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

// lockOf zwraca aktywną blokadę dla podanej ścieżki lub nil.
func lockOf(rel string) *lock {
	mu.RLock()
	defer mu.RUnlock()
	return locks[rel]
}

// lockState obsługuje endpoint /lockstate zwracający JSON z informacją o blokadzie.
func lockState(w http.ResponseWriter, r *http.Request) {
	pruneLocks()
	rel := r.URL.Query().Get("f")
	if l := lockOf(rel); l != nil {
		fmt.Fprintf(w, `{"user":%q,"since":%d}`, l.User, l.Since.UnixMilli())
		return
	}
	fmt.Fprint(w, `{}`)
}

// beat obsługuje heartbeat edytora, przedłużając blokadę co ~30s.
func beat(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	rel := r.URL.Query().Get("f")
	mu.Lock()
	if l := locks[rel]; l != nil && l.User == user {
		l.Beat = time.Now()
	}
	mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

// release obsługuje zwolnienie blokady bez zapisywania zmian.
func release(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	rel := r.URL.Query().Get("f")
	mu.Lock()
	if l := locks[rel]; l != nil && l.User == user {
		delete(locks, rel)
		log.Println("BLOKADA_OFF", user, rel)
	}
	mu.Unlock()
	http.Redirect(w, r, "/view?f="+urlq(rel), http.StatusFound)
}
