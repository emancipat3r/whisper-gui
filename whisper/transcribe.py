#!/usr/bin/env python3

import whisper
import sys
import argparse
import json

def main():
    parser = argparse.ArgumentParser(description='Transcribe audio using Whisper via IPC')
    parser.add_argument('--device', default='cpu', help='Device to use (cpu or cuda)')
    parser.add_argument('--model', default='small', help='Whisper model to use (tiny, base, small, medium, large)')
    args = parser.parse_args()

    try:
        model = whisper.load_model(args.model, device=args.device)
        print(json.dumps({"status": "READY"}), flush=True)
    except Exception as e:
        print(json.dumps({"status": "ERROR", "error": str(e)}), flush=True)
        sys.exit(1)

    for line in sys.stdin:
        try:
            req = json.loads(line)
            audio_file = req.get("audio_file")
            if not audio_file:
                print(json.dumps({"status": "ERROR", "error": "Missing audio_file in request"}), flush=True)
                continue
            
            result = model.transcribe(audio_file)
            print(json.dumps({"status": "SUCCESS", "text": result["text"].strip()}), flush=True)
        except Exception as e:
            print(json.dumps({"status": "ERROR", "error": str(e)}), flush=True)

if __name__ == "__main__":
    main()
