# SimpleSynth

A minimal, real-time midi synthesizer in Go.  Currently connects to a midi device and acts as a sine wave oscillator based on incoming MIDI Note-On and Note-Off messages.

## Why?

I wanted the basic skeleton of a performant synthesizer to be able to quickly experiment with DSP concepts.  Go is a super simple, statically typed language so there's not a lot to hide behind. The dependencies used are also fairly low-level but easy to read and understand if necessary.

I also felt like there were very few simple examples of a fully implemented synthesizer.  This shows how the fundamental OS's audio callback buffer can be written into based on incoming MIDI NoteOn and NoteOff messages.

## Usage

Using "go run":

`go run . -ls`: Lists all available midi devices

Then using the index printed from the "ls" command to specify a device to use:

`go run . -d <index>`:  this acts as a simply sine-wave synth

`go run . -d <index> -m`: print the incoming midi messages

## Requirements and References:

* [Oto](https://github.com/hajimehoshi/oto): a fantastic low-level audio library in Go.

* [PortMIDI](https://github.com/rakyll/portmidi): The PortMIDI wrapper in Go

     This does require portmidi to be installed, as mentioned in its Readme,
     ```
     apt-get install libportmidi-dev
     # or
     brew install portmidi
     ```
     Or get it from the source: http://portmedia.sourceforge.net/portmidi/

* [How I built an audio library using the composite pattern and higher-order functions](https://faiface.github.io/post/how-i-built-audio-lib-composite-pattern/): I really like the author's compositional approach to building an audio library and definitely pulled from it here.
