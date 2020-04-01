package main

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"log"
	"net/http"
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

	files, err := FilesFromDir(".")
	if err != nil {
		panic(err)
	}

	err = watcher.Add(".")
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
				case event.Op&fsnotify.Create == fsnotify.Create:
					fmt.Println("create")
					fmt.Println(event.Name)
				case event.Op&fsnotify.Write == fsnotify.Write:
					fmt.Println("write")
					destFile := files.FindByName(event.Name)
					if destFile != nil && destFile.Follow {
						destFile.Modify <- true
					}
				case event.Op&fsnotify.Remove == fsnotify.Remove:
					fallthrough
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
