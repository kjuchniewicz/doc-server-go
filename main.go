package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
)

// ---------- punkt wejścia ----------

func main() {
	cfg = loadConfig()
	go watchConfig()

	http.HandleFunc("/", pageStart)
	http.HandleFunc("/version", version)
	http.HandleFunc("/toggle", toggle)
	http.HandleFunc("/docs", pageDocs)
	http.HandleFunc("/view", pageView)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/logout", doLogout)
	http.HandleFunc("/edit", pageEdit)
	http.HandleFunc("/save", save)
	http.HandleFunc("/release", release)
	http.HandleFunc("/beat", beat)
	http.HandleFunc("/lockstate", lockState)
	http.HandleFunc("/preview", previewRender)

	// Zasoby statyczne: style.css.
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Próbujemy port z konfiguracji, potem kolejne (maksymalnie +4).
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
