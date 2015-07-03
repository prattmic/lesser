package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/nsf/termbox-go"
)

var tabStop = flag.Int("tabstop", 8, "Number of spaces per tab")

func main() {
	flag.Parse()
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s: %s filename\n", os.Args[0], os.Args[0])
		flag.PrintDefaults()
	}

	if len(flag.Args()) != 1 {
		flag.Usage()
		os.Exit(1)
	}

	name := flag.Arg(0)
	f, err := os.Open(name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open %s: %v\n", name, err)
		os.Exit(1)
	}

	err = termbox.Init()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to init: %v\n", err)
		os.Exit(1)
	}
	defer termbox.Close()

	l := NewLesser(f, *tabStop)
	l.Run()
}
