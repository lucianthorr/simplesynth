module github.com/lucianthorr/simplesynth

go 1.18

replace github.com/rakyll/portmidi => ../portmidi

require (
	github.com/hajimehoshi/oto/v2 v2.1.0
	github.com/rakyll/portmidi v0.0.0-00010101000000-000000000000
)

require golang.org/x/sys v0.0.0-20220408201424-a24fb2fb8a0f // indirect
