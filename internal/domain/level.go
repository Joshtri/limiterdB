// level.go
package domain

import "math"

const silenceFloor = -100.0

// Level adalah kekerasan sinyal dalam dBFS (0 = maksimum digital).
type Level float64

func LevelFromAmplitude(a float64) Level {
	if a <= 0 {
		return Level(silenceFloor)
	}
	db := 20 * math.Log10(a)
	return Level(math.Max(silenceFloor, math.Min(0, db)))
}
