package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"regexp"
	"sync"
	"syscall"

	"github.com/gordonklaus/portaudio"
)

const sampleRate = 44100

const attenuate = 0.3

var vs voices
var specs voiceConfigs
var ts triggers

type trigger struct {
	regex *regexp.Regexp
	voice string
}

type triggers struct {
	sync.Mutex
	triggers []trigger
}

type voices struct {
	sync.Mutex
	voices []*voice
}

func (vs *voices) add(v *voice) {
	vs.Lock()
	defer vs.Unlock()

	// TODO unbound growth: use maxVoices

	// check if there's an empty slot
	for i := range vs.voices {
		if vs.voices[i] == nil {
			vs.voices[i] = v
			return
		}
	}
	// no empty slot: append
	vs.voices = append(vs.voices, v)
}

type voiceConfigs struct {
	sync.Mutex
	specs map[string]VoiceConfig
}

func (c VoiceConfig) makeVoice() (*voice, error) {
	tval, ok := c.Oscillator["type"]
	if !ok {
		return nil, fmt.Errorf("voiceConfig has no type")
	}
	t, ok := tval.(string)
	if !ok {
		return nil, fmt.Errorf("type must be string value")
	}
	var osc oscillator
	switch t {
	case "noise":
		osc = random{}
	case "sine":
		fval, ok := c.Oscillator["frequency"]
		if !ok {
			return nil, fmt.Errorf("sine oscillator must have frequency")
		}
		f, ok := fval.(float64)
		if !ok {
			return nil, fmt.Errorf("frequency must be float, was: %T", fval)
		}
		osc = sine{int(f)}
	default:
		return nil, fmt.Errorf("unknown type: %s", t)
	}
	return &voice{
		osc: osc,
		// TODO createAdr should validate, return error
		envelope: c.Envelope.createAdr(),
	}, nil
}

func (c *voiceConfigs) makeVoice(name string) (*voice, error) {
	c.Lock()
	defer c.Unlock()

	spec, ok := c.specs[name]
	if !ok {
		return nil, fmt.Errorf("can't find spec for name: %s", name)
	}
	voice, err := spec.makeVoice()
	if err != nil {
		return nil, fmt.Errorf("can't create voice: %s: %w", name, err)
	}
	return voice, nil
}

func (c *voiceConfigs) set(configs map[string]VoiceConfig) {
	c.Lock()
	defer c.Unlock()

	c.specs = configs
}

func (ts *triggers) set(triggers []Trigger) error {
	ts.Lock()
	defer ts.Unlock()

	for _, t := range triggers {
		r, err := regexp.Compile(t.Regex)
		if err != nil {
			return nil
		}
		ts.triggers = append(ts.triggers, trigger{r, t.Voice})
	}
	return nil
}

func (ts *triggers) firstMatch(s []byte) string {
	ts.Lock()
	defer ts.Unlock()

	for _, t := range ts.triggers {
		if t.regex.Match(s) {
			return t.voice
		}
	}
	return ""
}

func output(out []float32) {
	// zero buffer
	for i := range out {
		out[i] = 0
	}
	vs.Lock()
	defer vs.Unlock()
	for i, v := range vs.voices {
		// TODO in parallel?
		samples := v.play(len(out))
		for i := range out {
			out[i] += samples[i]
		}
		if v.done() {
			vs.voices[i] = nil
		}
	}
	// apply global attenuate
	for i := range out {
		out[i] *= attenuate
	}
}

func scanLines(lines chan []byte, errors chan error) {
	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		line := s.Bytes()
		// ignore len, errors
		fmt.Println(string(line))
		lines <- line
	}
	errors <- s.Err()
}

func processLines(done <-chan struct{}, errs chan<- error) {
	lines := make(chan []byte, 1)
	scannerErrs := make(chan error)
	go scanLines(lines, scannerErrs)
	for {
		// in case we always choose <-lines down below and never terminate
		// TODO is this paranoid?
		select {
		case <-done:
			return
		case <-scannerErrs:
			// stop on scanner error, ignore error
			return
		default:
		}
		select {
		case line := <-lines:
			// maybe play sound
			vName := ts.firstMatch(line)
			if vName == "" {
				continue
			}
			v, err := specs.makeVoice(vName)
			if err != nil {
				errs <- err
				continue
			}
			vs.add(v)
		case <-done:
			return
		case <-scannerErrs:
			// stop on scanner error, ignore error
			return
		}
	}
}

func main() {
	configFile := flag.String("config", "", "Path to config, created with defaults if not found.")
	flag.Parse()
	if *configFile == "" {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		return
	}
	config, err := ReadConfig(*configFile)
	if err != nil {
		log.Fatalf("can't read config: %v because: %v", configFile, err)
	}
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	err = portaudio.Initialize()
	if err != nil {
		log.Fatalf("can't init portaudion: %v", err)
	}
	// ignore Terminate error
	defer portaudio.Terminate()
	// mono out
	stream, err := portaudio.OpenDefaultStream(0, 1, sampleRate, portaudio.FramesPerBufferUnspecified, output)
	if err != nil {
		log.Fatalf("can't open default stream: %v", err)
	}
	err = stream.Start()
	if err != nil {
		log.Fatalf("can't start stream: %v", err)
	}
	// ignore Close error
	defer stream.Close()

	configs := make(chan *Config)
	done := make(chan struct{})
	defer close(done)
	errors := make(chan error)
	if config.WatchConfig {
		err := Watch(*configFile, configs, errors, done)
		if err != nil {
			log.Fatalf("can't start watcher: %v", err)
		}
	}

	// apply initial config
	specs.set(config.Voices)
	err = ts.set(config.Triggers)
	if err != nil {
		log.Fatalf("trigger config error: %v", err)
	}

	// report errors to stderr
	go func() {
		select {
		case err := <-errors:
			fmt.Fprintf(os.Stderr, "error: %v", err)
		case <-done:
			return
		}
	}()

	// scan lines, trigger sounds
	go processLines(done, errors)

	for {
		select {
		// handle config changes
		case c := <-configs:
			err := ts.set(c.Triggers)
			if err != nil {
				errors <- err
			}
			specs.set(c.Voices)
		// block until SIGINT | SIGTERM
		case <-signals:
			if err != nil {
				log.Fatalf("can't stop stream: %v", err)
			}
			fmt.Println("exiting")
			os.Exit(0)
		}
	}
}
