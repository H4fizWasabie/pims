package handler

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"sync"
)

var contentTypes = map[string]string{
	".html":  "text/html; charset=utf-8",
	".css":   "text/css; charset=utf-8",
	".js":    "application/javascript; charset=utf-8",
	".json":  "application/json",
	".png":   "image/png",
	".svg":   "image/svg+xml",
	".ico":   "image/x-icon",
	".woff2": "font/woff2",
}

// asset is a static file prepared once: its bytes, a gzip copy (when it
// compresses), and an ETag so browsers can revalidate with a cheap 304.
// Files are embedded in the binary, so they never change while it runs.
type asset struct {
	raw, gz     []byte
	etag, ctype string
}

var assets sync.Map // name -> *asset

func (h *Handler) loadAsset(name string) (*asset, error) {
	if a, ok := assets.Load(name); ok {
		return a.(*asset), nil
	}
	data, err := fs.ReadFile(h.StaticFS, name)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(data)
	a := &asset{raw: data, etag: `W/"` + hex.EncodeToString(sum[:8]) + `"`, ctype: contentTypes[path.Ext(name)]}
	if a.ctype == "" {
		a.ctype = "application/octet-stream"
	}
	if path.Ext(name) != ".woff2" && path.Ext(name) != ".png" { // already compressed
		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		zw.Write(data)
		zw.Close()
		if buf.Len() < len(data) {
			a.gz = buf.Bytes()
		}
	}
	assets.Store(name, a)
	return a, nil
}

func (h *Handler) HandleSPA(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		h.Error(w, 404, "Not found")
		return
	}
	name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	if name == "" {
		name = "index.html"
	}
	a, err := h.loadAsset(name)
	if err != nil {
		// A path with a file extension is a missing file; anything else is an SPA route.
		if path.Ext(name) != "" {
			http.NotFound(w, r)
			return
		}
		name = "index.html"
		if a, err = h.loadAsset(name); err != nil {
			http.NotFound(w, r)
			return
		}
	}

	hd := w.Header()
	hd.Set("Content-Type", a.ctype)
	hd.Set("ETag", a.etag)
	hd.Add("Vary", "Accept-Encoding")
	if strings.HasPrefix(name, "vendor/") {
		hd.Set("Cache-Control", "public, max-age=604800") // third-party files change rarely
	} else {
		hd.Set("Cache-Control", "no-cache") // always revalidate app files; 304 when unchanged
	}
	if r.Header.Get("If-None-Match") == a.etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	if a.gz != nil && strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		hd.Set("Content-Encoding", "gzip")
		w.Write(a.gz)
		return
	}
	w.Write(a.raw)
}
