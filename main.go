package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
)

var watcher *Watcher
var once sync.Once

func main() {

}

func run() {
	once.Do(func() {
		w, err := NewWatcher(".", "test")
		if err != nil {
			log.Fatal("create watcher failed", err)
		}
		watcher = w
	})

	for _, fi := range watcher.watchedFiles {
		f, err := os.Open(filepath.Join(watcher.Dir, fi.Name()))
		if err != nil {
			log.Fatal("open file err", err)
		}
		t := NewTail(f)
		go t.Tail()
	}

	go func() {
		for {
			select {
			case e := <-watcher.Event:
				switch e.Op {
				case Create:
					fmt.Println("create", e.File)
					f, err := os.Open(filepath.Join(watcher.Dir, e.Name))
					if err != nil {
						panic(err)
					}
					// todo: should use lock
					t := NewTail(f)
					go t.Tail()
				case Modify:
					fmt.Print("modify", e.File)
				case Rename:
					fmt.Print("rename", e.File)
				case Remove:
					fmt.Print("remove", e.File)
				case Chmod:
					fmt.Print("change mod", e.File)
				}
			case err := <-watcher.Error:
				panic(err)
			}
		}
	}()

	//destFile := files.FindByName(event.Name)
	//if destFile != nil && destFile.Follow {
	//	destFile.Modify <- true
	//}
}
