package main

import (
	"fmt"
	"io"
	"os"

	"github.com/nsf/termbox-go"

	"github.com/prattmic/lesser/lineio"
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
	src    lineio.LineReader
	events chan Event
	size   size
}

func (l *Lesser) listenEvents() {
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

func (l *Lesser) fillScreen() error {
	for y := 0; y < l.size.y; y++ {
		buf := make([]byte, l.size.x)

		_, err := l.src.ReadLine(buf, y+1)
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
		src:    lineio.NewLineReader(f),
		events: make(chan Event, 1),
		size:   size{x: x, y: y},
	}
}
