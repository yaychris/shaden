## Lumen

Lumen is a modular audio synthesizer. Patches for the synthesizer are written in a Lisp dialect. A REPL and HTTP
interface is provided for interacting with the synthesizer in real-time. I started this project as a way of learning
more about digital signal processing and music theory. *Consider this an art project*.

### Highlights

- Lisp interpreter for creating patches
- [Large collection of builtin Units](https://github.com/brettbuddin/lumen/wiki/Units)
- [Music theory primitives](https://github.com/brettbuddin/lumen/wiki/Values#music-theory)
- MIDI controller and clock input
- Single-sample feedback loops
- Vim plugin for sending snippets of code over to the synth for evaluation

## Dependencies

- [Go 1.9](http://golang.org)+
- [PortAudio](http://www.portaudio.com/)
- [PortMIDI](http://portmedia.sourceforge.net/portmidi/)

On macOS you can install these dependencies with: `brew install go portaudio portmidi`

## Getting Started

### Install

    $ go get -u buddin.us/lumen
	$ lumen -h
	Usage of lumen:
  	-addr string
        	http address to serve (default ":5000")
  	-device-frame int
        	frame size used when writing to audio device (default 1024)
  	-device-in int
        	input device
  	-device-latency string
        	latency setting for audio device (default "low")
  	-device-list
        	list all devices
  	-device-out int
        	output device (default 1)
  	-repl
        	REPL
  	-seed int
        	random seed
	flag: help requested

### CLI Usage

#### REPL

    $ lumen -repl
    > (define gen (unit/gen))
    > (-> gen (table :freq (hz 300)))
    > (emit (<- gen :sine))

#### Load File

    $ lumen examples/frequency-modulation.lisp

#### HTTP

    $ lumen
    $ curl -X POST http://127.0.0.1:5000/eval -d "(define source (unit/gen)) ; ..."

This is my preferred way of interacting with the synthesizer. I've written a small Vim plugin that can send over
snippets of Lisp code to the program for evaluation. You can get [that plugin here](extra/lumen.vim).

The HTTP interface is limited to Lisp evaluation at the moment, but I have hopes of providing an API for direct graph
manipulation via HTTP.

### Lisp

For a more information about the Lisp dialect bundled with Lumen, [check out the wiki](https://github.com/brettbuddin/lumen/wiki).