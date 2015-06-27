package main

import (
	"fmt"
	"os"
	"time"

	"github.com/nsf/termbox-go"
)

func main() {
	err := termbox.Init()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to init: %v\n", err)
		os.Exit(1)
	}
	defer termbox.Close()

	fmt.Fprintf(os.Stderr, "Initialized!\n")
	time.Sleep(1*time.Second)
}
