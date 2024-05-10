package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

var globalWatcher *fsnotify.Watcher

func main() {
	// Create new watcher.
	watcher, err := fsnotify.NewWatcher()
	globalWatcher = watcher

	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	configFileMap := getConfigFileMap()

	// Start listening for events.
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Has(fsnotify.Write) {
					file := filepath.Base(event.Name)
					pauseFileWatcher(configFileMap[file])
					replaceAllConfig(event.Name, configFileMap[file])
					resumeFileWatcher(configFileMap[file])
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	// Block main goroutine forever.
	<-make(chan struct{})
}

func pauseFileWatcher(filePaths []string) {
	for _, path := range filePaths {
		if err := globalWatcher.Remove(path); err != nil {
			log.Println("ERROR: Couldn't pause FileWatcher:", err)
		}

	}
}

func resumeFileWatcher(filePaths []string) {
	for _, path := range filePaths {
		if err := globalWatcher.Add(path); err != nil {
			log.Println("ERROR: Couldn't resume FileWatcher:", err)
		}
	}
}

func getConfigFileMap() map[string][]string {
	vaults := os.Getenv("OBSIDIAN_VAULTS")
	configFileMap := make(map[string][]string)
	err := filepath.Walk(vaults, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Fatal(err)
		}
		file := filepath.Base(path)
		dir := filepath.Dir(path)
		parent := filepath.Base(dir)

		if parent == ".obsidian" && file != "workspace.json" {
			configFileMap[file] = append(configFileMap[file], path)
			if err = globalWatcher.Add(path); err != nil {
				log.Fatal(err)
			}
		}
		return nil
	})

	if err != nil {
		log.Fatal(err)
	}
	return configFileMap
}

func replaceAllConfig(source string, filePaths []string) {
	b, err := os.ReadFile(source)
	if err != nil {
		log.Fatal(err)
	}
	for _, path := range filePaths {
		// exclude source to avoid infinity loop
		if source == path {
			continue
		}
		if err := os.WriteFile(path, b, 0644); err != nil {
			log.Fatal(err)
		}
	}
}
