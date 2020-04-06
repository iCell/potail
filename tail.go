package main

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const newLine = '\n'

type Line struct {
	Text     string
	FileName string
}

func (line *Line) IsEmpty() bool {
	return len(line.Text) == 0
}

type Tail struct {
	sync.Mutex
	file       *os.File
	reader     *bufio.Reader
	isFollowed bool
	modify     chan struct{}
	newline    chan Line
}

func NewTail(f *os.File, l chan Line) *Tail {
	return &Tail{
		file:    f,
		reader:  bufio.NewReader(f),
		modify:  make(chan struct{}),
		newline: l,
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
	return &Line{Text: line, FileName: filepath.Base(t.file.Name())}, nil
}

func (t *Tail) sendLine(line *Line) {
	if line.IsEmpty() {
		return
	}
	t.newline <- *line
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
