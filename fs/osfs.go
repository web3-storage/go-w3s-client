package fs

import (
	"io/fs"
	"os"
)

// OsFs is an implementation of fs.FS backed by os.
type OsFs struct{}

func (fs *OsFs) Open(name string) (fs.File, error) {
	return os.Open(name)
}

func (fs *OsFs) ReadDir(name string) ([]fs.DirEntry, error) {
	return os.ReadDir(name)
}

func (fs *OsFs) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

func (fs *OsFs) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}
