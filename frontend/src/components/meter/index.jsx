import {useEffect, useState} from 'react'
import {cn} from '@/lib/cn'
import {fmtDb} from '@/lib/levels'
import {LIMIT_SPL, SAFE_SPL, SPL_SCALE, splToPercent, ZONE_STYLE, zoneOfSpl} from '@/lib/hearing'

const TICKS = [40, 50, 60, 70, 80, 90, 100, 110]
const PEAK_HOLD_MS = 1500

// Simpan nilai tertinggi selama PEAK_HOLD_MS supaya lonjakan singkat tetap
// kelihatan walau bar sudah turun.
function usePeakHold(value) {
  const [hold, setHold] = useState({value, at: 0})
  useEffect(() => {
    const now = performance.now()
    setHold((h) => (value >= h.value || now - h.at > PEAK_HOLD_MS ? {value, at: now} : h))
  }, [value])
  return hold.value
}

/**
 * Meter horizontal dB SPL. Latar belakang menampilkan zona aman (hijau),
 * waspada (kuning) dan terlalu keras (merah) — batas absolut, sama untuk
 * semua lagu. Garis putih = batas dengar yang dipilih.
 */
export function SplMeter({label, hint, spl, silent, targetSpl, showScale = false}) {
  const zone = zoneOfSpl(spl, silent)
  const style = ZONE_STYLE[zone]
  const peak = usePeakHold(silent ? SPL_SCALE.min : spl)

  const safePct = splToPercent(SAFE_SPL)
  const limitPct = splToPercent(LIMIT_SPL)

  return (
    <div className="flex flex-col gap-1.5">
      <div className="flex items-baseline justify-between gap-2">
        <span className="text-xs font-medium text-(--color-secondary)">
          {label}
          {hint && <span className="ml-1.5 font-normal text-(--color-muted)">{hint}</span>}
        </span>
        <span className="flex items-baseline gap-2 tabular-nums">
          <span className={cn('text-xs font-semibold', style.text)}>{style.label}</span>
          <span className="text-sm text-(--color-fg)">{silent ? '—' : `~${Math.round(spl)} dB`}</span>
        </span>
      </div>

      <div className="relative h-4 w-full overflow-hidden rounded-sm bg-(--color-surface-3)">
        <div className="absolute inset-y-0 left-0 bg-success/15" style={{width: `${safePct}%`}} />
        <div className="absolute inset-y-0 bg-warning/20" style={{left: `${safePct}%`, width: `${limitPct - safePct}%`}} />
        <div className="absolute inset-y-0 right-0 bg-danger/20" style={{left: `${limitPct}%`}} />

        {!silent && (
          <div
            className={cn('absolute inset-y-0.5 left-0 rounded-r-xs transition-[width] duration-100 ease-out', style.bar)}
            style={{width: `${splToPercent(spl)}%`}}
          />
        )}

        {peak > SPL_SCALE.min && (
          <div
            className={cn('absolute inset-y-0 w-0.5', ZONE_STYLE[zoneOfSpl(peak)].bar)}
            style={{left: `calc(${splToPercent(peak)}% - 1px)`}}
          />
        )}

        {targetSpl != null && (
          <div
            className="absolute inset-y-0 w-0.5 bg-(--color-fg)"
            style={{left: `calc(${splToPercent(targetSpl)}% - 1px)`}}
            title="Batas dengar"
          />
        )}
      </div>

      {showScale && (
        <div className="relative h-3.5 text-[10px] text-(--color-muted) tabular-nums">
          {TICKS.map((t) => (
            <span
              key={t}
              className="absolute -translate-x-1/2 first:translate-x-0 last:-translate-x-full"
              style={{left: `${splToPercent(t)}%`}}
            >
              {t}
            </span>
          ))}
        </div>
      )}
    </div>
  )
}

/** Bar gain volume Windows (dalam dB) dengan penanda ceiling. */
export function VolumeBar({volume, ceiling, minDb = -40}) {
  const toPct = (v) => (v > 0 ? Math.max(0, Math.min(100, (1 - (20 * Math.log10(v)) / minDb) * 100)) : 0)
  const db = volume > 0 ? 20 * Math.log10(volume) : -100
  return (
    <div className="flex flex-col gap-1.5">
      <div className="flex items-baseline justify-between gap-2">
        <span className="text-xs font-medium text-(--color-secondary)">
          Volume Windows
          <span className="ml-1.5 font-normal text-(--color-muted)">diatur limiter</span>
        </span>
        <span className="text-sm tabular-nums text-(--color-fg)">{fmtDb(db)} dB</span>
      </div>
      <div className="relative h-2.5 w-full overflow-hidden rounded-sm bg-(--color-surface-3)">
        <div
          className="absolute inset-y-0 left-0 bg-brand-500 transition-[width] duration-100 ease-out"
          style={{width: `${toPct(volume)}%`}}
        />
        <div
          className="absolute inset-y-0 w-0.5 bg-(--color-fg)"
          style={{left: `calc(${toPct(ceiling)}% - 1px)`}}
          title="Volume maksimum"
        />
      </div>
    </div>
  )
}
