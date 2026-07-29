package main

import (
	"fmt"
	"os"

	"github.com/JimmyMcBride/brain/cmd"
	officialplanning "github.com/JimmyMcBride/brain/internal/official/planning"
)

func main() {
	if err := cmd.Execute(officialplanning.Registration()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
