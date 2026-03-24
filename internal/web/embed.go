package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed dist/index.html dist/assets/*
var site embed.FS

func LoginPage() ([]byte, error) {
	return site.ReadFile("dist/index.html")
}

func MustLoginPage() []byte {
	page, err := LoginPage()
	if err != nil {
		panic(err)
	}
	return page
}

func AssetsHandler() http.Handler {
	subtree, err := fs.Sub(site, "dist")
	if err != nil {
		panic(err)
	}
	return http.FileServerFS(subtree)
}
