package utils

import (
	"os"
	"path/filepath"
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
}

func NewPath(p string) *Path {
	return &Path{
		path: filepath.Clean(p),
	}
}

func (p Path) String() string {
	return p.path
}

func (p Path) Base() string {
	return filepath.Base(p.path)
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
	return NewPath(res), nil
}

func (p Path) Exists() (bool, error) {
	_, err := os.Stat(p.path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (p Path) IsDir() (bool, error) {
	res, err := os.Stat(p.path)
	if err != nil {
		return false, err
	}
	return res.IsDir(), nil
}

func (p Path) IsRegular() (bool, error) {
	res, err := os.Stat(p.path)
	if err != nil {
		return false, err
	}
	return res.Mode().IsRegular(), nil
}

func (p Path) Stat() (os.FileInfo, error) {
	return os.Stat(p.path)
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
