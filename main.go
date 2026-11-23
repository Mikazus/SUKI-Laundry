package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	handler "suki-laundry/api"
)

func main() {
	mux := http.NewServeMux()

	// Semua request /api/* diteruskan ke handler bersama di api/index.go
	mux.HandleFunc("/api/", handler.Handler)

	// Static files + Clean URL Fallback Routing
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if p == "/" {
			http.Redirect(w, r, "/login.html", http.StatusFound)
			return
		}

		// Sajikan file fisik jika ada (.html partial, .js, login.html, dll.)
		cleanPath := filepath.Clean(strings.TrimPrefix(p, "/"))
		if info, err := os.Stat(cleanPath); err == nil && !info.IsDir() {
			http.ServeFile(w, r, cleanPath)
			return
		}

		// Fallback untuk clean URL SPA per role
		switch {
		case strings.HasPrefix(p, "/pelanggan"):
			http.ServeFile(w, r, "pelanggan/index.html")
		case strings.HasPrefix(p, "/owner"):
			http.ServeFile(w, r, "owner/index.html")
		default:
			http.NotFound(w, r)
		}
	})

	fmt.Println("SUKI Laundry berjalan di http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}