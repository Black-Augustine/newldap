package main

import (
	"io/fs"
	"net/http"
	"strings"

	"newldap/web"
)

// spaHandler 服务内嵌的前端静态资源；未命中的路径回退到 index.html（SPA 路由）。
func spaHandler() http.Handler {
	sub := web.Dist()
	files := http.FileServerFS(sub)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/")
		if p == "" {
			files.ServeHTTP(w, r)
			return
		}
		if _, err := fs.Stat(sub, p); err != nil {
			// SPA 回退：交给前端路由
			index, ierr := fs.ReadFile(sub, "index.html")
			if ierr != nil {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write(index) //nolint:errcheck
			return
		}
		files.ServeHTTP(w, r)
	})
}
