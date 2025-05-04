package ui

import (
	"fmt"
	"os"
	"time"

	"whispergui/audio"
	"whispergui/whisper"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
	"github.com/atotto/clipboard"
)

var isRecording bool

func Run(useGPU bool) {
	fmt.Println("Launching Whisper GUI...")

	a := app.New()
	w := a.NewWindow("Whisper GUI")
	w.Resize(fyne.NewSize(600, 400))

	bindStr := binding.NewString()
	textBox := widget.NewMultiLineEntry()
	textBox.Wrapping = fyne.TextWrapWord
	textBox.Bind(bindStr)
	textBox.SetPlaceHolder("Transcription output will appear here...")

	var startStop *widget.Button

	startStop = widget.NewButton("Start Recording", func() {
		if !isRecording {
			isRecording = true
			startStop.SetText("Stop Recording")

			go func() {
				audioPath := fmt.Sprintf("temp/%d.wav", time.Now().Unix())
				stopFunc, err := audio.StartRecording(audioPath)
				if err != nil {
					bindStr.Set("Error starting recording: " + err.Error())
					isRecording = false
					startStop.SetText("Start Recording")
					return
				}

				// Wait until user stops recording
				for isRecording {
					time.Sleep(200 * time.Millisecond)
				}

				stopFunc()

				transcript, err := whisper.Transcribe(audioPath, useGPU)
				if err != nil {
					bindStr.Set("Transcription error: " + err.Error())
					startStop.SetText("Start Recording")
					return
				}

				bindStr.Set(transcript)
				os.Remove(audioPath)
				startStop.SetText("Start Recording")
			}()

		} else {
			isRecording = false
		}
	})

	copyBtn := widget.NewButton("Copy to Clipboard", func() {
		val, _ := bindStr.Get()
		_ = clipboard.WriteAll(val)
	})

	w.SetContent(container.NewVBox(
		textBox,
		startStop,
		copyBtn,
	))

	w.ShowAndRun()
}
