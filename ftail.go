/*
ftail monitor a file's increasing output continuously.

Example:

	ft := NewFTailer("/var/log/messages", true)
	c := make(chan int)
	go ft.Tail(c)

		for line := range c {
			fmt.Println(line)
		}
*/
package ftail

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/fsnotify/fsnotify"
)

// error codes
const (
	ErrWatcherInit = 1 << iota
	ErrWatcherAdd
	ErrWatcherClosed
	ErrWatcherErr
	ErrFileOpen
	ErrFileRead
	ErrFileSeek
)

type FTailer struct {
	Name    string
	file    *os.File
	watcher *fsnotify.Watcher
	follow  bool // whether to keep running if the file is recreated/truncated/...
}

func NewFTailer(name string, follow bool) *FTailer {
	ft := FTailer{
		Name:   name,
		follow: follow,
	}
	ft.initWatcher()
	ft.initFile()
	ft.seekEnd()
	return &ft
}

func (ft *FTailer) initWatcher() {
	if ft.watcher != nil {
		ft.watcher.Close()
		ft.watcher = nil
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fail to create file watcher: %s", err.Error())
		os.Exit(ErrWatcherInit)
	}

	err = watcher.Add(ft.Name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fail to watch %s: %s", ft.Name, err.Error())
		os.Exit(ErrWatcherAdd)
	}
	ft.watcher = watcher
}

func (ft *FTailer) initFile() {
	if ft.file != nil {
		ft.file.Close()
		ft.file = nil
	}
	f, err := os.Open(ft.Name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fail to open file %s: %s", ft.Name, err.Error())
		os.Exit(ErrWatcherInit)
	}
	ft.file = f
}

func (ft *FTailer) seekEnd() int64 {
	pos, err := ft.file.Seek(0, io.SeekEnd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fail to seek to the end of file %s: %s", ft.Name, err.Error())
		os.Exit(ErrFileSeek)
	}
	return pos
}

func (ft *FTailer) Tail(c chan<- string) {
	defer close(c)
	for {
		select {
		case event, ok := <-ft.watcher.Events:
			if !ok {
				fmt.Fprintf(os.Stderr, "fail to get more file system notifications from file %s", ft.Name)
				os.Exit(ErrWatcherClosed)
			}
			// fmt.Println(event.Op.String())
			if event.Has(fsnotify.Write) {
				scanner := bufio.NewScanner(ft.file)
				for scanner.Scan() {
					c <- scanner.Text()
				}
				if err := scanner.Err(); err != io.EOF && err != nil {
					fmt.Fprintf(os.Stderr, "fail to read from file %s: %s", ft.Name, err.Error())
					os.Exit(ErrFileRead)
				} else {
					ft.seekEnd()
				}
			}
			if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) || event.Has(fsnotify.Chmod) {
				if ft.follow {
					ft.initWatcher()
					ft.initFile()
					ft.seekEnd()
				} else {
					os.Exit(0)
				}
			}
		case err, ok := <-ft.watcher.Errors:
			if !ok {
				fmt.Fprintf(os.Stderr, "fail to get more file system notifications from file %s", ft.Name)
				os.Exit(ErrWatcherClosed)
			}
			fmt.Fprintf(os.Stderr, "hit an error while monitoring file %s: %s", ft.Name, err.Error())
			os.Exit(ErrWatcherErr)
		}
	}
}
