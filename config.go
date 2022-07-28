package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
)

const defaultConfig = `
{
	"maxVoices": 50,
	"watchConfig": true,
	"voices": {
		"geiger": {
			"osc": { "type": "noise" },
			"env": {
				"attackSamples": 44100,
				"attackValue": 1,
				"decaySamples": 0,
				"decayValue": 1,
				"releaseSamples": 44100
			}
		},
		"phone": {
			"osc": { "type": "sine", "frequency": 425 },
			"env": {
				"attackSamples": 44100,
				"attackValue": 1,
				"decaySamples": 0,
				"decayValue": 1,
				"releaseSamples": 44100
			}
		}
	},
	"triggers": [
		{ "regex": "hey", "voice": "geiger" },
		{ "regex": "ho", "voice": "phone" }
	]
}
`

type StaticConfig struct {
	MaxVoices   int  `json:"maxVoices"`
	WatchConfig bool `json:"watchConfig"`
}

type Trigger struct {
	Regex string `json:"regex"`
	Voice string `json:"voice"`
}

type VoiceConfig struct {
	Oscillator map[string]interface{} `json:"osc"`
	Envelope   AdrConfig              `json:"env"`
}

type DynamicConfig struct {
	Triggers []Trigger              `json:"triggers"`
	Voices   map[string]VoiceConfig `json:"voices"`
}

type Config struct {
	StaticConfig
	DynamicConfig
}

func ReadConfig(p string) (*Config, error) {
	if _, err := os.Stat(p); errors.Is(err, os.ErrNotExist) {
		err = ioutil.WriteFile(p, []byte(defaultConfig), 0644)
		if err != nil {
			return nil, fmt.Errorf("can't write defaultConfig: %w", err)
		}
	}
	f, err := os.Open(p)
	if err != nil {
		return nil, fmt.Errorf("can't open config: %w", err)
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("can't read config: %w", err)
	}
	var c Config
	err = json.Unmarshal(data, &c)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling: %w", err)
	}
	return &c, nil
}
