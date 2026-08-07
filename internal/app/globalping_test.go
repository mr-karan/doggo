package app

import (
	"errors"
	"testing"
)

func TestGlobalpingMeasurementRequiresTarget(t *testing.T) {
	a := &App{}

	if _, err := a.GlobalpingMeasurement(); !errors.Is(err, ErrGlobalpingTargetRequired) {
		t.Fatalf("GlobalpingMeasurement() error = %v, want %v", err, ErrGlobalpingTargetRequired)
	}
}
