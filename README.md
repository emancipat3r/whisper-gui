# whisper-gui

A simple Go GUI for OpenAI's Whisper model.

## Prerequisites

1.  **Python 3.8+**
2.  **Go 1.20+**
3.  **PortAudio** (for microphone recording)
    *   Ubuntu/Debian: `sudo apt-get install portaudio19-dev`
    *   macOS: `brew install portaudio`

## Installation

1.  **Install Python dependencies:**
    ```bash
    pip install -r requirements.txt
    ```

    *Note: If you want GPU acceleration, ensure you have the correct version of PyTorch installed for your system (e.g., CUDA) before running the above command.*

2.  **Run the application:**
    ```bash
    go run main.go
    ```
