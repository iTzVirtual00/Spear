package utils

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type PathError struct {
	s string
}

func (e *PathError) Error() string {
	return e.s
}

type Path struct {
	path string
	fsys fs.FS
}

func (p Path) Type() fs.FileMode {
	if info, err := p.Stat(); err == nil {
		return info.Mode().Type()
	}
	return 0
}

func (p Path) Info() (fs.FileInfo, error) {
	return p.Stat()
}

func (p Path) GetFS() fs.FS {
	return p.fsys
}

func NewPath(p string) *Path {
	return NewPathFS(p, nil)
}
func NewPathFS(p string, fsys fs.FS) *Path {
	return &Path{
		path: filepath.ToSlash(filepath.Clean(p)),
		fsys: fsys,
	}
}

func (p Path) String() string {
	isDir := p.IsDir()
	if isDir && !strings.HasSuffix(p.path, "/") {
		return p.path + "/"
	} else {
		return p.path
	}
}

func (p Path) Base() string {
	isDir := p.IsDir()
	res := filepath.Base(p.path)
	if isDir && !strings.HasSuffix(res, "/") {
		return res + "/"
	} else {
		return res
	}
}

func (p Path) Name() string {
	return filepath.Base(p.path)
}

func (p Path) Size() int64 {
	info, err := p.Stat()
	if err != nil {
		return -1
	}
	return info.Size()
}

func (p Path) SizeHumanReadable() string {
	size := p.Size()
	if size < 0 {
		return "N/A"
	}
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

func (p Path) IsAbs() bool {
	return filepath.IsAbs(p.path)
}

func (p Path) IsRelative() bool {
	return !filepath.IsAbs(p.path)
}

func (p Path) Abs() (*Path, error) {
	res, err := filepath.Abs(p.path)
	if err != nil {
		return nil, err
	}
	return NewPathFS(res, p.fsys), nil
}

func (p Path) Exists() bool {
	_, err := p.Stat()
	if err != nil {
		return false
	}
	return true
}

func (p Path) IsDir() bool {
	res, err := p.Stat()
	if err != nil {
		return false
	}
	return res.IsDir()
}

// CanonicalPath returns the absolute, symlink-evaluated, user-home-expanded path.
// WARN: THIS FUNCTION IGNORES THE FSYS PARAMETER! It always uses the real OS filesystem.
func (p Path) CanonicalPath() (*Path, error) {
	path := p.path
	if len(path) > 0 && (path[0] == '~') {
		if home, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(home, path[1:])
		}
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	fp, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return nil, err
	}
	return NewPath(fp), nil
}

func (p Path) IsRegular() bool {
	res, err := p.Stat()
	if err != nil {
		return false
	}
	return res.Mode().IsRegular()
}

func (p Path) Stat() (fs.FileInfo, error) {
	f, err := p.Open()
	if err != nil {
		return nil, err
	}
	return f.Stat()
}

// INTERACTS WITH FS
func (p Path) Open() (fs.File, error) {
	if p.fsys == nil {
		return os.Open(p.path)
	}
	return p.fsys.Open(p.path)
}

// INTERACTS WITH FS
func (p Path) ReadDir() ([]*Path, error) {
	var res []*Path
	var entries []fs.DirEntry
	var err error
	if p.fsys == nil {
		entries, err = os.ReadDir(p.path)
	} else {
		entries, err = fs.ReadDir(p.fsys, p.path)
	}
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		res = append(res, NewPathFS(p.path+"/"+entry.Name(), p.fsys))
	}

	return res, nil
}

func (p Path) ReadDirOrdered() ([]*Path, error) {
	entries, err := p.ReadDir()
	if err != nil {
		return nil, err
	}

	sort.Slice(entries, func(a int, b int) bool { // is i < j?
		di := entries[a].IsDir()
		dj := entries[b].IsDir()

		if di != dj {
			// Directories first
			// 1 when Dir & File, 0 when File & Dir
			return di && !dj
		}
		// Same type → alphabetical by name (case-insensitive is optional)
		return entries[a].Name() < entries[b].Name()
	})

	return entries, nil
}

func (p Path) IsRelativeTo(parent *Path) (bool, error) {
	parent, err := parent.Abs()
	if err != nil {
		return false, err
	}
	toCheck, err := p.Abs()
	if err != nil {
		return false, err
	}
	return strings.HasPrefix(toCheck.path, parent.path), nil
}

func isSlashRune(r rune) bool { return r == '/' || r == '\\' }
func PathContainsDotDot(v string) bool {
	if !strings.Contains(v, "..") {
		return false
	}
	for _, ent := range strings.FieldsFunc(v, isSlashRune) {
		if ent == ".." {
			return true
		}
	}
	return false
}

// VisibleChildren returns the immediate children to show at `relativePath`,
// and whether this location should be listed fully from disk (fullDir=true)
// because it's at/inside a directory that was explicitly shared.
// Deprecated
func VisibleChildren(relativePath *Path, Files []*Path) (children []*Path, fullDir bool) {
	here := cleanSegs(relativePath.String()) // "" for root, not "." or "./"
	hereSegs := split(here)

	cwd, _ := os.Getwd()
	nextNames := map[string]struct{}{}

	hasPrefixSegs := func(pref, path []string) bool {
		if len(pref) > len(path) {
			return false
		}
		for i := range pref {
			if pref[i] != path[i] {
				return false
			}
		}
		return true
	}

	for _, fp := range Files {
		if fp == nil {
			continue
		}
		src := strings.TrimSpace(fp.String())
		if src == "" {
			continue
		}

		abs, _ := filepath.Abs(src)
		rel := relFrom(abs, cwd) // virtual path shown to the user
		rel = clean(rel)         // "somefolder/abc" or "file.txt"
		if rel == "" {
			continue
		}

		fSegs := split(rel)

		// Is this an explicitly shared directory?
		isDir := false
		if st, err := os.Stat(abs); err == nil && st.IsDir() {
			isDir = true
		}

		// Case A: a shared directory -> full listing at/inside it.
		if isDir && (rel == here || hasPrefixSegs(split(rel), hereSegs)) {
			return nil, true
		}

		// Case B: deep items -> expose only the next segment from `here`.
		if hasPrefixSegs(hereSegs, fSegs) && len(fSegs) > len(hereSegs) {
			next := fSegs[len(hereSegs)]
			if next != "" && next != "." {
				nextNames[next] = struct{}{}
			}
		}
	}

	if len(nextNames) == 0 {
		return nil, false
	}
	names := make([]string, 0, len(nextNames))
	for n := range nextNames {
		names = append(names, n)
	}
	sort.Strings(names)

	base := here
	for _, n := range names {
		var v string
		if base == "" {
			v = n
		} else {
			v = base + "/" + n
		}
		children = append(children, NewPath(v))
	}
	return children, false
}

// ---- helpers (unexported) ----

func relFrom(abs, cwd string) string {
	if rel, err := filepath.Rel(cwd, abs); err == nil && !strings.HasPrefix(rel, "..") && rel != "." {
		return filepath.ToSlash(rel)
	}
	return filepath.ToSlash(filepath.Base(abs))
}

func clean(s string) string {
	s = filepath.ToSlash(filepath.Clean(s))
	return strings.Trim(s, "/")
}

func cleanSegs(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || s == "." || s == "./" {
		return ""
	} // treat as root
	return clean(s)
}

func split(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "/")
}
