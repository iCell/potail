package main

import (
	"github.com/gobwas/glob"
	"io/ioutil"
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

	fis, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, fi := range fis {
		if fi.IsDir() {
			continue
		}

		n := fi.Name()
		if g.Match(n) {
			p := filepath.Join(dir, n)
			f, err := os.Open(p)
			if err != nil {
				return nil, err
			}
			fs = append(fs, &File{
				File:   f,
				Modify: make(chan bool),
			})
		}
	}

	return fs, nil
}
