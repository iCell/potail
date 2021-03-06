package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var watcher *Watcher
var once sync.Once

type KLog struct {
	Log    string `json:"log"`
	Time   string `json:"time"`
	Stream string `json:"stream"`
}

func main() {
	once.Do(func() {
		w, err := NewWatcher(os.Getenv("DIR_PATH"), os.Getenv("GLOB_PATTERN"))
		if err != nil {
			log.Fatal("create watcher failed", err)
		}
		watcher = w
	})

	tails := NewTails()
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
					tails.CloseTail(e.File)
				case Remove:
					tails.CloseTail(e.File)
				}
			case err := <-watcher.Error:
				log.Fatalln("receive watcher error", err)
			}
		}
	}()

	go func() {
		for {
			select {
			case line := <-tails.Newline:
				var log KLog
				json.Unmarshal([]byte(line.Text), &log)
				if log.Stream == os.Getenv("LOG_STREAM") {
					fmt.Println(log.Log, line.FileName)
					msg := map[string]string{
						"text": fmt.Sprintf(
							"*log file*: %s\n*log*: %s\n*time*: %s",
							line.FileName, log.Log, log.Time,
						),
					}
					jsonStr, _ := json.Marshal(msg)
					req, _ := http.NewRequest("POST", os.Getenv("SLACK_WEBHOOK"), bytes.NewBuffer(jsonStr))
					req.Header.Set("Content-Type", "application/json")

					client := &http.Client{Timeout: time.Second * 5}
					client.Do(req)
				}
			}
		}
	}()

	watcher.Watch()
}
