package audio

import (
	"encoding/binary"
	"fmt"
	"os"

	"github.com/gordonklaus/portaudio"
)

const channels = 1

var stream *portaudio.Stream
var file *os.File
var dataSize uint32
var sampleRate float64

func StartRecording(path string) (func() error, error) {
	if err := portaudio.Initialize(); err != nil {
		return nil, err
	}

	devices, err := portaudio.Devices()
	if err != nil {
		return nil, err
	}

	if len(devices) <= 28 {
		return nil, fmt.Errorf("device 28 not found (only %d devices)", len(devices))
	}

	device := devices[28]
	sampleRate = device.DefaultSampleRate

	file, err = os.Create(path)
	if err != nil {
		return nil, err
	}

	_, err = file.Write(make([]byte, 44)) // reserve WAV header space
	if err != nil {
		file.Close()
		return nil, err
	}

	in := make([]int16, 64)
	params := portaudio.StreamParameters{
		Input: portaudio.StreamDeviceParameters{
			Device:   device,
			Channels: channels,
			Latency:  device.DefaultLowInputLatency,
		},
		SampleRate:      sampleRate,
		FramesPerBuffer: len(in),
		Flags:           portaudio.ClipOff,
	}

	stream, err = portaudio.OpenStream(params, in)
	if err != nil {
		file.Close()
		return nil, err
	}

	if err = stream.Start(); err != nil {
		file.Close()
		return nil, err
	}

	go func() {
		for stream != nil {
			if err := stream.Read(); err != nil {
				break
			}
			for _, sample := range in {
				binary.Write(file, binary.LittleEndian, sample)
				dataSize += 2
			}
		}
	}()

	stop := func() error {
		if stream != nil {
			stream.Stop()
			stream.Close()
			stream = nil
		}
		portaudio.Terminate()

		file.Seek(0, 0)
		writeWavHeader(file, dataSize)
		return file.Close()
	}

	return stop, nil
}

func writeWavHeader(f *os.File, dataSize uint32) {
	var header [44]byte

	copy(header[0:], "RIFF")
	binary.LittleEndian.PutUint32(header[4:], 36+dataSize)
	copy(header[8:], "WAVE")
	copy(header[12:], "fmt ")
	binary.LittleEndian.PutUint32(header[16:], 16)
	binary.LittleEndian.PutUint16(header[20:], 1)
	binary.LittleEndian.PutUint16(header[22:], channels)
	binary.LittleEndian.PutUint32(header[24:], uint32(sampleRate))
	binary.LittleEndian.PutUint32(header[28:], uint32(sampleRate)*2)
	binary.LittleEndian.PutUint16(header[32:], 2)
	binary.LittleEndian.PutUint16(header[34:], 16)
	copy(header[36:], "data")
	binary.LittleEndian.PutUint32(header[40:], dataSize)

	f.Write(header[:])
}
