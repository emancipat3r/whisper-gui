#!/usr/bin/env python3

import whisper
import sys

if len(sys.argv) != 2:
    print("Usage: transcribe.py <audio.wav>")
    sys.exit(1)

model = whisper.load_model("small")
result = model.transcribe(sys.argv[1])
print(result["text"].strip())
