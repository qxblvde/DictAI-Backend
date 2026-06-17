package utils

import (
	"bytes"
	"fmt"
	"io"
	"time"
)

func NewAudioSegmentView(
	buffer *bytes.Buffer,
	start, duration time.Duration,
	sampleRate,
	bitDepth,
	numChannels int,
) (io.Reader, error) {
	bytesPerSample := bitDepth / 8
	bytesPerSecond := sampleRate * bytesPerSample * numChannels

	startByte := int64(start.Seconds()) * int64(bytesPerSecond)
	limitBytes := int64(duration.Seconds()) * int64(bytesPerSecond)

	if startByte >= int64(buffer.Len()) {
		return nil, fmt.Errorf("invalid interval")
	}
	if startByte+limitBytes > int64(buffer.Len()) {
		limitBytes = int64(buffer.Len()) - startByte
	}

	view := io.NewSectionReader(bytes.NewReader(buffer.Bytes()), startByte, limitBytes)
	return view, nil
}
