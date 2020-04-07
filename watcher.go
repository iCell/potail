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

const pollInterval = 250 * time.Millisecond

type fileInfo struct {
	Info      os.FileInfo
	IsWatched bool
}

type Watcher struct {
	sync.Mutex
	filter glob.Glob
	notify *fsnotify.Watcher
	files  map[string]fileInfo
	Dir    string
	Event  chan Event
	Error  chan error
}

func NewWatcher(dir, pattern string) (*Watcher, error) {
	g, err := glob.Compile(pattern)
	if err != nil {
		return nil, err
	}

	infos, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	fis := make(map[string]fileInfo)
	for _, info := range infos {
		fis[info.Name()] = fileInfo{
			Info:      info,
			IsWatched: g.Match(info.Name()),
		}
	}

	notify, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	for _, fi := range fis {
		// todo: support recursive directory
		if fi.Info.IsDir() || fi.IsWatched == false {
			continue
		}
		notify.Add(filepath.Join(dir, fi.Info.Name()))
	}

	return &Watcher{
		files:  fis,
		filter: g,
		notify: notify,
		Dir:    dir,
		Event:  make(chan Event),
		Error:  make(chan error),
	}, nil
}

func (w *Watcher) Watch() {
	timer := time.NewTimer(pollInterval)

	for {
		select {
		case <-timer.C:
			w.pollDirEvents()
			timer.Reset(pollInterval)
		case e := <-w.notify.Events:
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
			removes[name] = info.Info
		}
	}

	// check for created files
	for name, info := range current {
		old, found := w.files[name]
		if !found {
			creates[name] = info
			continue
		}
		if old.Info.Mode() != info.Mode() {
			delete(w.files, name)
			w.files[name] = fileInfo{
				Info:      info,
				IsWatched: w.files[name].IsWatched,
			}
			w.Event <- Event{
				Op:   Chmod,
				File: name,
			}
		}
	}

	// todo: rename not working
	for oldN, oldI := range removes {
		for newN, newI := range creates {
			if os.SameFile(oldI, newI) {
				w.Event <- Event{
					Op:   Rename,
					File: oldN,
				}
				// todo: watch renamed file if new name conforms to the pattern
				w.files[newN] = fileInfo{
					Info:      newI,
					IsWatched: false,
				}
				if w.files[oldN].IsWatched {
					w.notify.Remove(oldN)
				}
				delete(w.files, oldN)

				delete(removes, oldN)
				delete(creates, newN)
			}
		}
	}

	for name, _ := range removes {
		if w.files[name].IsWatched {
			w.notify.Remove(name)
		}
		delete(w.files, name)
		w.Event <- Event{
			Op:   Remove,
			File: name,
		}
	}
	for name, info := range creates {
		match := w.filter.Match(name)
		w.files[name] = fileInfo{
			Info:      info,
			IsWatched: match,
		}
		if match {
			w.notify.Add(filepath.Join(w.Dir, name))
		}
		w.Event <- Event{
			Op:   Create,
			File: name,
		}
	}
}
