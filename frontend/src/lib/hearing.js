// Perkiraan dB SPL (tekanan suara di telinga) dan batas aman dengar.
//
//   dB SPL ≈ maxSpl + level (dBFS) + gain volume (dB)
//
// maxSpl = seberapa keras perangkat di volume 100% dengan sinyal 0 dBFS.
// Nilainya tidak bisa dibaca dari Windows, jadi dipakai default per jenis
// perangkat dan bisa dikalibrasi user.

import {clamp} from './levels'

export const SPL_SCALE = {min: 40, max: 110}

// WHO (Safe Listening): 80 dB(A) aman ±40 jam/minggu untuk dewasa.
export const SAFE_SPL = 80
// NIOSH: 85 dB(A) maksimal 8 jam/hari, waktu aman separuh tiap +3 dB.
export const LIMIT_SPL = 85

export const DEFAULT_MAX_SPL = {
  headphones: 100,
  headset: 100,
  speakers: 85,
  display: 85,
  other: 95,
}

export const KIND_LABEL = {
  headphones: 'Headphone',
  headset: 'Headset',
  speakers: 'Speaker',
  display: 'Monitor/TV (HDMI)',
  other: 'Perangkat lain',
}

export const PRESETS = [
  {id: 'tenang', label: 'Tenang', spl: 60, desc: 'Setara percakapan pelan. Cocok malam hari.'},
  {id: 'nyaman', label: 'Nyaman', spl: 70, desc: 'Direkomendasikan. Aman didengar berjam-jam.'},
  {id: 'keras', label: 'Keras', spl: 80, desc: 'Batas atas WHO untuk dengar rutin.'},
]

export const REFERENCES = [
  {spl: 30, label: 'Berbisik'},
  {spl: 60, label: 'Percakapan normal'},
  {spl: 70, label: 'Penyedot debu, TV ruang tamu'},
  {spl: 80, label: 'Lalu lintas padat'},
  {spl: 85, label: 'Batas 8 jam/hari'},
  {spl: 94, label: 'Motor dari dekat'},
  {spl: 100, label: 'Earphone volume maksimum, konser'},
  {spl: 110, label: 'Depan speaker konser'},
]

export function defaultMaxSpl(kind) {
  return DEFAULT_MAX_SPL[kind] ?? DEFAULT_MAX_SPL.other
}

/** @returns {'silent'|'safe'|'warn'|'danger'} */
export function zoneOfSpl(spl, silent = false) {
  if (silent) return 'silent'
  if (spl <= SAFE_SPL) return 'safe'
  if (spl <= LIMIT_SPL) return 'warn'
  return 'danger'
}

export const ZONE_STYLE = {
  silent: {label: 'Senyap', text: 'text-(--color-muted)', bar: 'bg-(--color-muted)', dot: 'bg-(--color-muted)'},
  safe: {label: 'Aman', text: 'text-success', bar: 'bg-success', dot: 'bg-success'},
  warn: {label: 'Waspada', text: 'text-warning', bar: 'bg-warning', dot: 'bg-warning'},
  danger: {label: 'Terlalu keras', text: 'text-danger', bar: 'bg-danger', dot: 'bg-danger'},
}

/** Waktu dengar aman per hari (jam), standar NIOSH. */
export function safeHoursPerDay(spl) {
  return 8 * 2 ** ((LIMIT_SPL - spl) / 3)
}

export function fmtSafeTime(spl) {
  const h = safeHoursPerDay(spl)
  if (spl <= 75) return 'tanpa batas praktis'
  if (h >= 24) return 'lebih dari 24 jam/hari'
  if (h >= 1) return `±${h >= 10 ? Math.round(h) : h.toFixed(1).replace('.', ',')} jam/hari`
  const m = h * 60
  return m >= 1 ? `±${Math.round(m)} menit/hari` : 'kurang dari 1 menit'
}

export function splToPercent(spl) {
  return clamp(((spl - SPL_SCALE.min) / (SPL_SCALE.max - SPL_SCALE.min)) * 100, 0, 100)
}

// --- paparan mingguan (WHO: 80 dB × 40 jam per 7 hari = 100%) ---

export const WEEKLY_REF = {spl: 80, hours: 40}

/** @returns {'silent'|'safe'|'warn'|'danger'} */
export function zoneOfDose(pct, listened = true) {
  if (!listened) return 'silent'
  if (pct < 80) return 'safe'
  if (pct <= 100) return 'warn'
  return 'danger'
}

export const DOSE_LABEL = {
  silent: 'Belum ada data',
  safe: 'Aman',
  warn: 'Mendekati batas',
  danger: 'Melewati batas',
}

/** Jam yang masih tersisa di level spl sebelum jatah mingguan habis. */
export function remainingHoursAt(spl, dosePct) {
  const left = Math.max(0, 1 - dosePct / 100)
  return left * WEEKLY_REF.hours * 10 ** ((WEEKLY_REF.spl - spl) / 10)
}

export function fmtMinutes(min) {
  if (min < 1) return min > 0 ? '<1 mnt' : '0 mnt'
  if (min < 60) return `${Math.round(min)} mnt`
  const h = Math.floor(min / 60)
  const m = Math.round(min % 60)
  return m ? `${h} j ${m} mnt` : `${h} jam`
}

// --- migrasi sekali jalan dari versi lama (pengaturan dulu di localStorage) ---

export function takeLegacySettings(deviceId) {
  const out = {}
  try {
    const t = Number(localStorage.getItem('limiterdb.targetSpl'))
    if (localStorage.getItem('limiterdb.targetSpl') != null && Number.isFinite(t)) out.targetSpl = t
    const key = `limiterdb.maxSpl.${deviceId}`
    const c = Number(localStorage.getItem(key))
    if (localStorage.getItem(key) != null && Number.isFinite(c)) out.maxSpl = c
    localStorage.removeItem('limiterdb.targetSpl')
    localStorage.removeItem(key)
  } catch {
    /* localStorage tidak tersedia → tidak ada yang dimigrasi */
  }
  return out
}
