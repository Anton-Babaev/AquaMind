package simulation

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/Anton-Babaev/AquaMind/internal/edge/domain"
)

// row — одна строка сценария CSV.
type row struct {
	offsetSec float64
	tempC     float64
	flowLPM   float64
	o2MgL     float64
	ph        float64
}

// Provider — реализация порта SensorSource на основе CSV-сценария.
// Читает сценарий с заданной скоростью (ускорение времени) и умеет повторять его.
// Из CSV берутся только значения и ритм; время ставится в момент чтения (now()).
type Provider struct {
	rows  []row
	speed float64 // ускорение времени: 1 сек реального = speed сек сценария
	loop  bool

	startedAt time.Time // момент первого чтения
	lastIdx   int       // индекс последней отданной строки
}

// NewProvider загружает CSV-сценарий.
// speed — ускорение времени (например, 60 = 1 минута сценария за 1 секунду).
// loop — повторять сценарий по кругу.
func NewProvider(path string, speed float64, loop bool) (*Provider, error) {
	if speed <= 0 {
		return nil, fmt.Errorf("speed must be > 0, got %v", speed)
	}
	rows, err := loadCSV(path)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("scenario %q is empty", path)
	}
	return &Provider{rows: rows, speed: speed, loop: loop, lastIdx: -1}, nil
}

// Read возвращает очередное показание согласно сценарию.
// Время показания ставится в момент чтения (now, UTC, миллисекунды).
func (p *Provider) Read(ctx context.Context) (domain.SensorReading, error) {
	if err := ctx.Err(); err != nil {
		return domain.SensorReading{}, err
	}

	now := time.Now().UTC().Truncate(time.Millisecond)
	if p.startedAt.IsZero() {
		p.startedAt = now
	}

	// Сколько «секунд сценария» прошло с начала.
	elapsedScenario := now.Sub(p.startedAt).Seconds() * p.speed

	// Находим строку, соответствующую прошедшему времени сценария.
	idx := p.indexFor(elapsedScenario)
	if idx < 0 {
		// Сценарий закончился и loop=false — возвращаем последнюю строку.
		idx = len(p.rows) - 1
	}
	p.lastIdx = idx

	r := p.rows[idx]
	return domain.SensorReading{
		TempC:   r.tempC,
		FlowLPM: r.flowLPM,
		O2MgL:   r.o2MgL,
		PH:      r.ph,
		At:      now,
	}, nil
}

// indexFor возвращает индекс строки для прошедшего времени сценария.
// При loop=true время зацикливается по длительности сценария.
// Возвращает -1, если сценарий закончился и loop=false.
func (p *Provider) indexFor(elapsedScenario float64) int {
	duration := p.rows[len(p.rows)-1].offsetSec
	if duration <= 0 {
		return 0
	}

	if elapsedScenario > duration {
		if !p.loop {
			return -1
		}
		elapsedScenario = mod(elapsedScenario, duration)
	}

	// Последняя строка, у которой offsetSec <= elapsedScenario.
	idx := 0
	for i, r := range p.rows {
		if r.offsetSec <= elapsedScenario {
			idx = i
		} else {
			break
		}
	}
	return idx
}

func mod(a, m float64) float64 {
	r := a - m*float64(int(a/m))
	if r < 0 {
		r += m
	}
	return r
}

// loadCSV читает и парсит CSV-сценарий.
func loadCSV(path string) ([]row, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open scenario: %w", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read scenario: %w", err)
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("scenario %q has no data rows", path)
	}

	// Первая строка — заголовок, пропускаем.
	var rows []row
	for i, rec := range records[1:] {
		if len(rec) != 5 {
			return nil, fmt.Errorf("row %d: expected 5 columns, got %d", i+2, len(rec))
		}
		vals := make([]float64, 5)
		for j, s := range rec {
			v, err := strconv.ParseFloat(s, 64)
			if err != nil {
				return nil, fmt.Errorf("row %d col %d: %w", i+2, j+1, err)
			}
			vals[j] = v
		}
		rows = append(rows, row{
			offsetSec: vals[0],
			tempC:     vals[1],
			flowLPM:   vals[2],
			o2MgL:     vals[3],
			ph:        vals[4],
		})
	}
	return rows, nil
}
