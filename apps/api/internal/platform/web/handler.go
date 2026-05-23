package web

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

func Handler() http.Handler {
	sub, err := fs.Sub(StaticFS, "static")
	if err != nil {
		panic(err)
	}

	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}

		cleanPath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if cleanPath == "." || cleanPath == "" {
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
			return
		}

		if _, err := fs.Stat(sub, cleanPath); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}

		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}
