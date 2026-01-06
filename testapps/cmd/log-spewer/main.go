package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	var interval time.Duration
	var lines int
	flag.DurationVar(&interval, "interval", 50*time.Millisecond, "Delay between lines")
	flag.IntVar(&lines, "lines", 50, "Number of lines to emit before sleeping forever")
	flag.Parse()

	for i := 0; i < lines; i++ {
		_, _ = fmt.Fprintf(os.Stdout, "stdout line %d\n", i)
		_, _ = fmt.Fprintf(os.Stderr, "stderr line %d\n", i)
		time.Sleep(interval)
	}
	select {}
}
