package main

import (
	"fmt"
	"os"

	"github.com/victorhsb/branchless-pr/internal/cli"
)

func main() {
	if err := cli.Execute("bpr"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
