package fileserver

import (
	"fmt"
	"html/template"
	"net/http"
	"os"
	"spear/config"
	"spear/utils"
	"strconv"
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
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
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

	http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	return
}

// serveDirectory lists the contents of a directory in a nginx-style.
func (fb FileserverBackend) serveDirectory(w http.ResponseWriter, r *http.Request, dirPath *utils.Path) {

	entries, err := os.ReadDir(dirPath.String())
	fmt.Println(entries)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
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
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	err = tmpl.Execute(w, map[string]interface{}{
		"Path":    strings.TrimPrefix(r.URL.Path, "/files/"),
		"Entries": entries,
	})
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func (fb FileserverBackend) serveFile(w http.ResponseWriter, r *http.Request, filePath *utils.Path) {
	const chunkSize = 1 * 1024 * 1024 // 1MB buffer size
	fileStat, err := os.Stat(filePath.String())
	if err != nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	fileSize := uint64(fileStat.Size())
	fmt.Println(r.Header.Get("Range"))
	h := strings.TrimPrefix(r.Header.Get("Range"), "bytes=")
	var ranges []utils.Range
	if h != "" {
		ranges, err = utils.ParseRangeHeader(h, fileSize, chunkSize)
		if err != nil {
			fmt.Println(err)
			http.Error(w, http.StatusText(http.StatusRequestedRangeNotSatisfiable), http.StatusRequestedRangeNotSatisfiable)
			return
		}
	} else {
		ranges = []utils.Range{utils.NewRange(0, fileSize-1)}
	}

	inFile, err := os.Open(filePath.String())
	if err != nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	if len(ranges) > 1 {
		http.Error(w, http.StatusText(http.StatusRequestedRangeNotSatisfiable), http.StatusRequestedRangeNotSatisfiable)
		return
	}
	rg := ranges[0]
	_, err = inFile.Seek(int64(rg.Start), 0)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusRequestedRangeNotSatisfiable), http.StatusRequestedRangeNotSatisfiable)
		return
	}

	buf := make([]byte, chunkSize)

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filePath.Base()))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Length", strconv.FormatUint(rg.Size, 10))
	//w.Header().Set("Cache-Control", "no-cache")
	if rg.Start != 0 || rg.End != fileSize-1 {
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %v-%v/%v", rg.Start, rg.End, fileSize))
		w.WriteHeader(http.StatusPartialContent)
		fmt.Println("Partial content:", w.Header().Get("Content-Range"), w.Header().Get("Content-Length"))
	}
	fmt.Println(rg)
	for i := uint64(0); i < rg.End; i += chunkSize {
		read, err := inFile.Read(buf)
		if err != nil {
			w.Header().Del("Content-Disposition")
			w.Header().Del("Accept-Ranges")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		_, err = w.Write(buf[:read])
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if read < chunkSize {
			break
		}
	}
}
