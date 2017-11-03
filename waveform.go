// Copyright 2013, Carnegie Mellon University. All Rights Reserved.
// Use of this code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Author: Alok Parlikar <aup@cs.cmu.edu>

package goflite

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"
)

// Structure for Waveform Data
type Wave struct {
	SampleRate  uint16
	NumSamples  uint32
	NumChannels uint16
	Samples     []uint16
	encodedIn   *io.PipeWriter
	encodedOut  *io.PipeReader
}

type SampleFunc func(uint16) interface{}

func SampleNoOp(sample uint16) interface{} {
	return sample
}

func SampleToS16(sample uint16) interface{} {
	return int16(sample)
}

func SampleTo2U8(sample uint16) interface{} {
	return []uint8{
		uint8(sample & 0xFF),
		uint8(sample >> 8),
	}
}

// Get the Duration of Waveform in Seconds
func (self *Wave) Duration() time.Duration {
	if self.SampleRate == 0 {
		return 0
	}

	seconds := float64(self.NumSamples) / float64(self.SampleRate)

	return time.Duration(seconds * float64(time.Second))
}

func (self *Wave) Read(p []byte) (int, error) {
	if self.encodedOut == nil {
		self.encodedOut, self.encodedIn = io.Pipe()

		go func() {
			defer func() {
				self.encodedIn = nil
			}()

			if err := self.EncodeRIFF(self.encodedIn); err == nil {
				self.encodedIn.Close()
			} else {
				self.encodedIn.CloseWithError(err)
			}
		}()
	}

	return self.encodedOut.Read(p)
}

func (self *Wave) Close() error {
	defer func() {
		self.encodedOut.Close()
		self.encodedOut = nil
	}()

	return nil
}

// Write out the waveform, with RIFF headers
func (self *Wave) EncodeRIFF(out io.Writer) error {
	// File Type
	if _, err := fmt.Fprintf(out, "%s", "RIFF"); err != nil {
		return err
	}

	// Bytes in whole file
	binWrite(out, uint32(uint32(self.NumChannels)*self.NumSamples*2+8+16+12))

	if _, err := fmt.Fprintf(out, "%s", "WAVE"); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(out, "%s", "fmt "); err != nil {
		return err
	}

	binWrite(out, uint32(16))                                 // Size of Header
	binWrite(out, uint16(0x0001))                             // Sample Type (RIFF_FORMAT_PCM)
	binWrite(out, self.NumChannels)                           // Number of Channels
	binWrite(out, uint32(self.SampleRate))                    // Sample Rate
	binWrite(out, uint32(self.SampleRate*self.NumChannels*2)) // Average Bytes Per Second
	binWrite(out, uint16(self.NumChannels*2))                 // Block Align
	binWrite(out, uint16(16))                                 // Bits per Sample

	if _, err := fmt.Fprintf(out, "%s", "data"); err != nil {
		return err
	}

	binWrite(out, uint32(uint32(self.NumChannels)*self.NumSamples*2)) // Bytes in Data

	// Data Bytes
	if err := self.Encode(out); err != nil {
		return err
	}

	return nil
}

func (self *Wave) Encode(out io.Writer, sampleFn ...SampleFunc) error {
	if len(sampleFn) == 0 {
		sampleFn = []SampleFunc{SampleNoOp}
	}

	fn := sampleFn[0]

	for _, v := range self.Samples {
		if err := binWrite(out, fn(v)); err != nil {
			return err
		}
	}

	return nil
}

// Utility function to write binary LittleEndian output
func binWrite(w io.Writer, value interface{}) error {
	if err := binary.Write(w, binary.LittleEndian, value); err == nil {
		return nil
	} else {
		return nil
	}
}
