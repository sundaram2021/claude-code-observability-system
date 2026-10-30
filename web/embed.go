package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static/*
var staticFiles embed.FS

// GetFileSystem returns an http.FileSystem rooted at the static assets directory.
func GetFileSystem() http.FileSystem {
	sub, _ := fs.Sub(staticFiles, "static")
	return http.FS(sub)
}
