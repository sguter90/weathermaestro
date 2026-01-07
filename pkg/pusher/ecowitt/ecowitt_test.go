package ecowitt

import (
	"net/url"
	"testing"
)

func TestParse(t *testing.T) {
	p := &Pusher{}

	params := url.Values{}
	params.Set("tempf", "72.5")
	params.Set("humidity", "65")
	params.Set("PASSKEY", "test123")

	data, err := p.Parse(params)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if data.TempOutF != 72.5 {
		t.Errorf("Expected TempOutF=72.5, got %f", data.TempOutF)
	}

	expectedC := (72.5 - 32) * 5 / 9
	if data.TempOutC != expectedC {
		t.Errorf("Expected TempOutC=%f, got %f", expectedC, data.TempOutC)
	}
}
