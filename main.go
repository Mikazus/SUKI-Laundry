package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	handler "suki-laundry/api"
)

func loadEnv(filename string) {
	f, err := os.Open(filename)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if k, v, ok := strings.Cut(line, "="); ok {
			_ = os.Setenv(strings.TrimSpace(k), strings.Trim(strings.TrimSpace(v), `"'`))
		}
	}
}

func main() {
	loadEnv(".env")

	mux := http.NewServeMux()
	mux.HandleFunc("/api/", handler.Handler)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if p == "/" {
			http.Redirect(w, r, "/login.html", http.StatusFound)
			return
		}

		cleanPath := filepath.Clean(strings.TrimPrefix(p, "/"))
		if info, err := os.Stat(cleanPath); err == nil && !info.IsDir() {
			http.ServeFile(w, r, cleanPath)
			return
		}

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