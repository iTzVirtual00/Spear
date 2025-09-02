package serve

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"spear/utils"
	"strings"
)

type JsonFileResponse struct {
	Name  string `json:"name"`
	Size  uint64 `json:"size"`
	Mtime int64  `json:"mtime"`
	IsDir bool   `json:"isDir"`
}

// JsonResponse wraps directory or file responses for type distinction
type JsonResponse struct {
	Type     string             `json:"type"` // "file" or "directory"
	Contents []JsonFileResponse `json:"contents"`
}

func asJson(w http.ResponseWriter, entries []*utils.Path) {
	var jsonResponse []JsonFileResponse
	for _, entry := range entries {
		isDir := entry.IsDir()
		stat, err := entry.Stat()
		if err != nil {
			log.Println("Error getting file stat:", err)
			continue
		}
		jsonResponse = append(jsonResponse, JsonFileResponse{
			Name:  entry.Base(),
			Size:  uint64(stat.Size()),
			Mtime: stat.ModTime().Unix(),
			IsDir: isDir,
		})
	}
	resp := JsonResponse{
		Type:     "directory",
		Contents: jsonResponse,
	}
	json.NewEncoder(w).Encode(resp)
}

func asHtml(w http.ResponseWriter, entries []*utils.Path, template *template.Template, path string) {
	if template == nil {
		fmt.Errorf("template is nil")
	}

	err := template.ExecuteTemplate(w, "files.gohtml", map[string]interface{}{
		"Path":    path,
		"Entries": entries,
	})
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		fmt.Errorf("error executing template: %w", err)
	}
}

// ServeDirectory lists the contents of a directory in a nginx-autoindex style.
func ServeDirectory(w http.ResponseWriter, r *http.Request, template *template.Template, dirPath *utils.Path, contactName string) {
	var err error
	var entries []*utils.Path
	entries, err = dirPath.ReadDirOrdered()

	if err != nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	// serve json
	if strings.Contains(r.Header.Get("Accept"), "application/json") {
		asJson(w, entries)
		return
	}

	// serve html
	if template == nil {
		log.Fatal("Fileserver's templates are nil")
		return
	}
	asHtml(w, entries, template, dirPath.String())

}

func ServeFile(w http.ResponseWriter, r *http.Request, filePath *utils.Path, contactName string) {
	if r.URL.Query().Has("meta") {
		stat, err := filePath.Stat()
		if err != nil {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		resp := JsonResponse{
			Type: "file",
			Contents: []JsonFileResponse{{
				Name:  filePath.Base(),
				Size:  uint64(stat.Size()),
				Mtime: stat.ModTime().Unix(),
				IsDir: false,
			}},
		}
		json.NewEncoder(w).Encode(resp)
		return
	}
	_, err := filePath.Stat()
	if err != nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filePath.Base()))

	if fsys := filePath.GetFS(); fsys != nil {
		http.ServeFileFS(w, r, fsys, filePath.String())
	} else {
		http.ServeFile(w, r, filePath.String())
	}
}
