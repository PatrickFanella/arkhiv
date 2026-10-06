package sessionservice

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func (s *Server) proxy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodPatch && r.Method != http.MethodDelete {
		write(w, 405, map[string]string{"error": "Method unavailable."})
		return
	}
	if r.Method != http.MethodGet && !s.csrf(r) {
		write(w, 403, map[string]string{"error": "Same-origin authenticated action required."})
		return
	}
	identity, err := s.identity(r)
	anonymousChatRead := r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/chat/v1/rooms/")
	if err != nil && !anonymousChatRead {
		write(w, 401, map[string]string{"error": "Sign-in required."})
		return
	}
	base, _ := url.Parse(strings.TrimSuffix(s.Config.CoreURL, "/"))
	base.Path = r.URL.Path
	base.RawPath = r.URL.RawPath
	base.RawQuery = r.URL.RawQuery
	request, err := http.NewRequestWithContext(r.Context(), r.Method, base.String(), http.MaxBytesReader(w, r.Body, 8192))
	if err != nil {
		write(w, 400, map[string]string{"error": "Invalid request."})
		return
	}
	request.Header.Set("Content-Type", "application/json")
	if revision := r.Header.Get("If-Match"); revision != "" {
		request.Header.Set("If-Match", revision)
	}
	if identity != nil {
		assertion, err := s.assertion(identity)
		if err != nil {
			write(w, 503, map[string]string{"error": "Identity unavailable."})
			return
		}
		request.Header.Set("Authorization", "Bearer "+assertion)
	} // Browser cookies and supplied Authorization/forwarding headers are never forwarded.
	response, err := s.client.Do(request)
	if err != nil {
		write(w, 503, map[string]string{"error": "Personal service unavailable. Video remains available."})
		return
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, (16<<20)+1))
	if err != nil || len(body) > 16<<20 {
		write(w, 503, map[string]string{"error": "Personal response unavailable."})
		return
	}
	if !strings.HasPrefix(response.Header.Get("Content-Type"), "application/json") {
		write(w, 503, map[string]string{"error": "Invalid personal service response."})
		return
	}
	if response.Header.Get("Content-Disposition") != "" {
		w.Header().Set("Content-Disposition", response.Header.Get("Content-Disposition"))
	}
	if r.Method == http.MethodDelete && r.URL.Path == "/api/personal/v1/me" && response.StatusCode == 200 {
		s.setCookie(w, sessionCookie, "", -1)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(response.StatusCode)
	_, _ = w.Write(body)
}
