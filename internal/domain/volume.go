// volume.go
package domain

import "math"

type Volume float64

func NewVolume(v float64) Volume { 
	return Volume(math.Max(0, math.Min(1, v)))
}

func VolumeFromDB(db float64) Volume { return NewVolume(math.Pow(10, db/20)) }
