package main

import (
	"github.com/gobwas/glob"
	"os"
	"path/filepath"
)

const newLine = '\n'

type Line struct {
	Text string
}

func (line *Line) IsEmpty() bool {
	return len(line.Text) == 0
}

type File struct {
	File   *os.File
	Modify chan bool
	Follow bool
}

func (f *File) Name() string {
	return filepath.Base(f.Path())
}

func (f *File) Path() string {
	return f.File.Name()
}

type Files []*File

func (fs Files) FindByName(n string) *File {
	for _, f := range fs {
		if n == f.Name() {
			return f
		}
	}
	return nil
}

func (fs Files) RemoveByName(n string) {
	watcher.Remove(n)
	f := fs.FindByName(n)
	f.File.Close()
}

func FilesFromDir(dir string, pattern string) (Files, error) {
	var fs Files
	g := glob.MustCompile(pattern)

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() == false {
			base := filepath.Base(path)
			if g.Match(base) {
				f, err := os.Open(path)
				if err != nil {
					return err
				}
				fs = append(fs, &File{
					File:   f,
					Modify: make(chan bool),
				})
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return fs, nil
}
