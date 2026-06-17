package utils

import (
	"bytes"
	"fmt"
)

func GetWAVParams(audio *bytes.Buffer) (int, int, int, error) {
	data := audio.Bytes()
	sampleRate, bitDepth, numChannels, err := GetWAVParamsFromBytes(data)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to read WAV header: %w", err)
	}
	return sampleRate, bitDepth, numChannels, nil
}
