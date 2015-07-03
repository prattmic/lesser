package main

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"sync"

	"github.com/nsf/termbox-go"

	"github.com/prattmic/lesser/lineio"
)

type size struct {
	x int
	y int
}

type Event int

const (
	// EventQuit requests an application exit.
	EventQuit Event = iota

	// EventRefresh requests a display refresh.
	EventRefresh
)

type Lesser struct {
	// src is the source file being displayed.
	src lineio.LineReader

	// events is used to notify the main goroutine of events.
	events chan Event

	// mu locks the fields below.
	mu sync.Mutex

	// size is the size of the display.
	size size

	// line is the line number of the first line of the display.
	line int64
}

// scrollUp moves the display up (i.e., decrements the first line number).
// You cannot scroll beyond the beginning of the file.
// refreshScreen must be called for the display to actually update.
func (l *Lesser) scrollUp() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.line > 1 {
		l.line--
	}
}

// scrollDown moves the display down (i.e., increments the first line number).
// FIXME(prattmic): Nothing prevents scrolling beyond the end of the file.
// refreshScreen must be called for the display to actually update.
func (l *Lesser) scrollDown() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.line++
}

func (l *Lesser) listenEvents() {
	for {
		e := termbox.PollEvent()
		switch e.Type {
		case termbox.EventKey:
			switch e.Ch {
			case 'q':
				l.events <- EventQuit
			case 'j':
				l.scrollDown()
				l.events <- EventRefresh
			case 'k':
				l.scrollUp()
				l.events <- EventRefresh
			case 's':
				l.search()
			}
		}
	}
}

type searchResult struct {
	line    int64
	matches [][]int
}

func (l *Lesser) search() {
	// TODO: search more than a fixed regexp
	reg := regexp.MustCompile("line")

	results := make(chan searchResult, 100)

	searchLine := func(line int64) {
		r, err := l.src.SearchLine(reg, line)
		if err != nil {
			r = nil
		}

		results <- searchResult{
			line:    line,
			matches: r,
		}
	}

	// TODO: search more than the first hundred lines :)
	for i := int64(1); i <= 100; i++ {
		go searchLine(i)
	}

	all := make([]searchResult, 0)
	for len(all) < 100 {
		all = append(all, <-results)
	}

	fmt.Fprintf(os.Stderr, "Results: %+v\n", all)
}

func (l *Lesser) refreshScreen() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	for y := 0; y < l.size.y; y++ {
		buf := make([]byte, l.size.x)

		_, err := l.src.ReadLine(buf, l.line+int64(y))
		// EOF just means the line was shorter than the display.
		if err != nil && err != io.EOF {
			return err
		}

		for i, c := range buf {
			termbox.SetCell(i, y, rune(c), 0, 0)
		}
	}

	termbox.Flush()

	return nil
}

func (l *Lesser) Run() {
	go l.listenEvents()

	err := l.refreshScreen()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to refresh screen: %v\n", err)
		return
	}

	for {
		e := <-l.events

		switch e {
		case EventQuit:
			return
		case EventRefresh:
			err = l.refreshScreen()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to refresh screen: %v\n", err)
				return
			}
		}
	}
}

func NewLesser(f *os.File) Lesser {
	x, y := termbox.Size()

	return Lesser{
		src:    lineio.NewLineReader(f),
		size:   size{x: x, y: y},
		line:   1,
		events: make(chan Event, 1),
	}
}
