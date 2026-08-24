package decoder

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"

	"orbitlink/internal/recorder"
)

type Assembler struct {
	frameSize int
	buffer    []byte
	sequence  uint64
}

func NewAssembler(frameSize int) (*Assembler, error) {
	if frameSize < 8 || frameSize > 64*1024 {
		return nil, fmt.Errorf("frame size is outside decoder limits")
	}
	return &Assembler{frameSize: frameSize, buffer: make([]byte, 0, frameSize)}, nil
}

func (a *Assembler) Push(epoch int64, symbols []byte) []recorder.Frame {
	a.buffer = append(a.buffer, symbols...)
	var frames []recorder.Frame
	for len(a.buffer) >= a.frameSize {
		payload := append([]byte(nil), a.buffer[:a.frameSize]...)
		a.buffer = a.buffer[a.frameSize:]
		a.sequence++
		frames = append(frames, recorder.Frame{
			Sequence: a.sequence, Epoch: epoch, Payload: payload, CRC32: crc32.ChecksumIEEE(payload),
		})
	}
	return frames
}

func (a *Assembler) Reset() int {
	discarded := len(a.buffer)
	a.buffer = a.buffer[:0]
	return discarded
}

func EncodeTelemetry(counter uint64, size int) []byte {
	data := make([]byte, size)
	binary.BigEndian.PutUint64(data, counter)
	for index := 8; index < len(data); index++ {
		data[index] = byte((int(counter) + index) % 251)
	}
	return data
}
