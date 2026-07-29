package main

import (
	"fmt"
	"os"

	"brain/cmd"
	officialplanning "brain/internal/official/planning"
)

func main() {
	if err := cmd.Execute(officialplanning.Registration()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
