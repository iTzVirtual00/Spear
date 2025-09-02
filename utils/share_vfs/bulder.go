package share_vfs

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"spear/utils"
)

type dirMode int

const (
	// DirMountKeep exposes the directory as a subfolder at the virtual mount point
	// e.g. MountDir("pics/2025", "/real/Pictures/2025", DirMountKeep) -> /pics/2025/<...>
	DirMountKeep dirMode = iota

	// DirMountStrip exposes the *contents* of the directory at the virtual mount point
	// e.g. MountDir("pics", "/real/Pictures/Folder", DirMountStrip) -> /pics/<contents-of-Folder>
	DirMountStrip
)

type mount struct {
	virtual string      // cleaned, slash-separated virtual path (no leading or trailing '/')
	source  *utils.Path // real path on disk (utils.Path)
	kind    string      // "file" or "dir"
	mode    dirMode     // for dir mounts: keep or strip
}

// Builder assembles a small read-only union filesystem that implements fs.FS.
type Builder struct {
	mounts []mount
	// dirs holds synthetic directories that must exist virtually even if not backed
	// by a single source directory (e.g., /alias that aggregates multiple items)
	dirs map[string]struct{}
}

func NewBuilder() *Builder {
	return &Builder{dirs: map[string]struct{}{}}
}

// EnsureDir ensures the given *virtual* directory exists in the union FS tree.
func (b *Builder) EnsureDir(virtual string) {
	v := cleanSlash(virtual)
	if v == "" {
		return
	}
	// add this dir and all parents
	for {
		b.dirs[v] = struct{}{}
		parent := parentDir(v)
		if parent == "" || parent == "." {
			break
		}
		if _, ok := b.dirs[parent]; ok {
			break
		}
		v = parent
	}
}

// MountFile mounts a single *file* at the given *virtual* path.
func (b *Builder) MountFile(virtualFile string, realFile *utils.Path) {
	v := cleanSlash(virtualFile)
	b.EnsureDir(parentDir(v))
	b.mounts = append(b.mounts, mount{
		virtual: v,
		source:  realFile,
		kind:    "file",
	})
}

// MountDir mounts a *directory* at the given *virtual* path.
// mode controls whether the directory itself is kept or stripped (contents-only).
func (b *Builder) MountDir(virtual string, realDir *utils.Path, mode dirMode) {
	v := cleanSlash(virtual)
	b.EnsureDir(v)
	b.mounts = append(b.mounts, mount{
		virtual: v,
		source:  realDir,
		kind:    "dir",
		mode:    mode,
	})
}

// Build finalizes the builder into an fs.FS implementation.
func (b *Builder) Build() fs.FS {
	// Deterministic ordering for stable listings
	sort.Slice(b.mounts, func(i, j int) bool {
		if b.mounts[i].virtual == b.mounts[j].virtual {
			return b.mounts[i].source.String() < b.mounts[j].source.String()
		}
		return b.mounts[i].virtual < b.mounts[j].virtual
	})
	return &unionFS{
		mounts: b.mounts,
		dirs:   b.dirs,
	}
}

// -------------------- unionFS implements fs.FS --------------------

type unionFS struct {
	mounts []mount
	dirs   map[string]struct{}
}

func (u *unionFS) Open(name string) (fs.File, error) {
	p := cleanSlash(name)
	if p == "." {
		p = ""
	}

	// Root directory
	if p == "" {
		return u.openDir("")
	}

	// Exact *file* mount
	for _, m := range u.mounts {
		if m.kind == "file" && m.virtual == p {
			// Open the real file on disk
			return os.Open(m.source.String())
		}
	}

	// Directory mounts: KEEP and STRIP both map /p into the underlying directory tree.
	for _, m := range u.mounts {
		if m.kind != "dir" {
			continue
		}
		if p == m.virtual || strings.HasPrefix(p, m.virtual+"/") {
			rest := strings.TrimPrefix(p, m.virtual)
			rest = strings.TrimPrefix(rest, "/")
			target := filepath.Join(m.source.String(), filepath.FromSlash(rest))
			return os.Open(target)
		}
	}

	// Synthetic or aggregate directory?
	if u.isDir(p) {
		return u.openDir(p)
	}

	return nil, fs.ErrNotExist
}

func (u *unionFS) isDir(p string) bool {
	if p == "" {
		return true
	}
	if _, ok := u.dirs[p]; ok {
		return true
	}
	// A path is a dir if it is a parent of any mount or equals a dir mount root
	for _, m := range u.mounts {
		switch m.kind {
		case "file":
			if parentDir(m.virtual) == p || strings.HasPrefix(m.virtual, p+"/") {
				return true
			}
		case "dir":
			if p == m.virtual || strings.HasPrefix(m.virtual, p+"/") {
				return true
			}
		}
	}
	return false
}

func (u *unionFS) openDir(p string) (fs.File, error) {
	entries := map[string]fs.DirEntry{}

	// Helper to merge an entry by name (last-one-wins is fine; we sort later)
	merge := func(name string, de fs.DirEntry) {
		if name == "" || name == "." {
			return
		}
		if cur, ok := entries[name]; ok {
			// If we already have a dir, keep it.
			// If we have a file and the new one is a dir, replace it.
			if cur.IsDir() {
				return
			}
			if !de.IsDir() {
				return
			}
		}
		entries[name] = de
	}

	// Root: collect first-level segments from synthetic dirs and mounts
	if p == "" {
		for d := range u.dirs {
			seg := firstSeg(d)
			if seg != "" {
				merge(seg, dirEntry(seg))
			}
		}
		for _, m := range u.mounts {
			seg := firstSeg(m.virtual)
			if seg != "" {
				if m.kind == "file" {
					merge(seg, fileEntry{name: seg, info: fileInfoFromPath(m.source)})
				} else {
					merge(seg, dirEntry(seg))
				}
			}
		}
	} else {
		// Non-root directory
		prefix := p + "/"

		// Synthetic dirs that are at/under p -> add their immediate child segment
		for d := range u.dirs {
			if d == p || strings.HasPrefix(d, prefix) {
				if child := childSeg(p, d); child != "" {
					merge(child, dirEntry(child))
				}
			}
		}

		// Mounts contribute based on type and mode
		for _, m := range u.mounts {
			switch m.kind {
			case "file":
				if parentDir(m.virtual) == p {
					name := filepath.Base(m.virtual)
					merge(name, fileEntry{name: name, info: fileInfoFromPath(m.source)})
				}
			case "dir":
				if m.mode == DirMountKeep {
					// The mount root itself is a directory at m.virtual
					if parentDir(m.virtual) == p {
						name := filepath.Base(m.virtual)
						merge(name, dirEntry(name))
					}
					// If we are listing inside the keep mount root, delegate to source
					if p == m.virtual {
						if list, err := m.source.ReadDir(); err == nil {
							for _, de := range list {
								merge(de.Name(), realDirEntry(m.source, de))
							}
						}
					}
				} else { // DirMountStrip
					// The strip mount contributes its *contents* at p == m.virtual
					if p == m.virtual {
						if list, err := m.source.ReadDir(); err == nil {
							for _, de := range list {
								merge(de.Name(), realDirEntry(m.source, de))
							}
						}
					} else if strings.HasPrefix(p, prefixOf(m.virtual)) {
						// Browsing deeper inside a STRIP mount: map to real subdir and list
						rest := strings.TrimPrefix(p, m.virtual+"/")
						target := utils.NewPath(filepath.Join(m.source.String(), filepath.FromSlash(rest)))
						if target.IsDir() {
							if list, err := target.ReadDir(); err == nil {
								for _, de := range list {
									merge(de.Name(), realDirEntry(target, de))
								}
							}
						}
					}
				}
			}
		}
	}

	// Deterministic listing
	names := make([]string, 0, len(entries))
	for n := range entries {
		names = append(names, n)
	}
	sort.Strings(names)

	list := make([]fs.DirEntry, 0, len(names))
	for _, n := range names {
		list = append(list, entries[n])
	}

	return &memDir{
		name:    baseOrRoot(p),
		entries: list,
	}, nil
}

// -------------------- directory & entry shims --------------------

type memDir struct {
	name    string
	entries []fs.DirEntry
	pos     int
}

func (d *memDir) Stat() (fs.FileInfo, error) { return fakeDirInfo{name: d.name}, nil }
func (d *memDir) Read([]byte) (int, error)   { return 0, io.EOF }
func (d *memDir) Close() error               { return nil }

func (d *memDir) ReadDir(n int) ([]fs.DirEntry, error) {
	if d.pos >= len(d.entries) && n > 0 {
		return nil, io.EOF
	}
	if n <= 0 || d.pos+n > len(d.entries) {
		n = len(d.entries) - d.pos
	}
	out := d.entries[d.pos : d.pos+n]
	d.pos += n
	return out, nil
}

type fakeDirInfo struct{ name string }

func (f fakeDirInfo) Name() string       { return f.name }
func (f fakeDirInfo) Size() int64        { return 0 }
func (f fakeDirInfo) Mode() fs.FileMode  { return fs.ModeDir | 0o555 }
func (f fakeDirInfo) ModTime() time.Time { return time.Time{} }
func (f fakeDirInfo) IsDir() bool        { return true }
func (f fakeDirInfo) Sys() any           { return nil }

type dirEntry string

func (d dirEntry) Name() string               { return string(d) }
func (d dirEntry) IsDir() bool                { return true }
func (d dirEntry) Type() fs.FileMode          { return fs.ModeDir }
func (d dirEntry) Info() (fs.FileInfo, error) { return fakeDirInfo{name: string(d)}, nil }

type osDirEntry struct{ fs.DirEntry }

type fileEntry struct {
	name string
	info fs.FileInfo
}

func (f fileEntry) Name() string               { return f.name }
func (f fileEntry) IsDir() bool                { return false }
func (f fileEntry) Type() fs.FileMode          { return 0 }
func (f fileEntry) Info() (fs.FileInfo, error) { return f.info, nil }

// realDirEntry adapts an os.ReadDir entry under `base` into an fs.DirEntry.
func realDirEntry(base *utils.Path, de fs.DirEntry) fs.DirEntry {
	// For directories we can keep the DirEntry as-is; for files we try to stat once.
	if de.IsDir() {
		return osDirEntry{DirEntry: de}
	}
	// Build a concrete path for file info
	child := utils.NewPath(base.String() + "/" + de.Name())
	return fileEntry{name: de.Name(), info: fileInfoFromPath(child)}
}

func fileInfoFromPath(p *utils.Path) fs.FileInfo {
	if fi, err := p.Stat(); err == nil {
		return fi
	}
	// Fallback if stat fails
	return fakeFileInfo{name: p.Name(), size: 0}
}

type fakeFileInfo struct {
	name string
	size int64
}

func (f fakeFileInfo) Name() string       { return f.name }
func (f fakeFileInfo) Size() int64        { return f.size }
func (f fakeFileInfo) Mode() fs.FileMode  { return 0o444 }
func (f fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f fakeFileInfo) IsDir() bool        { return false }
func (f fakeFileInfo) Sys() any           { return nil }

// -------------------- small path helpers (virtual side only) --------------------

func cleanSlash(p string) string {
	if p == "" || p == "." || p == "./" {
		return ""
	}
	p = filepath.ToSlash(strings.TrimSpace(p))
	p = strings.Trim(p, "/")
	if p == "" {
		return ""
	}
	// Drop "." segments and any ".." (no traversal in virtual space)
	parts := strings.Split(p, "/")
	out := parts[:0]
	for _, s := range parts {
		if s == "" || s == "." {
			continue
		}
		if s == ".." {
			// refuse to include parent traversal
			continue
		}
		out = append(out, s)
	}
	return strings.Join(out, "/")
}

func parentDir(p string) string {
	if p == "" {
		return ""
	}
	if i := strings.LastIndexByte(p, '/'); i >= 0 {
		return p[:i]
	}
	return ""
}

func firstSeg(p string) string {
	if p == "" {
		return ""
	}
	if i := strings.IndexByte(p, '/'); i >= 0 {
		return p[:i]
	}
	return p
}

func childSeg(parent, full string) string {
	if full == parent {
		return ""
	}
	prefix := parent
	if prefix != "" {
		prefix += "/"
	}
	if !strings.HasPrefix(full, prefix) {
		return ""
	}
	rest := strings.TrimPrefix(full, prefix)
	return firstSeg(rest)
}

func baseOrRoot(p string) string {
	if p == "" {
		return "."
	}
	return filepath.Base(p)
}

func prefixOf(v string) string {
	if v == "" {
		return ""
	}
	return v + "/"
}
