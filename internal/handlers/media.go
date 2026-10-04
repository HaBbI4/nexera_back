package handlers

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
)

// NewLocalMediaHandler отдаёт файлы, сохранённые LocalStorage, по пути GET /media/{path...}.
// Используется только при MEDIA_DRIVER=local — при MEDIA_DRIVER=s3 файлы отдаются
// напрямую из бакета по постоянной публичной ссылке.
func NewLocalMediaHandler(root string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			writeErrorResponse(w, http.StatusMethodNotAllowed, "Method Not Allowed")
			return
		}

		// path.Clean убирает "..", защищая от выхода за пределы root
		cleaned := path.Clean("/" + r.PathValue("path"))
		fullPath := filepath.Join(root, filepath.FromSlash(cleaned))

		info, err := os.Stat(fullPath)
		if err != nil || info.IsDir() {
			writeErrorResponse(w, http.StatusNotFound, "file not found")
			return
		}

		http.ServeFile(w, r, fullPath)
	}
}
