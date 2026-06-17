package utils

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"time"
)

// GetWAVParamsFromBytes reads WAV header from raw bytes and returns sampleRate, bitDepth, numChannels.
// It scans chunks properly to handle non-standard WAV files with extra chunks (e.g. JUNK, LIST)
// before the fmt chunk, as produced by some recorders (including Flutter's record package on macOS).
func GetWAVParamsFromBytes(data []byte) (sampleRate, bitDepth, numChannels int, err error) {
	if len(data) < 12 {
		return 0, 0, 0, fmt.Errorf("data too short to be a WAV file")
	}
	if string(data[0:4]) != "RIFF" {
		return 0, 0, 0, fmt.Errorf("not a RIFF/WAV file")
	}
	if string(data[8:12]) != "WAVE" {
		return 0, 0, 0, fmt.Errorf("not a WAV file")
	}

	// Scan chunks to find "fmt "
	offset := 12
	for offset+8 <= len(data) {
		chunkID := string(data[offset : offset+4])
		chunkSize := int(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		if chunkID == "fmt " {
			if offset+8+16 > len(data) {
				return 0, 0, 0, fmt.Errorf("fmt chunk too short")
			}
			fmtData := data[offset+8:]
			numChannels = int(binary.LittleEndian.Uint16(fmtData[2:4]))
			sampleRate = int(binary.LittleEndian.Uint32(fmtData[4:8]))
			bitDepth = int(binary.LittleEndian.Uint16(fmtData[14:16]))
			return sampleRate, bitDepth, numChannels, nil
		}
		// Move to next chunk (chunk data + padding byte if odd size)
		next := offset + 8 + chunkSize
		if chunkSize%2 != 0 {
			next++
		}
		if next <= offset {
			break // prevent infinite loop on malformed data
		}
		offset = next
	}

	return 0, 0, 0, fmt.Errorf("fmt chunk not found in WAV file")
}

// NewWAVSegment returns an io.Reader for a time slice of the WAV audio data.
// The returned reader produces a valid WAV file with header.
func NewWAVSegment(
	audioData []byte,
	start, duration time.Duration,
	sampleRate, bitDepth, numChannels int,
) (io.Reader, error) {
	// Find the "data" chunk offset
	dataOffset, dataSize, err := findWAVDataChunk(audioData)
	if err != nil {
		return nil, fmt.Errorf("cannot find data chunk: %w", err)
	}

	bytesPerSample := bitDepth / 8
	bytesPerSecond := sampleRate * bytesPerSample * numChannels

	startByte := dataOffset + int(start.Seconds()*float64(bytesPerSecond))
	limitBytes := int(duration.Seconds() * float64(bytesPerSecond))

	if startByte >= dataOffset+dataSize {
		return nil, fmt.Errorf("start offset beyond audio data")
	}
	if startByte+limitBytes > dataOffset+dataSize {
		limitBytes = dataOffset + dataSize - startByte
	}

	pcmData := audioData[startByte : startByte+limitBytes]
	header := buildWAVHeader(numChannels, sampleRate, bitDepth, len(pcmData))

	return io.MultiReader(bytes.NewReader(header), bytes.NewReader(pcmData)), nil
}

// findWAVDataChunk scans RIFF chunks to locate the "data" chunk.
// Returns the byte offset of the PCM data (after the chunk header) and its size.
func findWAVDataChunk(data []byte) (offset, size int, err error) {
	if len(data) < 12 || string(data[0:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
		return 0, 0, fmt.Errorf("not a WAV file")
	}
	pos := 12
	for pos+8 <= len(data) {
		chunkID := string(data[pos : pos+4])
		chunkSize := int(binary.LittleEndian.Uint32(data[pos+4 : pos+8]))
		if chunkID == "data" {
			return pos + 8, chunkSize, nil
		}
		next := pos + 8 + chunkSize
		if chunkSize%2 != 0 {
			next++
		}
		if next <= pos {
			break
		}
		pos = next
	}
	return 0, 0, fmt.Errorf("data chunk not found")
}

func buildWAVHeader(numChannels, sampleRate, bitDepth, dataSize int) []byte {
	h := make([]byte, 44)
	copy(h[0:4], "RIFF")
	binary.LittleEndian.PutUint32(h[4:8], uint32(dataSize+36))
	copy(h[8:12], "WAVE")
	copy(h[12:16], "fmt ")
	binary.LittleEndian.PutUint32(h[16:20], 16)
	binary.LittleEndian.PutUint16(h[20:22], 1) // PCM
	binary.LittleEndian.PutUint16(h[22:24], uint16(numChannels))
	binary.LittleEndian.PutUint32(h[24:28], uint32(sampleRate))
	byteRate := sampleRate * numChannels * bitDepth / 8
	binary.LittleEndian.PutUint32(h[28:32], uint32(byteRate))
	blockAlign := numChannels * bitDepth / 8
	binary.LittleEndian.PutUint16(h[32:34], uint16(blockAlign))
	binary.LittleEndian.PutUint16(h[34:36], uint16(bitDepth))
	copy(h[36:40], "data")
	binary.LittleEndian.PutUint32(h[40:44], uint32(dataSize))
	return h
}
