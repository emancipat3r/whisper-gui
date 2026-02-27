package whisper

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

type Transcriber struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Scanner
	mu     sync.Mutex
}

var (
	instance *Transcriber
	initMu   sync.Mutex
)

type whisperRequest struct {
	AudioFile string `json:"audio_file"`
}

type whisperResponse struct {
	Status string `json:"status"`
	Text   string `json:"text,omitempty"`
	Error  string `json:"error,omitempty"`
}

func Init(useGPU bool, modelName string) error {
	initMu.Lock()
	defer initMu.Unlock()

	// Shut down any existing instance
	if instance != nil {
		Close()
		instance = nil
	}

	var err error
	instance, err = newTranscriber(useGPU, modelName)
	return err
}

func newTranscriber(useGPU bool, modelName string) (*Transcriber, error) {
	// Find the script relative to the executable or current dir
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	scriptPath := filepath.Join(cwd, "whisper", "transcribe.py")

	args := []string{scriptPath, "--model", modelName}
	if useGPU {
		args = append(args, "--device", "cuda")
	}

	// Make sure we use the venv python if it exists
	pythonExec := "python3"
	venvPath := filepath.Join(cwd, ".venv", "bin", "python3")
	if _, err := os.Stat(venvPath); err == nil {
		pythonExec = venvPath
	}

	cmd := exec.Command(pythonExec, args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	// Create a scanner for reading JSON lines from stdout
	stdout := bufio.NewScanner(stdoutPipe)

	// We also want to capture stderr in case of catastrophic failure
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	t := &Transcriber{
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
	}

	// Wait for the READY signal
	if !stdout.Scan() {
		return nil, fmt.Errorf("python process exited unexpectedly")
	}

	var resp whisperResponse
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("failed to parse ready signal: %v (raw: %s)", err, stdout.Text())
	}

	if resp.Status != "READY" {
		return nil, fmt.Errorf("python process failed to initialize: %s", resp.Error)
	}

	return t, nil
}

func Transcribe(audioPath string, useGPU bool) (string, error) {
	if instance == nil {
		return "", errors.New("whisper not initialized")
	}

	instance.mu.Lock()
	defer instance.mu.Unlock()

	req := whisperRequest{AudioFile: audioPath}
	reqData, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	// Send request with newline
	_, err = instance.stdin.Write(append(reqData, '\n'))
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}

	// Read response
	if !instance.stdout.Scan() {
		return "", errors.New("failed to read response from python process")
	}

	var resp whisperResponse
	if err := json.Unmarshal(instance.stdout.Bytes(), &resp); err != nil {
		return "", fmt.Errorf("failed to parse response: %v", err)
	}

	if resp.Status == "ERROR" {
		return "", errors.New(resp.Error)
	}

	return strings.TrimSpace(resp.Text), nil
}

func Close() {
	if instance != nil && instance.cmd != nil {
		instance.stdin.Close()
		instance.cmd.Wait() // wait for graceful exit
	}
}
