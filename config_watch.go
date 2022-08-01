package main

import (
	"fmt"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

func Watch(path string, configs chan<- *Config, errors chan<- error, done <-chan struct{}) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("can't create watcher: %w", err)
	}
	go func() {
	loop:
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					break loop
				}
				// getting rename, chmod, remove event when editing conf on linux
				if event.Op&(fsnotify.Write) > 0 && filepath.Base(event.Name) == filepath.Base(path) {
					c, err := ReadConfig(path)
					if err != nil {
						errors <- err
						continue loop
					}
					configs <- c
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					break loop
				}
				errors <- err
			case <-done:
				// select chooses the case pseudo randomly, should not be a problem here
				break loop
			}
		}
		// ignore close error
		watcher.Close()
	}()
	dir := filepath.Dir(path)
	// watch the dir atomic write of text editor gives rename, chmod, remove events; remove stops the watch
	err = watcher.Add(dir)
	if err != nil {
		return err
	}
	return nil
}
