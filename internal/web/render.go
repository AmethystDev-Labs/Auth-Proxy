package web

import "net/http"

func Handler(loginPage []byte, assets http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/index.html" || r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(loginPage)
			return
		}
		assets.ServeHTTP(w, r)
	})
}
