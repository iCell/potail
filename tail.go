package main

import (
	"bufio"
	"fmt"
	"io"
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

type Tail struct {
	sync.Mutex
	file   *File
	reader *bufio.Reader
}

func NewTail(f *File) *Tail {
	return &Tail{
		file:   f,
		reader: bufio.NewReader(f.File),
	}
}

func (t *Tail) SeekToEnd() {
	t.file.File.Seek(0, io.SeekEnd)
}

func (t *Tail) Close() {
	t.file.File.Close()
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
	defer t.Close()

	for {
		line, err := t.readLine()
		switch {
		case err == io.EOF:
			t.sendLine(line)

			//pos, _ := t.file.File.Seek(0, io.SeekCurrent)

			t.file.Follow = true

			isContinue := false
			for _ = range t.file.Modify {
				isContinue = true
				break
			}

			if isContinue {
				continue
			}

			return
		case err != nil:
			panic(err)
		}
		t.sendLine(line)
	}
}
