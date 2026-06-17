package utils

import (
	"strconv"
	"strings"
	"time"

	"github.com/Microservices/services/transcript-builder-service/internal/application/interfaces"
	"github.com/Microservices/services/transcript-builder-service/internal/model"
)

// ParseTranscription parses the whisper output format "0.00-1.23: text\n1.24-2.00: text"
// into a slice of TranscriptionSegments.
func ParseTranscription(transcription string) ([]interfaces.TranscriptionSegment, error) {
	var segments []interfaces.TranscriptionSegment
	for _, line := range strings.Split(transcription, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		colonIdx := strings.Index(line, ": ")
		if colonIdx < 0 {
			continue
		}
		timeRange := line[:colonIdx]
		text := strings.TrimSpace(line[colonIdx+2:])

		parts := strings.SplitN(timeRange, "-", 2)
		if len(parts) != 2 {
			continue
		}
		start, err := strconv.ParseFloat(parts[0], 64)
		if err != nil {
			continue
		}
		end, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			continue
		}

		startDur := time.Duration(start * float64(time.Second))
		dur := time.Duration((end - start) * float64(time.Second))
		segments = append(segments, interfaces.TranscriptionSegment{
			Interval: model.Interval{
				Start:    startDur,
				Duration: dur,
			},
			Text: text,
		})
	}
	return segments, nil
}

