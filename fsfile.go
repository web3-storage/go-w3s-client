package w3s

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	files "github.com/ipfs/go-ipfs-files"
)

// fsDirNode implements files.Node and reads from an fs.File.
// No more than one file will be opened at a time.
type fsDirNode struct {
	fsys fs.FS
	path string
	ents []fs.DirEntry
	file fs.File
	info fs.FileInfo
}

type osFs struct{}

func (fs *osFs) Open(name string) (fs.File, error) {
	return os.Open(name)
}

func (fs *osFs) ReadDir(name string) ([]fs.DirEntry, error) {
	return os.ReadDir(name)
}

func (fs *osFs) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

func (fs *osFs) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

// NewFsNode creates a new files.Node that is backed by an fs.File.
func NewFsNode(path string, f fs.File, fi fs.FileInfo, fsys fs.FS) (files.Node, error) {
	if fi == nil {
		var err error
		fi, err = f.Stat()
		if err != nil {
			return nil, err
		}
	}

	if fsys == nil {
		fsys = &osFs{}
	}

	if fi.IsDir() {
		if d, ok := f.(fs.ReadDirFile); ok {
			ents, err := d.ReadDir(0)
			if err != nil {
				return nil, err
			}
			return &fsDirNode{fsys, path, ents, f, fi}, nil
		}

		if dfsys, ok := fsys.(fs.ReadDirFS); ok {
			ents, err := dfsys.ReadDir(path)
			if err != nil {
				return nil, err
			}
			return &fsDirNode{fsys, path, ents, f, fi}, nil
		}

		return nil, fmt.Errorf("directory not readable: %s", path)
	}

	return files.NewReaderStatFile(f, fi), nil
}

func (f *fsDirNode) Entries() files.DirIterator {
	return &fsDirIterator{
		fsys: f.fsys,
		path: f.path,
		ents: f.ents,
		file: f.file,
		info: f.info,
	}
}

func (f *fsDirNode) Close() error {
	return nil
}

func (f *fsDirNode) Stat() fs.FileInfo {
	return f.info
}

func (n *fsDirNode) Size() (int64, error) {
	if !n.info.IsDir() {
		// something went terribly, terribly wrong
		return 0, errors.New("not a directory")
	}

	var du int64
	err := fs.WalkDir(n.fsys, n.path, func(path string, e fs.DirEntry, err error) error {
		if err != nil || e == nil {
			return err
		}

		fi, err := e.Info()
		if err != nil {
			return err
		}

		if fi.Mode().IsRegular() {
			du += fi.Size()
		}

		return nil
	})

	return du, err
}

type fsDirIterator struct {
	fsys fs.FS
	path string
	ents []fs.DirEntry
	file fs.File
	info fs.FileInfo

	curName string
	curNode files.Node

	err error
}

func (it *fsDirIterator) Name() string {
	return it.curName
}

func (it *fsDirIterator) Node() files.Node {
	return it.curNode
}

func (it *fsDirIterator) Next() bool {
	// if there aren't any files left in the root directory, we're done
	if len(it.ents) == 0 {
		return false
	}

	ent := it.ents[0]
	it.ents = it.ents[1:]
	path := filepath.ToSlash(filepath.Join(it.path, ent.Name()))

	// open the next file
	f, err := it.fsys.Open(path)
	if err != nil {
		it.err = err
		return false
	}

	fi, err := ent.Info()
	if err != nil {
		it.err = err
		return false
	}

	n, err := NewFsNode(path, f, fi, it.fsys)
	if err != nil {
		it.err = err
		return false
	}

	it.curName = ent.Name()
	it.curNode = n
	return true
}

func (it *fsDirIterator) Err() error {
	return it.err
}

var _ files.Directory = &fsDirNode{}
var _ files.DirIterator = &fsDirIterator{}
