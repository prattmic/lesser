package main

import (
	"fmt"
	"os"

	"github.com/nsf/termbox-go"
)

func writeString(s string) {
	cells := termbox.CellBuffer()

	for i, c := range s {
		cells[i].Ch = c
	}

	termbox.Flush()
}

func main() {
	err := termbox.Init()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to init: %v\n", err)
		os.Exit(1)
	}
	defer termbox.Close()

	writeString("Initialized!")

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
