package whisper

import (
	"bytes"
	"errors"
	"os/exec"
	"strings"
)

func Transcribe(audioPath string) (string, error) {
	cmd := exec.Command("python3", "/opt/whisper-gui/whisper/transcribe.py", audioPath)

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", errors.New("python whisper error: " + stderr.String())
	}

	return strings.TrimSpace(out.String()), nil
}
