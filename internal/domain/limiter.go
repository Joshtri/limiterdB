package domain

import "errors"

type Limiter struct {
	attack      float64 // 0<attack<=1; porsi jarak yang ditutup per tick saat menurun (1 = langsung)
	releaseStep float64 // kenaikan volume per tick, misal 0.001 tiap 50ms = 2%/detik
	minVolume   Volume  // lantai, supaya tidak pernah mute total (get the Volume type from volume.go)
}

func NewLimiter(attack, releaseStep float64, minVolume Volume) (*Limiter, error) {

	if attack <= 0 || attack > 1 || releaseStep <= 0 {
		return nil, errors.New("invalid limiter params")
	}
	return &Limiter{attack, releaseStep, minVolume}, nil
}

// NextVolume menghitung volume berikutnya. ceiling = volume maksimum
// yang diinginkan user (batas atas saat lagu pelan).
func (l *Limiter) NextVolume(cur Volume, lvl Level, th Threshold, ceiling Volume) Volume {
	target := VolumeFromDB(float64(th) - float64(lvl))
	if target > ceiling {
		target = ceiling
	}
	if target < l.minVolume {
		target = l.minVolume
	}

	switch {
	case target < cur: // terlalu keras → turun cepat
		return cur - Volume(float64(cur-target)*l.attack)
	case target > cur: // aman → naik pelan, tanpa melewati target
		next := cur + Volume(l.releaseStep)
		if next > target {
			next = target
		}
		return next
	}
	return cur
}
