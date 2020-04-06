package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const newLine = '\n'

type Line struct {
	Text string
}

func (line *Line) IsEmpty() bool {
	return len(line.Text) == 0
}

type Tails struct {
	sync.Mutex
	tails map[string]*Tail
}

func NewTails() *Tails {
	return &Tails{
		tails: make(map[string]*Tail),
	}
}

func (ts *Tails) Add(path string) (*Tail, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	tail := NewTail(f)
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

type Tail struct {
	sync.Mutex
	file       *os.File
	reader     *bufio.Reader
	isFollowed bool
	modify     chan struct{}
}

func NewTail(f *os.File) *Tail {
	return &Tail{
		file:   f,
		reader: bufio.NewReader(f),
		modify: make(chan struct{}),
	}
}

func (t *Tail) SeekToEnd() {
	t.file.Seek(0, io.SeekEnd)
}

func (t *Tail) readLine() (*Line, error) {
	t.Lock()
	defer t.Unlock()

	line, err := t.reader.ReadString(newLine)
	if err != nil {
		return &Line{Text: line}, err
	}
	line = strings.TrimRight(line, string(newLine))
	return &Line{Text: line}, nil
}

func (t *Tail) sendLine(line *Line) {
	if line.IsEmpty() {
		return
	}
	fmt.Println(line.Text)
}

func (t *Tail) Tail() {
	defer t.file.Close()

	for {
		line, err := t.readLine()
		switch {
		case err == io.EOF:
			t.sendLine(line)

			t.isFollowed = true

			next := false
			for _ = range t.modify {
				next = true
				break
			}

			if next {
				continue
			}

			return
		case err != nil:
			panic(err)
		}
		t.sendLine(line)
	}
}
