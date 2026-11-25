package domain

import "time"

// SensorReading — одно показание датчиков, снятое в момент чтения.
// Время ставится в момент чтения (now()), а не берётся из CSV.
type SensorReading struct {
	TempC   float64   // температура, °C
	FlowLPM float64   // поток, л/мин
	O2MgL   float64   // растворённый кислород, мг/л
	PH      float64   // кислотность (хранится и показывается, правил в MVP нет)
	At      time.Time // время чтения, UTC, точность до миллисекунд
}
