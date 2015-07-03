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

	// tabStop is the number of spaces per tab.
	tabStop int

	// events is used to notify the main goroutine of events.
	events chan Event

	// mu locks the fields below.
	mu sync.Mutex

	// size is the size of the file display.
	// There is a statusbar beneath the display.
	size size

	// line is the line number of the first line of the display.
	line int64
}

// lastLine returns the last line on the display.  It may be beyond the end
// of the file, if the file is short enough.
// mu must be held on call.
func (l *Lesser) lastLine() int64 {
	return l.line + int64(l.size.y) - 1
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
// refreshScreen must be called for the display to actually update.
func (l *Lesser) scrollDown() {
	l.mu.Lock()
	defer l.mu.Unlock()

	// You can only scroll down if the next line exists.
	if l.src.LineExists(l.lastLine() + 1) {
		l.line++
	}
}

// scrollTop moves to the first line.
// refreshScreen must be called for the display to actually update.
func (l *Lesser) scrollTop() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.line = 1
}

// scrollBottom moves to the last line.
// refreshScreen must be called for the display to actually update.
func (l *Lesser) scrollBottom() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.line = 1

	// TODO(prattmic): binary search
	for l.src.LineExists(l.lastLine() + 1) {
		l.line++
	}
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
			case 'g':
				l.scrollTop()
				l.events <- EventRefresh
			case 'G':
				l.scrollBottom()
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

// statusBar renders the status bar.
// mu must be held on call.
func (l *Lesser) statusBar() {
	// The statusbar is just below the display.
	// For now, it just has a colon, followed by the cursor.
	termbox.SetCell(0, l.size.y, ':', 0, 0)
	termbox.SetCursor(1, l.size.y)
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

		var tabOffset int
		for i, c := range buf {
			// If there are tabs, we may get to the end of the
			// display before we run out of characters.
			if i >= l.size.x {
				break
			}

			if c == '\t' {
				// Clear the tab spaces
				for j := 0; j < l.tabStop; j++ {
					termbox.SetCell(tabOffset+j, y, ' ', 0, 0)
				}
				tabOffset += l.tabStop - 1
			} else {
				termbox.SetCell(tabOffset+i, y, rune(c), 0, 0)
			}
		}
	}

	l.statusBar()

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

func NewLesser(f *os.File, ts int) Lesser {
	x, y := termbox.Size()

	return Lesser{
		src:     lineio.NewLineReader(f),
		tabStop: ts,
		// Save one line for statusbar.
		size:   size{x: x, y: y - 1},
		line:   1,
		events: make(chan Event, 1),
	}
}
