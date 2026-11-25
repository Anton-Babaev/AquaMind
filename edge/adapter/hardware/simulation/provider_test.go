package simulation

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func writeTempCSV(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "scenario.csv")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp csv: %v", err)
	}
	return path
}

const sampleCSV = `offset_sec,temp_c,flow_lpm,o2_mgl,ph
0,24.0,12.0,7.0,7.2
60,23.0,11.0,6.0,7.1
120,22.0,10.0,5.0,7.0
`

func TestNewProvider_InvalidSpeed(t *testing.T) {
	path := writeTempCSV(t, sampleCSV)
	if _, err := NewProvider(path, 0, true); err == nil {
		t.Fatal("expected error for speed=0")
	}
}

func TestNewProvider_MissingFile(t *testing.T) {
	if _, err := NewProvider("does-not-exist.csv", 1, true); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestProvider_ReadFirstRow(t *testing.T) {
	path := writeTempCSV(t, sampleCSV)
	p, err := NewProvider(path, 1, true)
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}

	r, err := p.Read(context.Background())
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if r.TempC != 24.0 || r.FlowLPM != 12.0 || r.O2MgL != 7.0 || r.PH != 7.2 {
		t.Errorf("unexpected first reading: %+v", r)
	}
	if r.At.IsZero() {
		t.Error("reading time must be set")
	}
}

func TestProvider_ContextCancelled(t *testing.T) {
	path := writeTempCSV(t, sampleCSV)
	p, err := NewProvider(path, 1, true)
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := p.Read(ctx); err == nil {
		t.Fatal("expected context error")
	}
}

func TestIndexFor(t *testing.T) {
	path := writeTempCSV(t, sampleCSV)
	p, err := NewProvider(path, 1, true)
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}

	tests := []struct {
		elapsed float64
		want    int
	}{
		{0, 0},
		{30, 0},
		{60, 1},
		{90, 1},
		{120, 2},
		{119, 1},
	}
	for _, tt := range tests {
		if got := p.indexFor(tt.elapsed); got != tt.want {
			t.Errorf("indexFor(%v) = %d, want %d", tt.elapsed, got, tt.want)
		}
	}
}

func TestIndexFor_Loop(t *testing.T) {
	path := writeTempCSV(t, sampleCSV)
	p, err := NewProvider(path, 1, true)
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}
	// duration = 120. elapsed 130 → 10 → строка 0.
	if got := p.indexFor(130); got != 0 {
		t.Errorf("indexFor(130) with loop = %d, want 0", got)
	}
}

func TestIndexFor_NoLoop(t *testing.T) {
	path := writeTempCSV(t, sampleCSV)
	p, err := NewProvider(path, 1, false)
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}
	// duration = 120. elapsed 130 без loop → -1 (конец сценария).
	if got := p.indexFor(130); got != -1 {
		t.Errorf("indexFor(130) without loop = %d, want -1", got)
	}
}
