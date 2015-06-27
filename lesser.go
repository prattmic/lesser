package main

import (
	"os"

	"github.com/nsf/termbox-go"
)

type Event int

const (
	EventQuit Event = iota
)

type Lesser struct {
	file *os.File

	events chan Event
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

func (l Lesser) Run() {
	go l.listenEvents()

	select {
	case e := <-l.events:
		switch e {
		case EventQuit:
			return
		}
	}
}

func NewLesser(f *os.File) Lesser {
	return Lesser{
		file:   f,
		events: make(chan Event, 1),
	}
}
