package main

import (
	"math"
	"math/rand"
)

type oscillator interface {
	samples(timeSamples int, numSamples int) []float32
}

type adrSampleConfig struct {
	AttackSamples  int
	AttackValue    float32
	DecaySamples   int
	DecayValue     float32
	ReleaseSamples int
}

type adr struct {
	adrSampleConfig
	attackSlope   float32
	decaySlope    float32
	decayOffset   float32
	releaseSlope  float32
	releaseOffset float32
}

type point struct {
	x, y float32
}

type line struct {
	start, end point
}

func (l line) slope() float32 {
	return (l.end.y - l.start.y) / (l.end.x - l.start.x)
}

func (c AdrConfig) createAdr() adr {
	attackSamples := sampleRate * c.AttackSeconds
	attack := line{
		point{0, 0},
		point{attackSamples, c.AttackValue},
	}
	decaySamples := sampleRate * c.DecaySeconds
	decay := line{
		attack.end,
		point{attack.end.x + decaySamples, c.DecayValue},
	}
	releaseSamples := sampleRate * c.ReleaseSeconds
	release := line{
		decay.end,
		point{decay.end.x + releaseSamples, 0},
	}
	ds := decay.slope()
	rs := release.slope()
	return adr{
		adrSampleConfig{
			int(attackSamples),
			c.AttackValue,
			int(decaySamples),
			c.DecayValue,
			int(releaseSamples),
		},
		attack.slope(),
		ds,
		ds*-decay.start.x + decay.start.y,
		rs,
		rs*-release.start.x + release.start.y,
	}
}

type random struct{}

func (r random) samples(timeSamples int, numSamples int) []float32 {
	samples := make([]float32, numSamples)
	for i := 0; i < numSamples; i++ {
		samples[i] = rand.Float32()
	}
	return samples
}

type sine struct {
	frequency int
}

func (s sine) samples(timeSamples int, numSamples int) []float32 {
	samples := make([]float32, numSamples)
	period := (2 * math.Pi) / (1 / float32(s.frequency))
	for i := 0; i < numSamples; i++ {
		samples[i] = float32(math.Sin(float64(period * (float32(timeSamples+i) / sampleRate))))
	}
	return samples
}

func (env *adr) done(timeSamples int) bool {
	return timeSamples > env.AttackSamples+env.DecaySamples+env.ReleaseSamples
}

func (env *adr) apply(timeSamples int, samples []float32) {
	for i := 0; i < len(samples); i++ {
		time := i + timeSamples
		threshold := env.AttackSamples
		if env.done(time) {
			samples[i] = 0
			continue
		}
		if time < threshold {
			// attack active
			samples[i] *= env.attackSlope * float32(time)
			continue
		}
		threshold += env.DecaySamples
		if time < threshold {
			// decay active
			samples[i] *= env.decaySlope*float32(time) + env.decayOffset
			continue
		}
		threshold += env.ReleaseSamples
		if time < threshold {
			// release active
			samples[i] *= env.releaseSlope*float32(time) + env.releaseOffset
			continue
		}
	}
}

type voice struct {
	osc         oscillator
	timeSamples int
	envelope    adr
}

func (v *voice) done() bool {
	if v == nil {
		return true
	}
	return v.envelope.done(v.timeSamples)
}

func (v *voice) play(numSamples int) []float32 {
	if v == nil {
		return make([]float32, numSamples)
	}
	samples := v.osc.samples(v.timeSamples, numSamples)
	v.envelope.apply(v.timeSamples, samples)
	v.timeSamples += numSamples
	return samples
}

func newVoice(osc oscillator, envelopeConfig AdrConfig) voice {
	return voice{
		osc,
		0,
		envelopeConfig.createAdr(),
	}
}
