// Helper dBFS (0 = maksimum digital) dan parameter limiter.

export const SILENCE_DB = -59

// Parameter limiter, cerminan dari app.go / tickInterval — hanya dipakai
// buat teks penjelasan di UI.
export const LIMITER_PARAMS = {
  tickMs: 50,
  attack: 0.5,
  releaseStep: 0.001,
  minVolume: 0.01,
}

export const THRESHOLD_MIN_DB = -60
export const THRESHOLD_MAX_DB = 0

export function volumeToDb(v) {
  return v > 0 ? 20 * Math.log10(v) : -100
}

export function clamp(x, lo, hi) {
  return Math.max(lo, Math.min(hi, x))
}

export function fmtDb(db, digits = 1) {
  if (db <= -99) return '−∞'
  const s = Math.abs(db).toFixed(digits)
  return db < 0 && Number(s) !== 0 ? `−${s}` : s
}

// Rata-rata energi (bukan rata-rata angka dB) dengan time constant tauMs,
// dipanggil tiap dtMs. Hasilnya mirip level rata-rata yang dirasakan telinga.
export function smoothDb(prevDb, nextDb, dtMs, tauMs) {
  if (prevDb == null) return nextDb
  const alpha = 1 - Math.exp(-dtMs / tauMs)
  const e = (1 - alpha) * 10 ** (prevDb / 10) + alpha * 10 ** (nextDb / 10)
  return 10 * Math.log10(e)
}
