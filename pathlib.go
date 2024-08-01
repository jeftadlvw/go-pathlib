package pathlib

import "path/filepath"

type Path struct {
	path string
}

func NewPath(path string) *Path {
	p := &Path{path: path}
	p.clean()

	return p
}

func (p *Path) String() string {
	return p.path
}

func (p *Path) clean() {
	p.path = filepath.Clean(p.path)
}
