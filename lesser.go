package main

import (
	"fmt"
	"os"

	"github.com/nsf/termbox-go"
)

type size struct {
	x int
	y int
}

type Event int

const (
	EventQuit Event = iota
)

type Lesser struct {
	file   *os.File
	events chan Event
	size   size
}

func (l Lesser) listenEvents() {
	for {
		e := termbox.PollEvent()
		switch e.Type {
		case termbox.EventKey:
			switch e.Ch {
			case 'q':
				l.events <- EventQuit
			}
		}
	}
}

func (l Lesser) fillScreen() error {
	buf := make([]byte, l.size.x*l.size.y)

	n, err := l.file.Read(buf)
	if err != nil {
		return err
	}

	buf = buf[:n]

	x, y := 0, 0

	for i := 0; i < len(buf); i++ {
		b := buf[i]

		// End of the screen
		if y >= l.size.y {
			break
		}

		// Next line
		if b == '\n' {
			y += 1
			x = 0
			continue
		}

		termbox.SetCell(x, y, rune(b), 0, 0)
		x += 1

		if x >= l.size.x {
			orig_y := y

			// Skip the rest of the line
			for j := i; j < len(buf); j++ {
				if buf[j] == '\n' {
					y++
					x = 0
					break
				}
			}

			// Couldn't find end of line
			if orig_y == y {
				break
			}
		}
	}

	termbox.Flush()

	return nil
}

func (l Lesser) Run() {
	go l.listenEvents()

	err := l.fillScreen()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to fill screen: %v\n", err)
		return
	}

	select {
	case e := <-l.events:
		switch e {
		case EventQuit:
			return
		}
	}
}

func NewLesser(f *os.File) Lesser {
	x, y := termbox.Size()

	return Lesser{
		file:   f,
		events: make(chan Event, 1),
		size:   size{x: x, y: y},
	}
}
