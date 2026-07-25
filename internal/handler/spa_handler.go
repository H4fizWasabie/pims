package handler

import (
	"io/fs"
	"net/http"
	"strings"
)

func (h *Handler) HandleSPA(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" || path == "/" {
		path = "index.html"
	}

	data, err := fs.ReadFile(h.StaticFS, path)
	if err != nil {
		// Fallback to index.html for SPA routing
		data, err = fs.ReadFile(h.StaticFS, "index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
	}

	contentType := "text/html; charset=utf-8"
	if strings.HasSuffix(path, ".css") {
		contentType = "text/css"
	} else if strings.HasSuffix(path, ".js") {
		contentType = "application/javascript"
	} else if strings.HasSuffix(path, ".png") {
		contentType = "image/png"
	}
	w.Header().Set("Content-Type", contentType)
	w.Write(data)
}
