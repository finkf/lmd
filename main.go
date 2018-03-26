package main

import (
	"os"
)

func main() {
	if err := execute(); err != nil {
		// no need to print error message
		// since cobra takes care of this
		os.Exit(1)
	}
}
