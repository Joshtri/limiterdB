package domain

import (
	"math"
	"sort"
	"time"
)

// Paparan suara dihitung seperti fitur Headphone Audio Level di iPhone,
// memakai patokan WHO: rata-rata 80 dB selama 40 jam per 7 hari = 100%.
// Aturan 3 dB (equal energy): tiap +3 dB jatah waktunya separuh.
const (
	WeeklyRefSPL   = 80.0
	WeeklyRefHours = 40.0
	dateLayout     = "2006-01-02"
)

// weeklyAllowance = energi total yang setara 80 dB × 40 jam.
var weeklyAllowance = WeeklyRefHours * 3600 * math.Pow(10, WeeklyRefSPL/10)

// DayExposure adalah akumulasi paparan satu hari (tanggal lokal).
type DayExposure struct {
	ListenSec float64 `json:"listenSec"` // detik ada audio terdengar
	Energy    float64 `json:"energy"`    // Σ dt·10^(dB/10)
}

// Leq: rata-rata level energi selama mendengar (dB SPL).
func (d DayExposure) Leq() float64 {
	if d.ListenSec <= 0 {
		return 0
	}
	return 10 * math.Log10(d.Energy/d.ListenSec)
}

// DosePct: persentase dari jatah aman mingguan WHO.
func (d DayExposure) DosePct() float64 { return d.Energy / weeklyAllowance * 100 }

// ExposureLog menyimpan paparan per hari. Key = tanggal "2006-01-02".
type ExposureLog struct {
	Days map[string]DayExposure `json:"days"`
}

func NewExposureLog() ExposureLog { return ExposureLog{Days: map[string]DayExposure{}} }

// Add mencatat dt detik mendengar di level spl pada waktu t.
func (l *ExposureLog) Add(t time.Time, dt time.Duration, spl float64) {
	if dt <= 0 || !isFinite(spl) {
		return
	}
	if l.Days == nil {
		l.Days = map[string]DayExposure{}
	}
	key := t.Format(dateLayout)
	d := l.Days[key]
	sec := dt.Seconds()
	d.ListenSec += sec
	d.Energy += sec * math.Pow(10, spl/10)
	l.Days[key] = d
}

// Prune membuang hari yang lebih lama dari keepDays.
func (l *ExposureLog) Prune(now time.Time, keepDays int) {
	cutoff := now.AddDate(0, 0, -keepDays).Format(dateLayout)
	for k := range l.Days {
		if k < cutoff {
			delete(l.Days, k)
		}
	}
}

// Clone membuat salinan (aman disimpan di goroutine lain).
func (l ExposureLog) Clone() ExposureLog {
	c := NewExposureLog()
	for k, v := range l.Days {
		c.Days[k] = v
	}
	return c
}

// DayStat adalah ringkasan satu hari untuk UI.
type DayStat struct {
	Date      string  `json:"date"`
	ListenMin float64 `json:"listenMin"`
	LeqDB     float64 `json:"leqDb"`
	DosePct   float64 `json:"dosePct"`
}

// ExposureSummary adalah ringkasan paparan hari ini dan 7 hari terakhir.
type ExposureSummary struct {
	Today DayStat   `json:"today"`
	Week  DayStat   `json:"week"` // Date = tanggal awal periode
	Days  []DayStat `json:"days"` // 7 hari, terlama → hari ini
}

func statOf(date string, d DayExposure) DayStat {
	return DayStat{Date: date, ListenMin: d.ListenSec / 60, LeqDB: d.Leq(), DosePct: d.DosePct()}
}

// Summary menghitung ringkasan 7 hari terakhir (termasuk hari ini).
func (l ExposureLog) Summary(now time.Time) ExposureSummary {
	var s ExposureSummary
	var week DayExposure
	for i := 6; i >= 0; i-- {
		date := now.AddDate(0, 0, -i).Format(dateLayout)
		d := l.Days[date]
		week.ListenSec += d.ListenSec
		week.Energy += d.Energy
		s.Days = append(s.Days, statOf(date, d))
	}
	s.Today = s.Days[len(s.Days)-1]
	s.Week = statOf(s.Days[0].Date, week)
	return s
}

// Dates mengembalikan tanggal yang tercatat, terurut (dipakai test/debug).
func (l ExposureLog) Dates() []string {
	out := make([]string, 0, len(l.Days))
	for k := range l.Days {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
