package domain

// Device menjelaskan output device default yang sedang dipakai.
type Device struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Kind  string  `json:"kind"` // headphones | headset | speakers | display | other
	Muted bool    `json:"muted"`
	MinDB float64 `json:"minDb"` // rentang volume endpoint (dB)
	MaxDB float64 `json:"maxDb"`
}
