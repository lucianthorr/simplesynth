package main

import (
	"flag"
	"fmt"
	"log"
	"math"

	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/hajimehoshi/oto/v2"
	"github.com/rakyll/portmidi"
)

var (
	listFlag    = flag.Bool("ls", false, "list available input devices")
	monitorFlag = flag.Bool("m", false, "run a simple midi monitor")
	deviceFlag  = flag.Int("d", -1, "device to listen")
)

type AudioContext struct {
	SampleRate      int
	NumChannels     int
	BitDepthInBytes int
}

type midiHandler func() []portmidi.Event                       // pulls and returns a list of midi events
type midiTranslator func() (freq, velocity float64, gate bool) // translates those events into parameters for a sound generator
type soundGen func(buf []byte) (int, error)                    // generates the sineWave and reads it to a buffer

func (sg soundGen) Read(buf []byte) (int, error) {
	return sg(buf)
}

func main() {
	flag.Parse()

	// midi bootstrap
	portmidi.Initialize()
	defer portmidi.Terminate()
	if *listFlag {
		listMidiDevices()
		return
	}
	if 0 < *deviceFlag && *deviceFlag < portmidi.CountDevices()-1 {
		in, err := portmidi.NewInputStream(portmidi.DeviceID(*deviceFlag-1), 64)
		if err != nil {
			log.Fatal(fmt.Errorf("Error creating stream: %s", err.Error()))
		}
		defer in.Close()

		in.Listen()
		midiHandler := makeMidiHandler(in)
		midiTranslator := makeMidiTranslator(midiHandler)
		if *monitorFlag {
			runMidiMonitor(midiHandler) // midi testing
		}

		// audio bootstrap
		ac := &AudioContext{
			SampleRate:      48000,
			NumChannels:     2,
			BitDepthInBytes: 2, // 16-bit
		}

		ctx, ready, err := oto.NewContext(ac.SampleRate, ac.NumChannels, ac.BitDepthInBytes)
		if err != nil {
			log.Fatal(err)
		}
		<-ready
		// connecting the pieces
		p := ctx.NewPlayer(makeSineGen(ac, midiTranslator))
		defer runtime.KeepAlive(p)
		p.(oto.BufferSizeSetter).SetBufferSize(512 * ac.NumChannels * ac.BitDepthInBytes) // 2048
		p.Play()

		wait := make(chan os.Signal, 1)
		signal.Notify(wait, os.Interrupt, syscall.SIGTERM)
		<-wait

	} else if *monitorFlag {
		listMidiDevices()
		fmt.Println("Specify an input device to monitor")
	}
}

// listDevices currently available, use the index shown to specify which device you'd like to use
func listMidiDevices() {
	for i := 0; i < portmidi.CountDevices(); i++ {
		info := portmidi.Info(portmidi.DeviceID(i))
		if info.IsInputAvailable {
			fmt.Printf("%d: %s\n", i+1, info.Name)
		}
	}
}

// builds a function to poll midi events
func makeMidiHandler(in *portmidi.Stream) midiHandler {
	return func() []portmidi.Event {
		res, err := in.Poll()
		if err != nil {
			log.Fatal(fmt.Errorf("Error polling: %s", err.Error()))
		}
		filteredEvents := []portmidi.Event{}
		if res {
			events, err := in.Read(1024)
			if err != nil {
				log.Fatal(fmt.Errorf("Error reading: %s", err.Error()))
			}
			for i := range events {
				if 0x08 <= events[i].Status&0xF0 && events[i].Status&0xF0 < 0xF0 {
					// filters out sysex and system real time messages
					filteredEvents = append(filteredEvents, events[i])
				}
			}
		}
		return filteredEvents
	}
}

// runMidiMonitor simply prints the midi messages received, for testing
func runMidiMonitor(handler midiHandler) {
	for {
		events := handler()
		for i := range events {
			e := events[i]
			fmt.Printf("ts: %d\tstatus: %d\tdata1: %d\tdata2: %d\n", e.Timestamp, e.Status, e.Data1, e.Data2)
		}
	}
}

// builds a functions to convert midi events into a frequency and gate
func makeMidiTranslator(handler midiHandler) midiTranslator {
	note := int64(0)
	velocity := float64(0)
	gate := false
	return func() (float64, float64, bool) {
		events := handler()
		for i := range events {
			if events[i].Status == 0x90 { // NOTE ON
				gate = true
				note = events[i].Data1
				velocity = float64(events[i].Data2) / 128.0
			}
			if events[i].Status == 0x80 { // NOTE OFF
				if events[i].Data1 == note {
					gate = false
					velocity = 0.0
				}
			}
		}
		return NOTE_MAP[note], velocity, gate
	}
}

func makeSineGen(ac *AudioContext, translator midiTranslator) soundGen {
	var lastFreq float64
	var lastVelocity float64
	var lastGate bool
	var pos float64
	return func(buf []byte) (int, error) {
		bytesRead := 0
		bytesPerSample := ac.BitDepthInBytes * ac.NumChannels
		numSamples := len(buf) / bytesPerSample
		deltaT := float64(1) / float64(ac.SampleRate)
		for sampleIdx := 0; sampleIdx < numSamples; sampleIdx++ {
			freq, velocity, gate := translator()

			if gate && !lastGate {
				pos = 0
			}

			if freq != lastFreq { // resolve clicking on new notes and between frequency changes
				pos = (lastFreq * pos) / freq
			}

			if gate {
				velocity *= 0.8 // scale the volume down a little
			} else {
				velocity = lastVelocity * 0.9995 // decay
			}

			b := int16(math.Sin(2*math.Pi*float64(freq)*pos) * (math.MaxInt16 - 1) * velocity)

			for channelIdx := 0; channelIdx < ac.NumChannels; channelIdx++ {
				idx := (bytesPerSample * sampleIdx) + (channelIdx * ac.BitDepthInBytes)
				buf[idx] = byte(b)
				buf[idx+1] = byte(b >> 8)
				bytesRead = idx + 2
			}

			lastFreq = freq
			lastVelocity = velocity
			lastGate = gate
			pos += deltaT
		}
		return bytesRead, nil
	}
}
