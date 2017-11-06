// Copyright 2013, Carnegie Mellon University. All Rights Reserved.
// Use of this code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Author: Gary Hetzel <garyhetzel@gmail.com>
package goflite

// +build linux,cgo

/*
 #cgo CFLAGS: -I ${SRCDIR} -I${SRCDIR}/dep/flite/include
 #cgo linux,amd64 LDFLAGS: ${SRCDIR}/dep/flite/build/x86_64-linux-gnu/lib/libflite_cmu_us_slt.a ${SRCDIR}/dep/flite/build/x86_64-linux-gnu/lib/libflite_cmulex.a ${SRCDIR}/dep/flite/build/x86_64-linux-gnu/lib/libflite_usenglish.a ${SRCDIR}/dep/flite/build/x86_64-linux-gnu/lib/libflite_cmu_indic_lex.a ${SRCDIR}/dep/flite/build/x86_64-linux-gnu/lib/libflite_cmu_indic_lang.a ${SRCDIR}/dep/flite/build/x86_64-linux-gnu/lib/libflite.a -lm

 #cgo linux,386   LDFLAGS: ${SRCDIR}/dep/flite/build/i386-linux-gnu/lib/libflite_cmu_us_slt.a ${SRCDIR}dep/flite/build/i386-linux-gnu/lib/libflite_cmulex.a ${SRCDIR}/dep/flite/build/i386-linux-gnu/lib/libflite_usenglish.a  ${SRCDIR}dep/flite/build/i386-linux-gnu/lib/libflite_cmu_indic_lex.a ${SRCDIR}/dep/flite/build/i386-linux-gnu/lib/libflite_cmu_indic_lang.a ${SRCDIR}/dep/flite/build/i386-linux-gnu/lib/libflite.a -lm

 #include <flitewrap.h>
 #include <flite.h>
*/
import "C"

import (
	"fmt"
	"time"
	"unsafe"

	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/wav"
)

var DefaultClientName = `goflite`
var DefaultStreamName = `Flite Voice Synthesizer`
var DefaultPostFinishDelay = 150 * time.Millisecond

type Synthesizer struct {
	StreamName      string
	PostFinishDelay time.Duration
	voice           flitevoice
	voicename       string
}

func NewSynthesizer() (*Synthesizer, error) {
	initFlite()

	voices.mutex.RLock()
	defer voices.mutex.RUnlock()

	if voice, ok := voices.flitevox[DefaultVoiceName]; ok {
		return &Synthesizer{
			StreamName:      DefaultStreamName,
			PostFinishDelay: DefaultPostFinishDelay,
			voice:           voice,
			voicename:       DefaultVoiceName,
		}, nil
	} else {
		return nil, fmt.Errorf("Unknown default voice %q", DefaultVoiceName)
	}
}

func (self *Synthesizer) Say(input string) error {
	if wave, err := self.Synthesize(input); err == nil {
		if streamer, format, err := wav.Decode(wave); err == nil {
			if err := speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10)); err == nil {
				done := make(chan struct{})

				speaker.Play(beep.Seq(streamer, beep.Callback(func() {
					if self.PostFinishDelay > 0 {
						select {
						case <-time.After(self.PostFinishDelay):
							break
						}
					}

					close(done)
				})))

				<-done
				return nil
			} else {
				return err
			}
		} else {
			return err
		}
	} else {
		return err
	}
}

func (self *Synthesizer) Synthesize(input string) (*Wave, error) {
	var w *Wave            // Waveform to Return
	var cstwav *C.cst_wave // Flite's wave structure

	initFlite()

	ctext := C.CString(input)
	defer C.free(unsafe.Pointer(ctext))

	cstwav = C.flite_text_to_wave(ctext, self.voice)

	if cstwav == nil {
		return nil, fmt.Errorf("Speech synthesis failed")
	}

	num_samples := uint32(cstwav.num_samples)

	w = &Wave{
		SampleRate:  uint16(cstwav.sample_rate),
		NumSamples:  num_samples,
		NumChannels: uint16(cstwav.num_channels),
		Samples:     make([]uint16, num_samples),
	}

	C.copy_wav_into_slice(cstwav, (*C.short)(unsafe.Pointer(&(w.Samples[0]))))
	C.delete_wave(cstwav)

	return w, nil
}

func (self *Synthesizer) SetVoice(name string) error {
	voices.mutex.RLock()
	voice, ok := voices.flitevox[name]
	voices.mutex.RUnlock()

	if ok {
		self.voice = voice
		self.voicename = name
		return nil
	} else {
		return fmt.Errorf("Cannot locate voice %q", name)
	}
}

func (self *Synthesizer) SetFloatFeature(name string, value float64) {
	C.flite_feat_set_float(self.voice.features, C.CString(name), C.float(value))
}

func (self *Synthesizer) SetIntFeature(name string, value int64) {
	C.flite_feat_set_int(self.voice.features, C.CString(name), C.int(value))
}

func (self *Synthesizer) SetFeature(name string, value string) {
	C.flite_feat_set_string(self.voice.features, C.CString(name), C.CString(value))
}
