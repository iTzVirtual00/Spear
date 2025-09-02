package client_vfs

import (
	"bytes"
	"io"
	"io/fs"
	"path"
	"path/filepath"
	"strings"
	"time"

	"spear/serve"
)

// NewSpearFS returns an fs.FS rooted at `root` on the remote.
func NewSpearFS(client *SpearClient, root string) fs.FS {
	root = path.Clean("/" + strings.TrimPrefix(root, "/")) // ensure leading "/" and clean
	if root == "/" {
		root = "" // make joining simpler
	}
	return &spearFS{c: client, root: root}
}

type spearFS struct {
	c    *SpearClient
	root string // remote root without trailing slash, e.g. "", "/sub"
}

func (s *spearFS) Open(name string) (fs.File, error) {
	// fs.FS rules: names are slash-separated, no leading slash, no ".."
	if filepath.IsAbs(name) || strings.Contains(name, "..") {
		return nil, fs.ErrInvalid
	}
	name = path.Clean(name)
	if name == "." {
		name = "" // root
	}

	remotePath := path.Join(s.root, name)
	if !strings.HasPrefix(remotePath, "/") {
		remotePath = "/" + remotePath
	}

	resp, err := s.c.GetFiles(remotePath)
	if err != nil {
		return nil, fs.ErrNotExist
	}
	switch resp.Type {
	case "directory":
		return &dirFile{remote: remotePath, entries: resp.Contents}, nil
	case "file":
		fd, ferr := s.c.DownloadFile(remotePath)
		if ferr != nil {
			return nil, fs.ErrNotExist
		}
		data := fd.Data
		info := &spearFileInfo{
			name: path.Base(fd.Name),
			size: int64(len(data)),
			dir:  false,
		}
		return &spearFile{
			name: fd.Name,
			buf:  bytes.NewReader(data),
			info: info,
		}, nil
	default:
		return nil, fs.ErrInvalid
	}
}

// ===== Directory implementation =====

type dirFile struct {
	remote  string
	entries []serve.JsonFileResponse
	i       int
	closed  bool
}

func (d *dirFile) Stat() (fs.FileInfo, error) {
	if d.closed {
		return nil, fs.ErrClosed
	}
	return &spearFileInfo{
		name: path.Base(d.remote),
		dir:  true,
	}, nil
}

func (d *dirFile) Read([]byte) (int, error) { // dirs aren't byte-readable
	return 0, io.EOF
}

func (d *dirFile) Close() error {
	d.closed = true
	return nil
}

// Implement fs.ReadDirFile for efficient listing
func (d *dirFile) ReadDir(n int) ([]fs.DirEntry, error) {
	if d.closed {
		return nil, fs.ErrClosed
	}
	if d.i >= len(d.entries) {
		return []fs.DirEntry{}, nil
	}
	var out []fs.DirEntry
	limit := len(d.entries)
	if n > 0 && d.i+n < limit {
		limit = d.i + n
	}
	for ; d.i < limit; d.i++ {
		e := d.entries[d.i]
		out = append(out, &spearDirEntry{e})
	}
	return out, nil
}

type spearDirEntry struct {
	serve.JsonFileResponse
}

func (e *spearDirEntry) Name() string { return e.JsonFileResponse.Name }
func (e *spearDirEntry) IsDir() bool  { return e.JsonFileResponse.IsDir }
func (e *spearDirEntry) Type() fs.FileMode {
	if e.IsDir() {
		return fs.ModeDir
	}
	return 0
}
func (e *spearDirEntry) Info() (fs.FileInfo, error) {
	return &spearFileInfo{
		name: e.Name(),
		dir:  e.IsDir(),
		// If your JsonFileResponse has size/modtime, set them here.
	}, nil
}

// ===== File implementation =====

type spearFile struct {
	name   string
	buf    *bytes.Reader
	info   *spearFileInfo
	closed bool
}

func (f *spearFile) Stat() (fs.FileInfo, error) {
	if f.closed {
		return nil, fs.ErrClosed
	}
	return f.info, nil
}

func (f *spearFile) Read(p []byte) (int, error) {
	if f.closed {
		return 0, fs.ErrClosed
	}
	return f.buf.Read(p)
}

func (f *spearFile) Close() error {
	f.closed = true
	return nil
}

// ===== FileInfo =====

type spearFileInfo struct {
	name    string
	size    int64
	dir     bool
	modTime time.Time
}

func (fi *spearFileInfo) Name() string { return fi.name }
func (fi *spearFileInfo) Size() int64  { return fi.size }
func (fi *spearFileInfo) Mode() fs.FileMode {
	if fi.dir {
		return fs.ModeDir | 0o555
	}
	return 0o444
}
func (fi *spearFileInfo) ModTime() time.Time { return fi.modTime }
func (fi *spearFileInfo) IsDir() bool        { return fi.dir }
func (fi *spearFileInfo) Sys() any           { return nil }

// ===== Optional helpers =====

// Sub returns a new fs.FS rooted at remote subdir.
func (s *spearFS) Sub(dir string) (fs.FS, error) {
	if strings.Contains(dir, "..") || path.IsAbs(dir) {
		return nil, fs.ErrInvalid
	}
	dir = path.Clean(dir)
	remote := path.Join(s.root, dir)
	if !strings.HasPrefix(remote, "/") {
		remote = "/" + remote
	}
	// sanity check that it exists and is a dir
	if _, err := s.c.GetFiles(remote); err != nil {
		return nil, fs.ErrNotExist
	}
	return &spearFS{c: s.c, root: remote}, nil
}
