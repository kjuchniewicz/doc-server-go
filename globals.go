package main

import (
	"sync"
	"time"
)

// ---------- zmienne globalne ----------

var (
	// mu chroni wszystkie współdzielone mapy i konfigurację.
	mu sync.RWMutex

	// sessions przechowuje aktywne sesje użytkowników (cookie token -> login).
	sessions = map[string]string{}

	// locks przechowuje aktualne blokady edycji plików.
	locks = map[string]*lock{}

	// cfg to aktualna konfiguracja ładowana z ustawienia.toml.
	cfg Config
)

// lock reprezentuje pojedynczą blokadę edycji pliku.
type lock struct {
	User  string    // użytkownik trzymający blokadę
	Since time.Time // czas rozpoczęcia edycji
	Beat  time.Time // ostatni heartbeat
}
