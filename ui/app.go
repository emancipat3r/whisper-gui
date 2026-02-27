package ui

import (
	"context"
	"fmt"
	"image/color"
	"os"
	"time"

	"whispergui/audio"
	"whispergui/whisper"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/atotto/clipboard"
)

var (
	isRecording      bool
	selectedModel    string = "small"
	lastWorkingModel string = "small"
)

func Run(useGPU bool, gpuName string, vramGB float64, ramGB float64) {
	fmt.Println("Launching Whisper GUI...")

	a := app.New()
	w := a.NewWindow("Whisper Voice-to-Text")
	w.Resize(fyne.NewSize(700, 500))

	// Create a context that will be cancelled when the window closes
	ctx, cancel := context.WithCancel(context.Background())
	w.SetOnClosed(func() {
		cancel()
		audio.Terminate()
	})

	// Get audio devices
	deviceNames, err := audio.GetInputDeviceNames()
	var selectedDevice string
	if err == nil && len(deviceNames) > 0 {
		selectedDevice = deviceNames[0]
	}
	deviceSelect := widget.NewSelect(deviceNames, nil) // OnChanged will be set after volume slider is defined
	if selectedDevice != "" {
		deviceSelect.SetSelected(selectedDevice)
	}

	// Create LEDs (brighter colors for better visibility)
	redColor := color.RGBA{R: 255, G: 80, B: 80, A: 255}
	greenColor := color.RGBA{R: 80, G: 255, B: 80, A: 255}
	greyColor := color.RGBA{R: 180, G: 180, B: 180, A: 255}

	ledSize := fyne.NewSize(8, 8)

	// Helper to create vertically-centered fixed-size LEDs using Fyne layouts
	createLed := func(c color.Color) (*canvas.Circle, *fyne.Container) {
		led := canvas.NewCircle(c)
		led.Resize(ledSize)
		// Shift led by half its size so its center coincides with the layout's calculated center
		led.Move(fyne.NewPos(-ledSize.Width/2, -ledSize.Height/2))

		// WithoutLayout container tracks the exact center position without forcing object resizing
		box := container.NewWithoutLayout(led)

		// widget.NewLabel("  ") establishes the default text height for perfect vertical tracking
		indicatorWrapper := container.NewStack(widget.NewLabel("  "), container.NewCenter(box))
		return led, indicatorWrapper
	}

	gpuLed, gpuIndicator := createLed(redColor)
	gpuLabelText := "GPU: Not Detected"
	if useGPU {
		gpuLed.FillColor = greenColor
		if gpuName != "" {
			gpuLabelText = "GPU: " + gpuName
		} else {
			gpuLabelText = "GPU: Detected"
		}
	}
	gpuStatusLabel := widget.NewLabel(gpuLabelText)

	readyLed, readyIndicator := createLed(redColor)
	readyStatusLabel := widget.NewLabel("Backend: Loading...")

	// Model Selection Dropdown
	modelSelect := widget.NewSelect([]string{"tiny", "base", "small", "medium", "large"}, nil)
	modelSelect.SetSelected(selectedModel)

	// Create status label with binding
	statusBinding := binding.NewString()
	statusBinding.Set("Ready to record")
	statusLabel := widget.NewLabelWithData(statusBinding)
	statusLabel.Alignment = fyne.TextAlignCenter
	statusLabel.TextStyle = fyne.TextStyle{Bold: true}

	// Create recording indicator
	recordingIndicator, recordingIndicatorWrapper := createLed(greyColor)

	// Create Volume Control and Meter
	volumeSlider := widget.NewSlider(0, 5)
	volumeSlider.SetValue(1.0)

	// Create Custom VU Meter (10 segments)
	vuSegments := make([]*canvas.Rectangle, 10)
	vuMeter := container.NewHBox()
	for i := 0; i < 10; i++ {
		seg := canvas.NewRectangle(color.RGBA{R: 50, G: 50, B: 50, A: 255}) // dark background
		seg.SetMinSize(fyne.NewSize(8, 20))
		vuSegments[i] = seg
		// spacing between bars
		vuMeter.Add(seg)
		if i < 9 {
			spacer := canvas.NewRectangle(color.Transparent)
			spacer.SetMinSize(fyne.NewSize(2, 20))
			vuMeter.Add(spacer)
		}
	}

	// Set up a callback for the audio level meter
	onLevel := func(level float64) {
		fyne.Do(func() {
			activeSegments := int(level * 10)
			if activeSegments > 10 {
				activeSegments = 10
			}

			for i := 0; i < 10; i++ {
				if i < activeSegments {
					if i < 6 {
						vuSegments[i].FillColor = color.RGBA{R: 80, G: 255, B: 80, A: 255}
					} else if i < 8 {
						vuSegments[i].FillColor = color.RGBA{R: 255, G: 255, B: 80, A: 255}
					} else {
						vuSegments[i].FillColor = color.RGBA{R: 255, G: 80, B: 80, A: 255}
					}
				} else {
					vuSegments[i].FillColor = color.RGBA{R: 50, G: 50, B: 50, A: 255}
				}
				vuSegments[i].Refresh()
			}
		})
	}

	// Function to start or restart the monitor stream
	startAudioMonitor := func() {
		audio.StartMonitoring(selectedDevice, func() float64 { return volumeSlider.Value }, onLevel)
	}

	// Now set the OnChanged for deviceSelect since we have onLevel defined
	deviceSelect.OnChanged = func(s string) {
		selectedDevice = s
		startAudioMonitor()
	}

	// Start initial monitoring stream
	startAudioMonitor()

	// Status bar with indicator
	statusBar := container.NewHBox(
		recordingIndicatorWrapper,
		statusLabel,
		layout.NewSpacer(),
		widget.NewLabel("Vol:"),
		container.NewGridWrap(fyne.NewSize(100, 36), volumeSlider),
		widget.NewLabel("Lvl:"),
		container.NewCenter(vuMeter),
		layout.NewSpacer(),
		widget.NewLabel("Model:"),
		modelSelect,
		layout.NewSpacer(),
		widget.NewLabel("Input:"),
		deviceSelect,
		layout.NewSpacer(),
		gpuIndicator,
		gpuStatusLabel,
		layout.NewSpacer(),
		readyIndicator,
		readyStatusLabel,
	)

	bindStr := binding.NewString()
	textBox := widget.NewMultiLineEntry()
	textBox.Wrapping = fyne.TextWrapWord
	textBox.Bind(bindStr)
	textBox.SetPlaceHolder("Your transcribed text will appear here...\n\nClick 'Start Recording' to begin.")

	var startStop *widget.Button

	startStop = widget.NewButton("Start Recording", func() {
		if !isRecording {
			isRecording = true
			startStop.SetText("⏹ Stop Recording")
			startStop.Importance = widget.HighImportance
			statusBinding.Set("🎤 Recording...")
			recordingIndicator.FillColor = color.RGBA{R: 220, G: 20, B: 60, A: 255} // Crimson red
			recordingIndicator.Refresh()

			go func() {
				// Use the OS temp directory
				audioPath := fmt.Sprintf("%s/%d.wav", os.TempDir(), time.Now().Unix())

				// Ensure temp file is cleaned up even on crash
				defer os.Remove(audioPath)
				err := audio.StartRecording(audioPath)
				if err != nil {
					select {
					case <-ctx.Done():
						return
					default:
						fyne.Do(func() {
							bindStr.Set("Error starting recording: " + err.Error())
							isRecording = false
							startStop.SetText("▶ Start Recording")
							startStop.Importance = widget.MediumImportance
							statusBinding.Set("Error: " + err.Error())
							recordingIndicator.FillColor = color.RGBA{R: 128, G: 128, B: 128, A: 255}
							recordingIndicator.Refresh()
						})
						return
					}
				}

				// Wait until user stops recording or window closes
				for isRecording {
					select {
					case <-ctx.Done():
						audio.StopRecording()
						return
					case <-time.After(200 * time.Millisecond):
						// Continue waiting
					}
				}

				audio.StopRecording()

				// Check if context is cancelled before UI updates
				select {
				case <-ctx.Done():
					return
				default:
					fyne.Do(func() {
						statusBinding.Set("⏳ Transcribing...")
						recordingIndicator.FillColor = color.RGBA{R: 255, G: 165, B: 0, A: 255} // Orange
						recordingIndicator.Refresh()
					})
				}

				transcript, err := whisper.Transcribe(audioPath, useGPU)
				if err != nil {
					select {
					case <-ctx.Done():
						return
					default:
						fyne.Do(func() {
							bindStr.Set("Transcription error: " + err.Error())
							startStop.SetText("▶ Start Recording")
							startStop.Importance = widget.MediumImportance
							statusBinding.Set("Error: " + err.Error())
							recordingIndicator.FillColor = color.RGBA{R: 128, G: 128, B: 128, A: 255}
							recordingIndicator.Refresh()
						})
						return
					}
				}

				// Check if context is cancelled before final UI updates
				select {
				case <-ctx.Done():
					return
				default:
					fyne.Do(func() {
						bindStr.Set(transcript)
						startStop.SetText("▶ Start Recording")
						startStop.Importance = widget.MediumImportance
						statusBinding.Set("✓ Transcription complete")
						recordingIndicator.FillColor = color.RGBA{R: 34, G: 139, B: 34, A: 255} // Green
						recordingIndicator.Refresh()
					})
				}
			}()

		} else {
			isRecording = false
			statusBinding.Set("⏳ Processing...")
		}
	})

	startStop.Importance = widget.MediumImportance

	copyBtn := widget.NewButton("📋 Copy to Clipboard", func() {
		val, _ := bindStr.Get()
		if val != "" {
			_ = clipboard.WriteAll(val)
			statusBinding.Set("✓ Copied to clipboard")
			recordingIndicator.FillColor = color.RGBA{R: 34, G: 139, B: 34, A: 255} // Green
			recordingIndicator.Refresh()
		}
	})

	// Button container with better layout
	buttonBar := container.NewHBox(
		layout.NewSpacer(),
		startStop,
		copyBtn,
		layout.NewSpacer(),
	)

	// Main content with padding
	content := container.NewBorder(
		container.NewVBox(
			widget.NewSeparator(),
			statusBar,
			widget.NewSeparator(),
		),
		container.NewVBox(
			widget.NewSeparator(),
			buttonBar,
		),
		nil,
		nil,
		container.NewPadded(textBox),
	)

	w.SetContent(content)

	// Show GPU status dialog at startup
	if !useGPU {
		dialog.ShowConfirm(
			"⚠️  GPU Not Detected",
			"GPU/CUDA is not available on this system.\n\nTranscription will use CPU mode, which is significantly slower.\n\nDo you want to continue?",
			func(continue_app bool) {
				if !continue_app {
					a.Quit()
				}
			},
			w,
		)
	}

	startStop.Disable() // Disable start button while loading model
	modelSelect.Disable()

	var loadModel func(modelName string, gpuMode bool)
	loadModel = func(modelName string, gpuMode bool) {
		fyne.Do(func() {
			readyStatusLabel.SetText("Backend: Loading...")
			readyLed.FillColor = color.RGBA{R: 255, G: 165, B: 0, A: 255} // Orange
			readyLed.Refresh()
			startStop.Disable()
			modelSelect.Disable()
		})

		err := whisper.Init(gpuMode, modelName)
		select {
		case <-ctx.Done():
			return
		default:
			fyne.Do(func() {
				if err != nil {
					// Fall back to the last working model if initialization failed
					readyStatusLabel.SetText("Backend: Error")
					readyLed.FillColor = redColor
					readyLed.Refresh()
					bindStr.Set(fmt.Sprintf("Error loading model '%s': %v\nFalling back to '%s'...", modelName, err, lastWorkingModel))
					statusBinding.Set("Error loading model")
					recordingIndicator.FillColor = redColor
					recordingIndicator.Refresh()

					// Revert the dropdown visually without triggering OnChanged
					modelSelect.SetSelected(lastWorkingModel)

					// Trigger background reload of the last working model
					// Use the global useGPU state for our safe fallback
					go loadModel(lastWorkingModel, useGPU)
				} else {
					// Success! Update our fallback state
					lastWorkingModel = modelName

					startStop.Enable()
					modeStr := "CPU"
					if gpuMode {
						modeStr = "GPU"
					}
					readyStatusLabel.SetText(fmt.Sprintf("Backend: Ready (%s, %s)", modelName, modeStr))
					readyLed.FillColor = greenColor
					readyLed.Refresh()
					statusBinding.Set("Ready to record")
					recordingIndicator.FillColor = greenColor
					recordingIndicator.Refresh()
					modelSelect.Enable()
				}
			})
		}
	}

	modelSelect.OnChanged = func(s string) {
		// Only trigger reload if the value actually changed
		if s != selectedModel && s != "" {
			selectedModel = s

			// Always prompt the user with system requirements when switching models
			var vramReq float64
			switch s {
			case "tiny":
				vramReq = 1.0
			case "base":
				vramReq = 1.0
			case "small":
				vramReq = 2.0
			case "medium":
				vramReq = 5.0
			case "large":
				vramReq = 10.5
			default:
				vramReq = 2.0
			}

			// Format the resource usage string
			var resourceStats string
			if useGPU {
				resourceStats = fmt.Sprintf("Target Mode: GPU (CUDA)\n\nSystem Status:\n• GPU VRAM: %.1f GB available\n• System RAM: %.1f GB available", vramGB, ramGB)
			} else {
				resourceStats = fmt.Sprintf("Target Mode: CPU\n\nSystem Status:\n• System RAM: %.1f GB available", ramGB)
			}

			warningMessage := fmt.Sprintf(
				"The Whisper '%s' model requires approximately %.1f GB of RAM to load into memory.\n\n%s\n\nIf the model size exceeds your available memory, the operating system will forcefully crash this application.\n\nDo you want to proceed and load the %s model anyway?",
				s, vramReq, resourceStats, s,
			)

			dialog.ShowConfirm(
				fmt.Sprintf("Load %s Model", s),
				warningMessage,
				func(proceed bool) {
					if proceed {
						// Attempt to load the model (may crash)
						go loadModel(selectedModel, useGPU)
					} else {
						// Revert dropdown nicely
						modelSelect.SetSelected(lastWorkingModel)
						selectedModel = lastWorkingModel
					}
				},
				w,
			)
		}
	}

	go loadModel(selectedModel, useGPU)

	w.ShowAndRun()
}
