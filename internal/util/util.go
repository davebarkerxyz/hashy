package util

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

func PrintfAtLine(format string, line int, args ...any) {
	fmt.Printf("\033[%d;0H\033[K", line)
	fmt.Printf(format, args...)
}

func Die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

func TermPrint(format string, args ...any) {
	width, _, err := term.GetSize(0)
	if err == nil {
		full := fmt.Sprintf(format, args...)
		if len(full) > width {
			fmt.Printf(full[:width-3] + "...\n")
			return
		}
	}

	fmt.Printf(format+"\n", args...)
}
