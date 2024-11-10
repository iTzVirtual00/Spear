package fileserver

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"spear/auths"
	"spear/config"
	"spear/utils"
	"strconv"
	"strings"
)

// FileserverBackend holds the set of paths that can be served by this backend.
type FileserverBackend struct {
	Config      *config.SpearConfig
	Parameters  config.SpearParameters
	AuthMethods []auths.AuthMethod
	templates   *template.Template
}

func (fb *FileserverBackend) RegisterFileserver(mux *http.ServeMux) {
	var err error
	fb.templates, err = template.New("").ParseFiles("templates/files.gohtml")
	if err != nil {
		log.Fatal("error while loading template/files.gohtml", err)
	}
	log.Println("Fileserver Template loaded")
	mux.HandleFunc("/files/", fb.fileserverHandler)
	//mux.HandleFunc("/files", fb.fileserverHandler)
}

// fileserverHandler handles serving files and directories based on fb.Paths.
func (fb *FileserverBackend) fileserverHandler(w http.ResponseWriter, r *http.Request) {
	var contactName = ""
	var authResult auths.AuthResult
OuterLoop:
	for _, authMethod := range fb.AuthMethods {
		authResult = authMethod.Authenticate(w, r)
		fmt.Println(authResult)
		switch authResult.State {
		case auths.Valid:
			fallthrough
		case auths.Denied:
			break OuterLoop
		}
	}

	switch authResult.State {
	case auths.Valid:
		contactName = authResult.ContactName
		break
	// if last state was Denied, we found a valid auth with incorrect credentials
	case auths.Denied:
		http.Error(w, "Permission denied", http.StatusForbidden)
		return
	// if last state was Inapplicable, no auth method succeded, we now try default auth
	case auths.Inapplicable:
		http.Redirect(w, r, "/basic"+"?location="+r.URL.Path, http.StatusFound)
		return
	}

	fmt.Println("authenticated: ", contactName)

	relativePath := utils.NewPath(strings.TrimPrefix(r.URL.Path, "/files/"))

	var isDir = false
	var isRegular = false
	var target *utils.Path = nil
	var filesOverride []*utils.Path = nil

	if r.URL.Path != "/files/" {
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
			http.Error(w, http.StatusText(http.StatusNotFound)+"<br>"+contactName, http.StatusNotFound)
			return
		}
		isDir, _ = target.IsDir()
		isRegular, _ = target.IsRegular()
	} else {
		isDir = true
		isRegular = false
		filesOverride = fb.Parameters.Paths
		log.Println(filesOverride)
	}
	fmt.Println(target, isRegular)
	if isDir {
		if !strings.HasSuffix(r.URL.Path, "/") {
			http.Redirect(w, r, r.URL.Path+"/", http.StatusFound)
			return
		}
		fb.serveDirectory(w, r, target, filesOverride, contactName)
		return
	} else if isRegular {
		fb.serveFile(w, r, target, contactName)
		return
	}

	http.Error(w, http.StatusText(http.StatusNotFound)+"<br>"+contactName, http.StatusNotFound)
	return
}

// serveDirectory lists the contents of a directory in a nginx-style.
func (fb *FileserverBackend) serveDirectory(w http.ResponseWriter, r *http.Request, dirPath *utils.Path, filesOverride []*utils.Path, contactName string) {
	var err error
	var entries []*utils.Path
	entries = filesOverride
	if filesOverride == nil {
		entries, err = dirPath.ReadDir()
	}

	fmt.Println(entries)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	if fb.templates == nil {
		log.Fatal("Fileserver's templates are nil")
		return
	}

	err = fb.templates.ExecuteTemplate(w, "files.gohtml", map[string]interface{}{
		"Path":    strings.TrimPrefix(r.URL.Path, "/files/"),
		"Entries": entries,
	})
	if err != nil {
		fmt.Println(err.Error())
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func (fb *FileserverBackend) serveFile(w http.ResponseWriter, r *http.Request, filePath *utils.Path, contactName string) {
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
