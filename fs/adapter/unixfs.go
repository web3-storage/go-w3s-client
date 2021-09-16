package adapter

import (
	"errors"
	"io/fs"
	"time"

	files "github.com/ipfs/go-ipfs-files"
)

type unixFsFile struct {
	path string
	info fs.FileInfo
	node files.Node
}

// NewFile creates an fs.File that is backed by an IPFS files.Node.
func NewFile(name string, node files.Node) (fs.File, error) {
	size, err := node.Size()
	if err != nil {
		return nil, err
	}
	_, isDir := node.(files.Directory)
	return &unixFsFile{
		info: &unixFsFileInfo{
			name:    name,
			size:    size,
			modTime: time.Now(),
			isDir:   isDir,
		},
		node: node,
	}, nil
}

func (f *unixFsFile) Stat() (fs.FileInfo, error) {
	return f.info, nil
}

func (f *unixFsFile) Read(p []byte) (int, error) {
	if ff, ok := f.node.(files.File); ok {
		return ff.Read(p)
	}
	return 0, errors.New("file not readable")
}

func (f *unixFsFile) Close() error {
	return f.node.Close()
}

func (f *unixFsFile) ReadDir(n int) ([]fs.DirEntry, error) {
	fd, isDir := f.node.(files.Directory)
	if !isDir {
		return nil, errors.New("not a directory")
	}

	var ents []fs.DirEntry
	it := fd.Entries()
	for it.Next() {
		node := it.Node()
		NewFile(it.Name(), node)
	}
	if it.Err() != nil {
		return nil, it.Err()
	}

	fl := len(d.files)
	if n <= 0 || n > fl {
		n = fl
	}
	var ents []fs.DirEntry
	for i := 0; i < n; i++ {
		inf, err := d.files[i].Stat()
		if err != nil {
			return nil, err
		}
		ents = append(ents, &DirEntry{file: d.files[i], info: inf})
	}
	return ents, nil
}

type unixFsFileInfo struct {
	name    string
	size    int64
	modTime time.Time
	isDir   bool
}

func (i *unixFsFileInfo) Name() string {
	return i.name
}

func (i *unixFsFileInfo) Size() int64 {
	return i.size
}

func (i *unixFsFileInfo) Mode() fs.FileMode {
	if i.isDir {
		return fs.ModeDir | 0555
	}
	return fs.ModePerm
}

func (i *unixFsFileInfo) ModTime() time.Time {
	return i.modTime
}

func (i *unixFsFileInfo) IsDir() bool {
	return i.Mode().IsDir()
}

func (i *unixFsFileInfo) Sys() interface{} {
	return nil
}
