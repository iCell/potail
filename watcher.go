package main

import (
	"github.com/fsnotify/fsnotify"
	"github.com/gobwas/glob"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Event struct {
	Op   Operation
	File string
}

type Operation int

const (
	Create Operation = iota
	Rename
	Remove
	Chmod
	Modify
)

type Watcher struct {
	sync.Mutex
	inotify      *fsnotify.Watcher
	files        map[string]os.FileInfo
	watchedFiles map[string]os.FileInfo
	Dir          string
	Event        chan Event
	Error        chan error
}

func NewWatcher(dir, pattern string) (*Watcher, error) {
	g, err := glob.Compile(pattern)
	if err != nil {
		return nil, err
	}

	fis, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	fs := make(map[string]os.FileInfo)
	wfs := make(map[string]os.FileInfo)
	for _, fi := range fis {
		fs[fi.Name()] = fi
		if g.Match(fi.Name()) {
			wfs[fi.Name()] = fi
		}
	}

	ionotify, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	for _, wf := range wfs {
		// todo: support recursive directory
		if wf.IsDir() {
			continue
		}
		ionotify.Add(filepath.Join(dir, wf.Name()))
	}

	return &Watcher{
		files:        fs,
		watchedFiles: wfs,
		inotify:      ionotify,
		Dir:          dir,
		Event:        make(chan Event),
	}, nil
}

func (w *Watcher) Watch() {
	ticker := time.NewTicker(time.Millisecond * 250)
	for {
		select {
		case <-ticker.C:
			w.pollDirEvents()
		case e := <-w.inotify.Events:
			if e.Op&fsnotify.Write == fsnotify.Write {
				w.Event <- Event{
					Op:   Modify,
					File: e.Name,
				}
			}
		case err := <-w.Error:
			w.Error <- err
		}
	}
}

func (w *Watcher) pollDirEvents() {
	w.Lock()
	defer w.Unlock()

	current := make(map[string]os.FileInfo)
	fis, err := ioutil.ReadDir(w.Dir)
	if err != nil {
		w.Error <- err
		return
	}
	for _, fi := range fis {
		current[fi.Name()] = fi
	}

	creates := make(map[string]os.FileInfo)
	removes := make(map[string]os.FileInfo)

	// check for removed files
	for name, info := range w.files {
		if _, found := current[name]; !found {
			removes[name] = info
			delete(w.files, name)
			w.Event <- Event{
				Op:   Remove,
				File: name,
			}
		}
	}

	// check for created files
	for name, info := range current {
		old, found := w.files[name]
		if !found {
			creates[name] = info
			w.files[name] = info
			w.Event <- Event{
				Op:   Create,
				File: name,
			}
			continue
		}
		if old.Mode() != info.Mode() {
			delete(w.files, name)
			w.files[name] = info
			w.Event <- Event{
				Op:   Chmod,
				File: name,
			}
		}
	}

	for name1, info1 := range removes {
		for name2, info2 := range creates {
			if os.SameFile(info1, info2) {
				w.Event <- Event{
					Op:   Rename,
					File: name1,
				}

				delete(removes, name1)
				delete(creates, name2)
			}
		}
	}
}
