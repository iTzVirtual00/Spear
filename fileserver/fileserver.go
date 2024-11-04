package fileserver

import (
	"fmt"
	"html/template"
	"main/config"
	"main/utils"
	"net/http"
	"os"
	"strings"
)

// FileserverBackend holds the set of paths that can be served by this backend.
type FileserverBackend struct {
	Config     *config.SpearConfig
	Parameters config.SpearParameters
}

func (fb FileserverBackend) RegisterFileserver(mux *http.ServeMux) {
	mux.HandleFunc("/files/", fb.fileserverHandler)
	mux.HandleFunc("/files", fb.fileserverHandler)
}

// fileserverHandler handles serving files and directories based on fb.Paths.
func (fb FileserverBackend) fileserverHandler(w http.ResponseWriter, r *http.Request) {
	relativePath := utils.NewPath(strings.TrimPrefix(r.URL.Path, "/files/"))

	var target *utils.Path = nil
	for _, allowedPath := range fb.Parameters.Paths {
		res, err := relativePath.IsRelativeTo(allowedPath)
		if err != nil {
			target = nil
			break
		}
		if res {
			target, err = relativePath.Abs()
			if err != nil {
				target = nil
				break
			}
			break
		}
	}

	if target == nil { // remove carefully (required by forloop)
		http.Error(w, "404 Not Found", http.StatusNotFound)
		return
	}

	isDir, _ := target.IsDir()
	isRegular, _ := target.IsRegular()

	fmt.Println(target, isRegular)
	if isDir {
		fb.serveDirectory(w, r, target)
		return
	} else if isRegular {
		fb.serveFile(w, r, target)
		return
	}

	http.Error(w, "404 Not Found", http.StatusNotFound)
	return
}

// serveDirectory lists the contents of a directory in a nginx-style.
func (fb FileserverBackend) serveDirectory(w http.ResponseWriter, r *http.Request, dirPath *utils.Path) {

	entries, err := os.ReadDir(dirPath.String())
	fmt.Println(entries)
	if err != nil {
		http.Error(w, "404 Not Found", http.StatusNotFound)
		return
	}

	const tpl = `
<!DOCTYPE html>
<html>
<head>
	<title>Index of /{{.Path}}</title>
</head>
<body>
	<h1>Index of /{{.Path}}</h1>
	<ul>
		{{range .Entries}}
		<li><a href="/files/{{$.Path}}/{{.Name}}">{{.}}</a></li>
		{{end}}
	</ul>
</body>
</html>
`
	tmpl, err := template.New("directory").Parse(tpl)
	if err != nil {
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
	err = tmpl.Execute(w, map[string]interface{}{
		"Path":    strings.TrimPrefix(r.URL.Path, "/files/"),
		"Entries": entries,
	})
	if err != nil {
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (fb FileserverBackend) serveFile(w http.ResponseWriter, r *http.Request, filePath *utils.Path) {
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filePath.Base()))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", "no-cache")

	// TODO: replace dummy data
	fileData := []byte("Example content of the file.")
	w.Write(fileData)
}
