# whisper-gui

A simple Go GUI for OpenAI's Whisper model.

## Prerequisites

1.  **Python 3.8+**
2.  **Go 1.20+**
3.  **PortAudio** (for microphone recording)
    *   Ubuntu/Debian: `sudo apt-get install portaudio19-dev`
    *   macOS: `brew install portaudio`

## Installation

1.  **Create a Python Virtual Environment:**
    ```bash
    python3 -m venv .venv
    ```

2.  **Activate the Virtual Environment and Install Dependencies:**
    ```bash
    source .venv/bin/activate
    pip install -r requirements.txt
    ```
    *Note: If you want GPU acceleration, ensure you have the correct version of PyTorch installed for your system (e.g., CUDA) before running the above command.*

3.  **Compile the Application:**
    ```bash
    go build -o whisper-gui
    ```

4.  **Run the Compiled Application:**
    ```bash
    ./whisper-gui
    ```
    *(You can now move this binary anywhere on your system. It will automatically attempt to locate the original `.venv` directory based on where it was compiled.)*

## Alternative Execution Methods

### Running directly with Go
During development, you can run the application directly without compiling a final binary:
```bash
go run main.go
```

### Advanced Usage (Custom Python Environments)
If you move the compiled binary out of the project directory (e.g., to your `~/bin`), the application will no longer be able to automatically find the `.venv` directory for its Python dependencies if the original project path changes.
You can use the `PYTHON_ENV` environment variable to explicitly point the binary to your environment.

**Point to the `.venv` directory:**
```bash
PYTHON_ENV=/path/to/whisper-gui/.venv whisper-gui
```

**Point directly to a python executable:**
```bash
PYTHON_ENV=/usr/local/bin/python3.10 whisper-gui
```

*Tip: You can add an alias in your `~/.bashrc` or `~/.zshrc` to make this easier:*
```bash
alias whisper-gui="PYTHON_ENV=/path/to/whisper-gui/.venv whisper-gui"
```
