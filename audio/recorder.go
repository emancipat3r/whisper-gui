package audio

import (
	"encoding/binary"
	"math"
	"os"
	"sync"

	"github.com/gordonklaus/portaudio"
)

const channels = 1

var (
	mu              sync.Mutex
	initialized     bool
	initErr         error
	terminateNeeded bool
)

func Initialize() error {
	mu.Lock()
	defer mu.Unlock()

	if initialized {
		return initErr
	}

	initErr = portaudio.Initialize()
	if initErr == nil {
		initialized = true
		terminateNeeded = true
	}
	return initErr
}

func Terminate() {
	mu.Lock()
	defer mu.Unlock()

	if terminateNeeded {
		portaudio.Terminate()
		terminateNeeded = false
		initialized = false
	}
}

var (
	monitorStream     *portaudio.Stream
	monitorFile       *os.File
	monitorDataSize   uint32
	monitorSampleRate float64
	monitorRecording  bool
	monitorMux        sync.Mutex
	monitorDone       chan struct{}
	monitorFinished   chan struct{}
)

func StartMonitoring(deviceName string, getVolumeGain func() float64, onLevel func(float64)) error {
	StopMonitoring() // Ensure previous monitor is closed

	if err := Initialize(); err != nil {
		return err
	}

	devices, err := portaudio.Devices()
	if err != nil {
		return err
	}

	var device *portaudio.DeviceInfo
	if deviceName == "" {
		device, err = portaudio.DefaultInputDevice()
		if err != nil {
			return err
		}
	} else {
		for _, d := range devices {
			if d.Name == deviceName {
				device = d
				break
			}
		}
		if device == nil {
			device, err = portaudio.DefaultInputDevice()
			if err != nil {
				return err
			}
		}
	}

	monitorSampleRate = device.DefaultSampleRate

	in := make([]int16, 1024)
	params := portaudio.StreamParameters{
		Input: portaudio.StreamDeviceParameters{
			Device:   device,
			Channels: channels,
			Latency:  device.DefaultLowInputLatency,
		},
		SampleRate:      monitorSampleRate,
		FramesPerBuffer: len(in),
		Flags:           portaudio.ClipOff,
	}

	monitorStream, err = portaudio.OpenStream(params, in)
	if err != nil {
		return err
	}

	if err = monitorStream.Start(); err != nil {
		return err
	}

	monitorDone = make(chan struct{})
	monitorFinished = make(chan struct{})

	// Continuous Monitoring Goroutine
	go func() {
		defer close(monitorFinished)
		for {
			select {
			case <-monitorDone:
				return
			default:
				if err := monitorStream.Read(); err != nil {
					return
				}

				var sumSquares float64
				volGain := 1.0
				if getVolumeGain != nil {
					volGain = getVolumeGain()
				}

				// Copy in order to hold the lock briefly if recording
				var recordingBuffer []int16

				for _, sample := range in {
					scaled := float64(sample) * volGain
					if scaled > 32767 {
						scaled = 32767
					} else if scaled < -32768 {
						scaled = -32768
					}
					finalSample := int16(scaled)

					sumSquares += float64(finalSample) * float64(finalSample)
					recordingBuffer = append(recordingBuffer, finalSample)
				}

				// If we are actively recording, write to file under mutex
				monitorMux.Lock()
				if monitorRecording && monitorFile != nil {
					for _, s := range recordingBuffer {
						binary.Write(monitorFile, binary.LittleEndian, s)
						monitorDataSize += 2
					}
				}
				monitorMux.Unlock()

				// Report RMS level
				if onLevel != nil && len(in) > 0 {
					var rms float64 = 0.0
					if sumSquares > 0 {
						ms := sumSquares / float64(len(in))
						rmsValue := math.Sqrt(ms)
						// Highly sensitive mapped RMS, clamped 0.0-1.0
						rms = (rmsValue / 32768.0) * 15.0
					}

					if rms < 0 {
						rms = 0
					} else if rms > 1.0 {
						rms = 1.0
					}
					onLevel(rms)
				}
			}
		}
	}()

	return nil
}

func StopMonitoring() {
	if monitorStream != nil {
		close(monitorDone)
		<-monitorFinished
		monitorStream.Stop()
		monitorStream.Close()
		monitorStream = nil
	}
}

func StartRecording(path string) error {
	monitorMux.Lock()
	defer monitorMux.Unlock()

	if monitorRecording {
		return nil // already recording
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}

	_, err = file.Write(make([]byte, 44)) // reserve WAV header space
	if err != nil {
		file.Close()
		return err
	}

	monitorFile = file
	monitorDataSize = 0
	monitorRecording = true
	return nil
}

func StopRecording() error {
	monitorMux.Lock()
	defer monitorMux.Unlock()

	if !monitorRecording {
		return nil
	}
	monitorRecording = false

	if monitorFile != nil {
		monitorFile.Seek(0, 0)
		writeWavHeader(monitorFile, monitorDataSize, monitorSampleRate)
		err := monitorFile.Close()
		monitorFile = nil
		return err
	}
	return nil
}

func writeWavHeader(f *os.File, dataSize uint32, sampleRate float64) {
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
