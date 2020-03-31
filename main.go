package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/fsnotify/fsnotify"
)

type File struct {
	File   *os.File
	Modify chan bool
}

func (f *File) Name() string {
	return "test"
}

type Files []*File

func (fs Files) FindByName(n string) *File {
	return fs[0]
	for _, f := range fs {
		fmt.Println("name is", f.File.Name())
	}
	return nil
}

func FilesFromDir(dir string) (Files, error) {
	f, err := os.Open("./test")
	if err != nil {
		return nil, err
	}
	return Files{&File{File: f, Modify: make(chan bool)}}, nil
}

var watcher *fsnotify.Watcher
var once sync.Once

func main() {
	once.Do(func() {
		wt, err := fsnotify.NewWatcher()
		if err != nil {
			log.Fatal("create watcher failed", err)
		}
		watcher = wt
	})
	defer watcher.Close()

	files, err := FilesFromDir("")
	if err != nil {
		panic(err)
	}

	for _, f := range files {
		err = watcher.Add(f.File.Name())
		if err != nil {
			panic(err)
		}
	}

	t := NewTail(files[0])

	go t.Tail()

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				switch {
				case event.Op&fsnotify.Write == fsnotify.Write:
					fmt.Println("write")
					destFile := files.FindByName(event.Name)
					if destFile != nil {
						destFile.Modify <- true
					}
				case event.Op&fsnotify.Remove == fsnotify.Remove:
					fmt.Println("remove")
				case event.Op&fsnotify.Rename == fsnotify.Rename:
					fmt.Println("rename")
				}
			case err := <-watcher.Errors:
				fmt.Println(err)
			}
		}
	}()

	http.ListenAndServe(":8080", nil)
}
