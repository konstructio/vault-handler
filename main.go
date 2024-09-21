package main

import (
	"fmt"
	"os"

	"github.com/kubefirst/vault-handler/cmd"
)

func main() {
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err.Error())
		os.Exit(1)
	}
}
