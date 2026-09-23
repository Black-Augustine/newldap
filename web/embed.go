// Package web 内嵌构建好的前端静态资源（web/dist）。
// dist/index.html 是构建前的占位页；执行 cd web && npm run build 后被真实产物覆盖。
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// Dist 返回 dist 子文件系统。
func Dist() fs.FS {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic(err)
	}
	return sub
}
