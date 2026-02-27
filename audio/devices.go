package audio

import "github.com/gordonklaus/portaudio"

func GetInputDeviceNames() ([]string, error) {
	if err := Initialize(); err != nil {
		return nil, err
	}
	devices, err := portaudio.Devices()
	if err != nil {
		return nil, err
	}
	var names []string

	// Set of names to ignore (common ALSA/Pulse audio pseudo-devices)
	ignoreList := []string{
		"sysdefault", "spdif", "lavrate", "samplerate", "speexrate", "jack",
		"pipewire", "pulse", "speex", "upmix", "vdownmix", "default", "dmix", "hw",
	}

	for _, d := range devices {
		if d.MaxInputChannels > 0 {
			// Check if name is in ignore list
			ignore := false
			for _, ig := range ignoreList {
				if d.Name == ig {
					ignore = true
					break
				}
			}
			if !ignore {
				names = append(names, d.Name)
			}
		}
	}
	return names, nil
}
