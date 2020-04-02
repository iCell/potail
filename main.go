package main

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"log"
	"os"
	"path/filepath"
	"sync"
)

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

	dir := "."

	files, err := FilesFromDir(dir, "*test*")
	if err != nil {
		panic(err)
	}

	err = watcher.Add(".")
	if err != nil {
		panic(err)
	}
	for _, f := range files {
		err = watcher.Add(f.Path())
		if err != nil {
			panic(err)
		}

		t := NewTail(f)
		go t.Tail()
	}

	for {
		select {
		case event := <-watcher.Events:
			switch {
			case event.Op&fsnotify.Create == fsnotify.Create:
				fmt.Println("create", event.Name, event.String())
				f, err := os.Open(filepath.Join(dir, event.Name))
				if err != nil {
					panic(err)
				}
				// todo: should use lock
				file := &File{
					File:   f,
					Modify: make(chan bool),
				}
				files = append(files, file)
				t := NewTail(file)
				go t.Tail()
			case event.Op&fsnotify.Write == fsnotify.Write:
				fmt.Println("write")
				destFile := files.FindByName(event.Name)
				if destFile != nil && destFile.Follow {
					destFile.Modify <- true
				}
			case event.Op&fsnotify.Remove == fsnotify.Remove:
				fmt.Println("remove", event.Name)
			case event.Op&fsnotify.Rename == fsnotify.Rename:
				fmt.Println("rename")
			}
		case err := <-watcher.Errors:
			panic(err)
		}
	}
}
