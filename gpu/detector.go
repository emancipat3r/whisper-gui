package gpu

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// DetectCUDA checks if CUDA is available for PyTorch and returns (available, device name, vram in GB)
func DetectCUDA() (bool, string, float64) {
	cwd, _ := os.Getwd()
	pythonExec := "python3"
	venvPath := filepath.Join(cwd, ".venv", "bin", "python3")
	if _, err := os.Stat(venvPath); err == nil {
		pythonExec = venvPath
	}

	cmd := exec.Command(pythonExec, "-c", "import torch; print(f'True\\n{torch.cuda.get_device_name(0)}\\n{torch.cuda.get_device_properties(0).total_memory / 1e9}' if torch.cuda.is_available() else 'False')")
	output, err := cmd.Output()
	if err != nil {
		return false, "", 0.0
	}
	res := string(output)
	if len(res) >= 4 && res[:4] == "True" {
		parts := strings.Split(res, "\n")
		name := ""
		vram := 0.0
		if len(parts) > 1 {
			name = parts[1]
		}
		if len(parts) > 2 {
			vram, _ = strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
		}
		return true, name, vram
	}
	return false, "", 0.0
}

// GetSystemRAMGB reads /proc/meminfo to determine the total physical system RAM in GB
func GetSystemRAMGB() float64 {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0.0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				kb, _ := strconv.ParseFloat(parts[1], 64)
				return kb / 1024.0 / 1024.0 // Convert KB to GB
			}
		}
	}
	return 0.0
}
