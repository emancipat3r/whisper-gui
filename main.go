package main

import (
	"os"
	"whispergui/ui"
)

func main() {
	useGPU := false

	for _, arg := range os.Args[1:] {
		if arg == "--gpu" {
			useGPU = true
		}
	}

	ui.Run(useGPU)
}
