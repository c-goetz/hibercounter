package main

import (
	"encoding/json"
	"testing"
)

func TestDefaultConfigUnmarshal(t *testing.T) {
	var c Config
	err := json.Unmarshal([]byte(defaultConfig), &c)
	if err != nil {
		t.Fatalf("error unmarshalling: %v", err)
	}
	if c.MaxVoices == 0 {
		t.Fatalf("expected maxVoices to be set")
	}
	if c.Voices["geiger"].Oscillator == nil {
		t.Fatalf("expected geiger voice")
	}
}
