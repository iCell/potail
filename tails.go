package main

import (
	"os"
	"path/filepath"
	"sync"
)

type Tails struct {
	sync.Mutex
	tails   map[string]*Tail
	Newline chan Line
}

func NewTails() *Tails {
	return &Tails{
		tails:   make(map[string]*Tail),
		Newline: make(chan Line),
	}
}

func (ts *Tails) Add(path string) (*Tail, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	tail := NewTail(f, ts.Newline)
	ts.Lock()
	ts.tails[filepath.Base(f.Name())] = tail
	ts.Unlock()

	return tail, nil
}

func (ts *Tails) NotifyTail(name string) {
	destTail := ts.destTail(name)
	if destTail == nil {
		return
	}
	destTail.modify <- struct{}{}
}

func (ts *Tails) CloseTail(name string) {
	destTail := ts.destTail(name)
	if destTail == nil {
		return
	}
	close(destTail.modify)
	ts.Lock()
	delete(ts.tails, filepath.Base(destTail.file.Name()))
	ts.Unlock()
}

func (ts *Tails) destTail(name string) *Tail {
	ts.Lock()
	defer ts.Unlock()
	for key, t := range ts.tails {
		if key == name {
			return t
		}
	}
	return nil
}
