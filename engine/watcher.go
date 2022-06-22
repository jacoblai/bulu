package engine

import (
	"encoding/json"
	"github.com/fsnotify/fsnotify"
	"github.com/jacoblai/bulu/model"
	"log"
	"os"
)

func (e *Engine) Watcher(fpath string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					bts, err := os.ReadFile(event.Name)
					if err == nil {
						var conf model.Config
						err = json.Unmarshal(bts, &conf)
						if err == nil {
							_ = e.InitNodes(conf)
							log.Println("bulu config file was updated...")
						}
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.Add(fpath)
	if err != nil {
		log.Fatal(err)
	}
}
