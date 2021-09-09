package fs

import (
	"io"
	"io/fs"
	"time"
)

type Dir struct {
	name    string
	files   []fs.File
	modTime time.Time
}

// NewDir creates a new fs.File compatible "directory" from the passed slice of
// fs.File.
//
// It implements fs.ReadDir and directory entries returned implement Opener,
// allowing access to the underlying files passed here.
//
// You can add nested directories (of the same type) to this directory. Adding
// filesystem directories to this directory is NOT supported.
func NewDir(name string, files []fs.File) *Dir {
	modTime := time.Now()
	return &Dir{name, files, modTime}
}

func (d *Dir) Stat() (fs.FileInfo, error) {
	return &DirInfo{d.name, d.modTime}, nil
}

func (d *Dir) Read([]byte) (int, error) {
	return 0, io.EOF
}

func (d *Dir) Close() error {
	return nil
}

// ReadDir reads the contents of the directory and returns a slice of up to n
// DirEntry values in directory order.
func (d *Dir) ReadDir(n int) ([]fs.DirEntry, error) {
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

var _ fs.File = (*Dir)(nil)
var _ fs.ReadDirFile = (*Dir)(nil)

type DirInfo struct {
	name    string
	modTime time.Time
}

func (di *DirInfo) IsDir() bool {
	return true
}

func (di *DirInfo) Name() string {
	return di.name
}

func (di *DirInfo) Size() int64 {
	return 0
}

func (di *DirInfo) ModTime() time.Time {
	return di.modTime
}

func (di *DirInfo) Mode() fs.FileMode {
	return fs.ModeDir | 0555
}

func (di *DirInfo) Sys() interface{} {
	return nil
}

var _ fs.FileInfo = (*DirInfo)(nil)

type Opener interface {
	Open() (fs.File, error)
}

type DirEntry struct {
	file fs.File
	info fs.FileInfo
}

func (de *DirEntry) Name() string {
	return de.info.Name()
}

func (de *DirEntry) IsDir() bool {
	return de.info.IsDir()
}

func (de *DirEntry) Type() fs.FileMode {
	return de.info.Mode().Type()
}

func (de *DirEntry) Info() (fs.FileInfo, error) {
	return de.info, nil
}

func (de *DirEntry) Open() (fs.File, error) {
	return de.file, nil
}

var _ fs.DirEntry = (*DirEntry)(nil)
var _ Opener = (*DirEntry)(nil)
