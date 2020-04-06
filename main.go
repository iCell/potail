package main

import (
	"log"
	"path/filepath"
	"sync"
)

var watcher *Watcher
var once sync.Once

func main() {
	run()
}

func run() {
	once.Do(func() {
		w, err := NewWatcher(".", "test*")
		if err != nil {
			log.Fatal("create watcher failed", err)
		}
		watcher = w
	})

	tails := &Tails{}
	for _, file := range watcher.files {
		if file.IsWatched == false {
			continue
		}
		t, err := tails.Add(filepath.Join(watcher.Dir, file.Info.Name()))
		if err != nil {
			log.Fatal("create tail err", err)
		}
		go t.Tail()
	}

	go func() {
		for {
			select {
			case e := <-watcher.Event:
				switch e.Op {
				case Create:
					t, err := tails.Add(filepath.Join(watcher.Dir, e.File))
					if err != nil {
						panic(err)
					}
					go t.Tail()
				case Modify:
					tails.NotifyTail(e.File)
				case Rename:
					fallthrough
				case Remove:
					tails.CloseTail(e.File)
				}
			case err := <-watcher.Error:
				panic(err)
			}
		}
	}()

	watcher.Watch()
}
