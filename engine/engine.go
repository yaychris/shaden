package engine

import (
	"fmt"
	"sync/atomic"
	"time"

	"buddin.us/shaden/dsp"
	"buddin.us/shaden/graph"
	"buddin.us/shaden/unit"
)

// Engine is the connection of the synthesizer to PortAudio
type Engine struct {
	backend              Backend
	graph                *graph.Graph
	unit                 *unit.Unit
	processors           []unit.FrameProcessor
	messages             chan *Message
	errors, stop         chan error
	input                []float64
	lout, rout           []float64
	chunks               int
	singleSampleDisabled bool
	stopping             *uint32
}

// Backend is a low-level callback-based engine
type Backend interface {
	Start(func([]float32, [][]float32)) error
	Stop() error
	FrameSize() int
}

// New returns a new Sink
func New(backend Backend, singleSampleDisabled bool) (*Engine, error) {
	var stopping uint32 = 0

	sinkUnit, sink := newSink(&stopping)
	g := graph.New()
	if err := sinkUnit.Attach(g); err != nil {
		return nil, err
	}

	return &Engine{
		backend:              backend,
		graph:                g,
		messages:             make(chan *Message),
		unit:                 sinkUnit,
		lout:                 sink.left.out,
		rout:                 sink.right.out,
		errors:               make(chan error),
		stop:                 make(chan error),
		input:                make([]float64, dsp.FrameSize),
		chunks:               int(dsp.Float64(backend.FrameSize()) / dsp.FrameSize),
		singleSampleDisabled: singleSampleDisabled,
		stopping:             &stopping,
	}, nil
}

// UnitBuilders returns all unit.BuildFuncs for Units provided by the Engine.
func (e *Engine) UnitBuilders() map[string]unit.BuildFunc {
	return unitBuilders(e)
}

// Reset clears the state of the Engine. This includes clearing the audio graph.
func (e *Engine) Reset() error {
	e.graph = graph.New()

	sinkUnit, sink := newSink(e.stopping)
	if err := sinkUnit.Attach(e.graph); err != nil {
		return err
	}
	e.unit = sinkUnit
	e.lout = sink.left.out
	e.rout = sink.right.out

	e.sort()

	return nil
}

// Messages provides a send-only channel that can be used to execute code on the main audio goroutine
func (e *Engine) Messages() chan<- *Message {
	return e.messages
}

// Errors returns a channel that expresses any errors during operation of the Engine
func (e *Engine) Errors() <-chan error {
	return e.errors
}

// Run starts the Engine; running the audio stream
func (e *Engine) Run() {
	if err := e.backend.Start(e.callback); err != nil {
		e.errors <- err
		return
	}
	<-e.stop

	// Mark the flag for shutdown so that the sink's outputs know we are leaving. This will cause them to perform a
	// fade-out while we wait. Not imperative that we synchronize things so a sleep will do.
	atomic.AddUint32(e.stopping, 1)
	time.Sleep(150 * time.Millisecond)

	e.stop <- e.backend.Stop()
}

// Stop shuts down the Engine
func (e *Engine) Stop() error {
	e.stop <- nil
	err := <-e.stop
	close(e.errors)
	close(e.stop)
	return err
}

func (e *Engine) call(action interface{}) (interface{}, error) {
	switch fn := action.(type) {
	case func(e *Engine) (interface{}, error):
		return fn(e)
	case func(g *graph.Graph) (interface{}, error):
		return fn(e.graph)
	default:
		return nil, fmt.Errorf("unhandled function type %T", action)
	}
}

func (e *Engine) sort() {
	processors := e.processors[:0]
	for _, v := range e.graph.Sorted() {
		e.collectProcessor(&processors, v)
	}
	e.processors = processors
	e.graph.AckChange()
}

func (e *Engine) collectProcessor(processors *[]unit.FrameProcessor, nodes []*graph.Node) {
	if len(nodes) > 1 {
		e.collectGroup(processors, nodes)
		return
	}

	first := nodes[0]
	if in, ok := first.Value.(*unit.In); ok && !e.singleSampleDisabled {
		in.Mode = unit.Block
	}
	if p, ok := first.Value.(frameProcessor); ok && p.ExternalNeighborCount() > 0 {
		if isp, ok := p.(condProcessor); ok {
			if isp.IsProcessable() {
				*processors = append(*processors, p)
			}
		} else {
			*processors = append(*processors, p)
		}
	}
}

func (e *Engine) collectGroup(processors *[]unit.FrameProcessor, nodes []*graph.Node) {
	var g group
	for _, w := range nodes {
		if in, ok := w.Value.(*unit.In); ok && !e.singleSampleDisabled {
			in.Mode = unit.Sample
		}
		if p, ok := w.Value.(unit.SampleProcessor); ok {
			if isp, ok := p.(condProcessor); ok {
				if isp.IsProcessable() {
					g.processors = append(g.processors, p)
				}
			} else {
				g.processors = append(g.processors, p)
			}
		}
	}
	*processors = append(*processors, g)
}

func (e *Engine) handle(msg *Message) {
	start := time.Now()
	data, err := e.call(msg.Action)

	if err == nil && e.graph.HasChanged() {
		e.sort()
	}

	if msg.Reply != nil {
		msg.Reply <- &Reply{
			Duration: time.Since(start),
			Data:     data,
			Error:    err,
		}
	}
}

// callback is the callback function provided to PortAudio; it drives the entire synthesiser.
func (e *Engine) callback(in []float32, out [][]float32) {
	for k := 0; k < e.chunks; k++ {
		select {
		case msg := <-e.messages:
			e.handle(msg)
		default:
		}

		offset := int(dsp.FrameSize * k)
		for i := 0; i < int(dsp.FrameSize); i++ {
			e.input[i] = float64(in[offset+i])
		}
		for _, p := range e.processors {
			p.ProcessFrame(dsp.FrameSize)
		}
		for i := range out {
			for j := 0; j < dsp.FrameSize; j++ {
				if i%2 == 0 {
					out[i][offset+j] = float32(e.lout[j])
				} else {
					out[i][offset+j] = float32(e.rout[j])
				}
			}
		}
	}
}

type frameProcessor interface {
	unit.FrameProcessor
	ExternalNeighborCount() int
}

type condProcessor interface {
	IsProcessable() bool
}

type group struct {
	processors []unit.SampleProcessor
}

func (g group) ProcessFrame(n int) {
	for i := 0; i < n; i++ {
		for _, p := range g.processors {
			p.ProcessSample(i)
		}
	}
}
