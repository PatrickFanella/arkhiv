package app

import "net/http"

// Handler exposes process health only; persistence is not connected yet.
func Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write([]byte(`{"status":"ok","service":"arkhiv","stage":"bootstrap"}`))
	})
	return mux
}
