package main

import (
	"fmt"
	"io"
	"log"
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

type Mode int

const (
	// ModeNormal is the standard mode, allowing file navigation.
	ModeNormal Mode = iota

	// ModeSearchEntry is search entry mode. Key presses are added
	// to the search string.
	ModeSearchEntry
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

	// mode is the viewer mode.
	mode Mode

	// regexp is the search regexp specified by the user.
	// Must only be modified by the event goroutine.
	regexp string

	// searchResults are the results for the current search.
	// They should be highlighted.
	searchResults searchResults
}

// lastLine returns the last line on the display.  It may be beyond the end
// of the file, if the file is short enough.
// mu must be held on call.
func (l *Lesser) lastLine() int64 {
	return l.line + int64(l.size.y) - 1
}

// Scroll describes a scroll action.
type Scroll int

const (
	// ScrollTop goes to the first line.
	ScrollTop Scroll = iota
	// ScrollBottom goes to the last line.
	ScrollBottom
	// ScrollUp goes up one line.
	ScrollUp
	// ScrollDown goes down one line.
	ScrollDown
	// ScrollUpPage goes up one page full.
	ScrollUpPage
	// ScrollDownPage goes down one page full.
	ScrollDownPage
	// ScrollUpHalfPage goes up one half page full.
	ScrollUpHalfPage
	// ScrollDownHalfPage goes down one half page full.
	ScrollDownHalfPage
)

// scroll moves the display based on the passed scroll action, without
// going past the beginning or end of the file.
func (l *Lesser) scroll(s Scroll) {
	l.mu.Lock()
	defer l.mu.Unlock()

	var dest int64
	var delta int64
	switch s {
	case ScrollTop:
		dest = 1
		delta = -1
	case ScrollBottom:
		// Just try to go to int64 max.
		dest = 0x7fffffffffffffff
		delta = 1
	case ScrollUp:
		dest = l.line - 1
		delta = -1
	case ScrollDown:
		dest = l.line + 1
		delta = 1
	case ScrollUpPage:
		dest = l.line - int64(l.size.y)
		delta = -1
	case ScrollDownPage:
		dest = l.line + int64(l.size.y)
		delta = 1
	case ScrollUpHalfPage:
		dest = l.line - int64(l.size.y)/2
		delta = -1
	case ScrollDownHalfPage:
		dest = l.line + int64(l.size.y)/2
		delta = 1
	}

	for l.line != dest && l.line+delta > 0 && l.src.LineExists(l.lastLine()+delta) {
		l.line += delta
	}
}

func (l *Lesser) handleEvent(e termbox.Event) {
	l.mu.Lock()
	mode := l.mode
	l.mu.Unlock()

	if e.Type != termbox.EventKey {
		return
	}

	c := e.Ch
	k := e.Key
	// Key is only valid is Ch is 0
	if c != 0 {
		k = 0
	}

	switch mode {
	case ModeNormal:
		switch {
		case c == 'q':
			l.events <- EventQuit
		case c == 'j':
			l.scroll(ScrollDown)
			l.events <- EventRefresh
		case c == 'k':
			l.scroll(ScrollUp)
			l.events <- EventRefresh
		case c == 'g':
			l.scroll(ScrollTop)
			l.events <- EventRefresh
		case c == 'G':
			l.scroll(ScrollBottom)
			l.events <- EventRefresh
		case k == termbox.KeyPgup:
			l.scroll(ScrollUpPage)
			l.events <- EventRefresh
		case k == termbox.KeyPgdn:
			l.scroll(ScrollDownPage)
			l.events <- EventRefresh
		case k == termbox.KeyCtrlU:
			l.scroll(ScrollUpHalfPage)
			l.events <- EventRefresh
		case k == termbox.KeyCtrlD:
			l.scroll(ScrollDownHalfPage)
			l.events <- EventRefresh
		case c == '/':
			l.mu.Lock()
			l.mode = ModeSearchEntry
			l.mu.Unlock()
			l.events <- EventRefresh
		}
	case ModeSearchEntry:
		switch {
		case k == termbox.KeyEnter:
			r := l.search(l.regexp)
			l.mu.Lock()
			l.mode = ModeNormal
			l.regexp = ""
			l.searchResults = r
			l.mu.Unlock()
			l.events <- EventRefresh
		default:
			l.mu.Lock()
			l.regexp += string(c)
			l.mu.Unlock()
			l.events <- EventRefresh
		}
	}
}

func (l *Lesser) listenEvents() {
	for {
		e := termbox.PollEvent()
		l.handleEvent(e)
	}
}

// searchResult describes search matches on a single line.
type searchResult struct {
	line    int64
	matches [][]int
	err     error
}

// matchesChar returns true if the search result contains a match for
// character index c.
func (s searchResult) matchesChar(c int) bool {
	for _, match := range s.matches {
		if len(match) < 2 {
			continue
		}

		if c >= match[0] && c < match[1] {
			return true
		}
	}
	return false
}

type searchResults []searchResult

// findLine finds the result for a specific line, if any.
// TODO(prattmic): Make a much more efficient data structure.
func (s searchResults) findLine(line int64) searchResult {
	for _, r := range s {
		if r.line == line {
			return r
		}
	}

	return searchResult{}
}

func (l *Lesser) search(s string) []searchResult {
	reg, err := regexp.Compile(s)
	if err != nil {
		// TODO(prattmic): display a better error
		log.Printf("regexp failed to compile: %v", err)
		return nil
	}

	results := make(chan searchResult, 100)

	searchLine := func(line int64) {
		r, err := l.src.SearchLine(reg, line)
		if err != nil {
			r = nil
		}

		results <- searchResult{
			line:    line,
			matches: r,
			err:     err,
		}
	}

	nextLine := int64(1)
	// Spawn initial search goroutines
	for ; nextLine <= 5; nextLine++ {
		go searchLine(nextLine)
	}

	all := make([]searchResult, 0)

	// Collect results, start searching next lines until we start
	// hitting EOF.
	for {
		ret := <-results
		all = append(all, ret)

		// We started hitting errors on a previous line,
		// there is no reason to search later lines.
		if ret.err != nil {
			break
		}

		go searchLine(nextLine)
		nextLine++
	}

	// Collect the remaing results.
	for int64(len(all)) < nextLine-1 {
		all = append(all, <-results)
	}

	return all
}

// statusBar renders the status bar.
// mu must be held on call.
func (l *Lesser) statusBar() {
	// The statusbar is just below the display.

	// Clear the statusbar
	for i := 0; i < l.size.x; i++ {
		termbox.SetCell(i, l.size.y, ' ', 0, 0)
	}

	switch l.mode {
	case ModeNormal:
		// Just a colon and a cursor
		termbox.SetCell(0, l.size.y, ':', 0, 0)
		termbox.SetCursor(1, l.size.y)
	case ModeSearchEntry:
		// / and search string
		termbox.SetCell(0, l.size.y, '/', 0, 0)
		for i, c := range l.regexp {
			termbox.SetCell(1+i, l.size.y, c, 0, 0)
		}
		termbox.SetCursor(1+len(l.regexp), l.size.y)
	}
}

func (l *Lesser) refreshScreen() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	for y := 0; y < l.size.y; y++ {
		buf := make([]byte, l.size.x)
		line := l.line + int64(y)

		_, err := l.src.ReadLine(buf, line)
		// EOF just means the line was shorter than the display.
		if err != nil && err != io.EOF {
			return err
		}

		highlight := l.searchResults.findLine(line)

		var tabOffset int
		for i, c := range buf {
			// If there are tabs, we may get to the end of the
			// display before we run out of characters.
			if i >= l.size.x {
				break
			}

			fg := termbox.ColorDefault
			bg := termbox.ColorDefault

			// Highlight matches
			if highlight.matchesChar(i) {
				fg = termbox.ColorBlack
				bg = termbox.ColorWhite
			}

			if c == '\t' {
				// Clear the tab spaces
				for j := 0; j < l.tabStop; j++ {
					termbox.SetCell(tabOffset+j, y, ' ', 0, 0)
				}
				tabOffset += l.tabStop - 1
			} else {
				termbox.SetCell(tabOffset+i, y, rune(c), fg, bg)
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
		mode:   ModeNormal,
	}
}
