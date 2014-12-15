package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/bmizerany/assert"
)

func TestExpandKeyIncrementsHistogram(t *testing.T) {
	key, action := expandKey("timers.event.histogram.bin_<0.00")

	assert.Equal(t, "increment", action)
	assert.Equal(t, "event.histogram.bin_<0.00", key)
}

func TestExpandKeyTrimsAction(t *testing.T) {
	tests := map[string]string{
		"counts.times_occurred": "times_occurred",
		"timers.event.timing":   "event.timing",
		"gauges.number.gauged":  "number.gauged",
		"sets.some.set.test":    "some.set.test",
	}

	for key, expected := range tests {
		actual, _ := expandKey(key)
		assert.Equal(t, expected, actual)
	}
}

func TestExpandKeyMapsAction(t *testing.T) {
	tests := map[string]string{
		"gauges.X": "gauge",
		"counts.X": "increment",
		"timers.X": "gauge",
		"sets.X":   "gauge_absolute",
	}

	for key, expected := range tests {
		_, actual := expandKey(key)
		assert.Equal(t, expected, actual)
	}
}

func TestExpandKeyReturnsUnkownAction(t *testing.T) {
	_, action := expandKey("hoopie.froods")
	assert.Equal(t, "", action)
}

func TestFunnel(t *testing.T) {
	input := bytes.NewBuffer(make([]byte, 0))
	input.Write([]byte("counts.times_occurred|20.000000|1391189888\n"))
	input.Write([]byte("timers.event.p95|1.899065|1391189890\n"))
	input.Write([]byte("gauges.number|200.000000|1391189899\n"))
	input.Write([]byte("sets.mappings|4|1391189900\n"))

	output := bytes.NewBuffer(make([]byte, 0))
	err := funnel(input, output)
	if err != nil {
		t.Error(err)
	}

	lines := strings.Split(output.String(), "\n")

	assert.Equal(t, 5, len(lines))
	assert.Equal(t, "increment times_occurred 20.000000 1391189888", lines[0])
	assert.Equal(t, "gauge event.p95 1.899065 1391189890", lines[1])
	assert.Equal(t, "gauge number 200.000000 1391189899", lines[2])
	assert.Equal(t, "gauge_absolute mappings 4 1391189900", lines[3])
	assert.Equal(t, "", lines[4])
}

func TestFunnelPrefix(t *testing.T) {
	// Global state sucks
	before_config := Config

	Config = &config{
		Prefix:  "blarg.",
		Postfix: ".baz",
	}

	input := bytes.NewBufferString("counts.numbers|1|1400000000\n")
	output := bytes.NewBuffer(make([]byte, 0))

	if err := funnel(input, output); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "increment blarg.numbers.baz 1 1400000000\n", output.String())

	Config = before_config
}
