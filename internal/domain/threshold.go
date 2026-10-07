// threshold.go
package domain

import "errors"

var ErrInvalidThreshold = errors.New("threshold must be between -60 and 0 dBFS")

// Threshold adalah batas level efektif dalam dBFS.
type Threshold float64

func NewThreshold(db float64) (Threshold, error) {
	if !(db >= -60 && db <= 0) { // juga menolak NaN
		return 0, ErrInvalidThreshold
	}
	return Threshold(db), nil
}