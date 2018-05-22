// Copyright 2013, Carnegie Mellon University. All Rights Reserved.
// Use of this code is governed by a BSD-style license that can be
// found in the LICENSE file.

// Author: Alok Parlikar <aup@cs.cmu.edu>

// Use the CMU Flite Text-To-Speech Engine from Go
// +build linux,cgo

package goflite

/*
 #cgo CFLAGS: -I ${SRCDIR} -I${SRCDIR}/dep/flite/include
 #cgo linux,amd64 LDFLAGS: ${SRCDIR}/dep/flite/build/x86_64-linux-gnu/lib/libflite_cmu_us_slt.a ${SRCDIR}/dep/flite/build/x86_64-linux-gnu/lib/libflite_cmulex.a ${SRCDIR}/dep/flite/build/x86_64-linux-gnu/lib/libflite_usenglish.a ${SRCDIR}/dep/flite/build/x86_64-linux-gnu/lib/libflite_cmu_indic_lex.a ${SRCDIR}/dep/flite/build/x86_64-linux-gnu/lib/libflite_cmu_indic_lang.a ${SRCDIR}/dep/flite/build/x86_64-linux-gnu/lib/libflite.a -lm

 #cgo linux,386   LDFLAGS: ${SRCDIR}/dep/flite/build/i386-linux-gnu/lib/libflite_cmu_us_slt.a ${SRCDIR}dep/flite/build/i386-linux-gnu/lib/libflite_cmulex.a ${SRCDIR}/dep/flite/build/i386-linux-gnu/lib/libflite_usenglish.a  ${SRCDIR}dep/flite/build/i386-linux-gnu/lib/libflite_cmu_indic_lex.a ${SRCDIR}/dep/flite/build/i386-linux-gnu/lib/libflite_cmu_indic_lang.a ${SRCDIR}/dep/flite/build/i386-linux-gnu/lib/libflite.a -lm

 #include <flitewrap.h>
 #include <flite.h>
*/
import "C"

import (
	"errors"
	"unsafe"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger(`goflite`)

var voices *voxbase // List of available voices stored here
var isInitialized bool

// Initialize Flite
func initFlite() {
	if !isInitialized {
		C.flitewrap_init()
		voices = newVoxBase()
		isInitialized = true
	}
}

// If you have built flite voices and have the flitevox files
// generated, use this function to add them to goflite. Provide a name
// to the voice being added and a path to the location of the flitevox
// file.  Prefer absolute pathname.
func AddVoice(name, path string) error {
	initFlite()

	return voices.addVoice(name, path)
}

// Run Text to Speech on a given text with a selected voice and return
// Wave data. If voicename is empty, a default voice will be used for
// the speech synthesis.
func TextToWave(text string, v flitevoice) (*Wave, error) {
	var w *Wave            // Waveform to Return
	var cstwav *C.cst_wave // Flite's wave structure

	initFlite()

	ctext := C.CString(text)
	defer C.free(unsafe.Pointer(ctext))

	cstwav = C.flite_text_to_wave(ctext, v)
	if cstwav == nil {
		return nil, errors.New("Speech synthesis failed")
	}

	log.Debugf("voice: %+v", v.features.head.val)

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
