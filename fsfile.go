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

type basicFs struct{}

func (fs *basicFs) Open(name string) (fs.File, error) {
	return os.Open(name)
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
		fsys = &basicFs{}
	}

	if fi.IsDir() {
		d, ok := f.(fs.ReadDirFile)
		if !ok {
			return nil, fmt.Errorf("directory not readable: %s", path)
		}

		ents, err := d.ReadDir(0)
		if err != nil {
			return nil, err
		}

		return &fsDirNode{fsys, path, ents, f, fi}, nil
	}

	return files.NewReaderStatFile(f, fi), nil
}

func (f *fsDirNode) Entries() files.DirIterator {
	return &serialIterator{
		path:   f.path,
		files:  f.files,
		filter: f.filter,
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

type serialIterator struct {
	files  []os.FileInfo
	path   string
	filter *files.Filter

	curName string
	curFile files.Node

	err error
}

func (it *serialIterator) Name() string {
	return it.curName
}

func (it *serialIterator) Node() files.Node {
	return it.curFile
}

func (it *serialIterator) Next() bool {
	// if there aren't any files left in the root directory, we're done
	if len(it.files) == 0 {
		return false
	}

	stat := it.files[0]
	it.files = it.files[1:]
	for it.filter.ShouldExclude(stat) {
		if len(it.files) == 0 {
			return false
		}

		stat = it.files[0]
		it.files = it.files[1:]
	}

	// open the next file
	filePath := filepath.ToSlash(filepath.Join(it.path, stat.Name()))

	// recursively call the constructor on the next file
	// if it's a regular file, we will open it as a ReaderFile
	// if it's a directory, files in it will be opened serially
	sf, err := files.NewSerialFileWithFilter(filePath, it.filter, stat)
	if err != nil {
		it.err = err
		return false
	}

	it.curName = stat.Name()
	it.curFile = sf
	return true
}

func (it *serialIterator) Err() error {
	return it.err
}

var _ files.Directory = &fsDirNode{}
var _ files.DirIterator = &fsDirIterator{}
