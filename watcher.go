package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"time"
)

type Event int

const (
	Create Event = iota
	Rename
	Remove
	Chmod
	Modify
)

type FS struct {
	sync.Mutex
	Dir   string
	Files map[string]os.FileInfo
}

func NewFromDir(dir string) (*FS, error) {
	fis, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	i := make(map[string]os.FileInfo)
	for _, fi := range fis {
		i[fi.Name()] = fi
	}

	return &FS{
		Dir:   dir,
		Files: i,
	}, nil
}

func (fs *FS) Watch() {
	ticker := time.NewTicker(time.Millisecond * 250)
	for {
		select {
		case <-ticker.C:
			fs.pollDirEvents()
		}
	}
}

func (fs *FS) pollDirEvents() error {
	fs.Lock()
	defer fs.Unlock()

	current := make(map[string]os.FileInfo)
	fis, err := ioutil.ReadDir(fs.Dir)
	if err != nil {
		return err
	}
	for _, fi := range fis {
		current[fi.Name()] = fi
	}

	creates := make(map[string]os.FileInfo)
	removes := make(map[string]os.FileInfo)

	// check for removed files
	for name, info := range fs.Files {
		if _, found := current[name]; !found {
			removes[name] = info
			delete(fs.Files, name)
			fmt.Println("remove", name, time.Now().Unix())
		}
	}

	// check for created files
	for name, info := range current {
		old, found := fs.Files[name]
		if !found {
			creates[name] = info
			fs.Files[name] = info
			fmt.Println("create", info.Name(), time.Now().Unix())
			continue
		}
		if old.Mode() != info.Mode() {
			// event chmod
			delete(fs.Files, name)
			fs.Files[name] = info
			fmt.Println("chmod", info.Name())
		}
	}

	for name1, info1 := range removes {
		for name2, info2 := range creates {
			if os.SameFile(info1, info2) {
				// event rename
				fmt.Printf("rename from %s to %s", info1.Name(), info2.Name())

				delete(removes, name1)
				delete(creates, name2)
			}
		}
	}

	return nil
}
