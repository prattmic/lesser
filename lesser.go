package main

import (
	"os"

	"github.com/nsf/termbox-go"
)

type Lesser struct {
	file *os.File
}

func (l Lesser) Run() {
	for {
		e := termbox.PollEvent()
		switch e.Type {
		case termbox.EventKey:
			switch e.Ch {
			case 'q':
				return
			}
		}
	}
}

func NewLesser(f *os.File) Lesser {
	return Lesser{file: f}
}
