package main

import (
	"whispergui/gpu"
	"whispergui/ui"
	"whispergui/whisper"
)

func main() {
	// Detect if GPU/CUDA is available
	gpuAvailable, gpuName, vramGB := gpu.DetectCUDA()
	ramGB := gpu.GetSystemRAMGB()

	// Ensure whisper backend closes gracefully
	defer whisper.Close()

	ui.Run(gpuAvailable, gpuName, vramGB, ramGB)
}
