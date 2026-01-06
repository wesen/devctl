package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	var after time.Duration
	var code int
	flag.DurationVar(&after, "after", 250*time.Millisecond, "Duration before exit")
	flag.IntVar(&code, "code", 2, "Exit code")
	flag.Parse()

	_, _ = fmt.Fprintf(os.Stderr, "crash-after starting (after=%s code=%d)\n", after, code)
	_, _ = fmt.Fprintln(os.Stdout, "crash-after: hello")
	time.Sleep(after)
	_, _ = fmt.Fprintln(os.Stderr, "crash-after: exiting now")
	os.Exit(code)
}
