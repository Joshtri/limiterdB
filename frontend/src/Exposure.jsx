import {useEffect, useState} from 'react'
import {Exposure as FetchExposure} from '../wailsjs/go/main/App'
import {cn} from '@/lib/cn'
import {useVisible} from '@/lib/useVisible'
import {DOSE_LABEL, fmtMinutes, remainingHoursAt, WEEKLY_REF, ZONE_STYLE, zoneOfDose, zoneOfSpl} from '@/lib/hearing'

const POLL_MS = 5000
const CHART_H = 132
const DAILY_SHARE = 100 / 7 // jatah rata-rata per hari supaya seminggu ≤ 100%

const fmtPct = (p) => (p > 0 && p < 1 ? '<1%' : `${Math.round(p)}%`)

function dayLabel(date, isToday, style = 'short') {
  if (isToday) return 'Hari ini'
  const d = new Date(`${date}T12:00:00`)
  return d.toLocaleDateString('id-ID', style === 'short' ? {weekday: 'short'} : {weekday: 'long', day: 'numeric', month: 'short'})
}

function niceMax(v) {
  const steps = [5, 10, 20, 25, 50, 100, 150, 200, 300, 500]
  return steps.find((s) => s >= v) ?? Math.ceil(v / 100) * 100
}

function useExposure() {
  const visible = useVisible()
  const [data, setData] = useState(null)
  const [error, setError] = useState('')
  useEffect(() => {
    if (!visible) return
    const load = () =>
      FetchExposure()
        .then((d) => {
          setData(d)
          setError('')
        })
        .catch((e) => setError(String(e)))
    load()
    const id = setInterval(load, POLL_MS)
    return () => clearInterval(id)
  }, [visible])
  return {data, error}
}

function WeeklyHero({week, targetSpl}) {
  const listened = week.listenMin > 0
  const zone = zoneOfDose(week.dosePct, listened)
  const style = ZONE_STYLE[zone]
  const remaining = remainingHoursAt(targetSpl, week.dosePct)

  return (
    <div className="flex flex-col gap-3 rounded-md border border-(--color-border) bg-(--color-surface-2) p-4">
      <div className="flex items-start justify-between gap-3">
        <div className="flex flex-col">
          <span className="text-xs font-medium text-(--color-secondary)">Paparan 7 hari terakhir</span>
          <span className="text-5xl font-semibold leading-tight text-(--color-fg)">{fmtPct(week.dosePct)}</span>
          <span className="text-xs text-(--color-muted)">dari batas aman mingguan WHO</span>
        </div>
        <span className="flex items-center gap-1.5 rounded-full border border-(--color-border) px-2.5 py-1 text-xs font-semibold text-(--color-fg)">
          <span className={cn('size-2 rounded-full', style.dot)} />
          {DOSE_LABEL[zone]}
        </span>
      </div>

      <div
        className="relative h-2.5 w-full overflow-hidden rounded-full bg-(--color-surface-3)"
        role="progressbar"
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={Math.round(week.dosePct)}
        aria-label="Paparan mingguan"
      >
        <div
          className={cn('absolute inset-y-0 left-0 rounded-full transition-[width] duration-300', style.bar)}
          style={{width: `${Math.min(100, week.dosePct)}%`}}
        />
      </div>

      <div className="text-xs text-(--color-secondary)">
        {listened ? (
          <>
            Rata-rata <span className="text-(--color-fg)">~{Math.round(week.leqDb)} dB</span> selama{' '}
            <span className="text-(--color-fg)">{fmtMinutes(week.listenMin)}</span>.{' '}
            {week.dosePct < 100 ? (
              <>
                Sisa jatah minggu ini: ±
                <span className="text-(--color-fg)">{remaining >= 100 ? '100+' : Math.round(remaining)} jam</span> lagi di{' '}
                {targetSpl} dB (batas dengar kamu).
              </>
            ) : (
              <>Jatah minggu ini sudah habis — sebaiknya kurangi volume atau durasi beberapa hari ke depan.</>
            )}
          </>
        ) : (
          'Belum ada audio yang tercatat. limiterdB mencatat paparan selama berjalan, termasuk saat di tray.'
        )}
      </div>
    </div>
  )
}

function StatTile({label, value, sub}) {
  return (
    <div className="flex flex-col gap-0.5 rounded-md border border-(--color-border) bg-(--color-surface-2) p-3">
      <span className="text-[11px] text-(--color-muted)">{label}</span>
      <span className="text-lg font-semibold text-(--color-fg)">{value}</span>
      {sub && <span className="text-[11px] text-(--color-secondary)">{sub}</span>}
    </div>
  )
}

function DailyChart({days}) {
  const [hover, setHover] = useState(null)
  const max = niceMax(Math.max(DAILY_SHARE * 1.2, ...days.map((d) => d.dosePct)))
  const y = (pct) => (pct / max) * CHART_H
  const ticks = [0, max / 2, max]
  const todayIdx = days.length - 1

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-baseline justify-between">
        <span className="text-xs font-medium text-(--color-secondary)">Per hari · % dari jatah mingguan</span>
        <span className="flex items-center gap-1.5 text-[11px] text-(--color-muted)">
          <span className="inline-block w-4 border-t border-dashed border-(--color-secondary)" />
          jatah rata-rata/hari ({Math.round(DAILY_SHARE)}%)
        </span>
      </div>

      <div className="relative flex gap-2 pt-4">
        {/* sumbu Y */}
        <div className="relative w-8 shrink-0 text-right text-[10px] tabular-nums text-(--color-muted)" style={{height: CHART_H}}>
          {ticks.map((t) => (
            <span key={t} className="absolute right-0 translate-y-1/2" style={{bottom: y(t)}}>
              {Math.round(t)}%
            </span>
          ))}
        </div>

        <div className="relative flex-1" style={{height: CHART_H}}>
          {/* grid */}
          {ticks.map((t) => (
            <div key={t} className="absolute inset-x-0 border-t border-(--color-border)" style={{bottom: y(t)}} />
          ))}
          {/* referensi jatah harian */}
          <div className="absolute inset-x-0 border-t border-dashed border-(--color-secondary)" style={{bottom: y(DAILY_SHARE)}} />

          <div className="absolute inset-0 flex">
            {days.map((d, i) => {
              const h = d.dosePct > 0 ? Math.max(2, y(d.dosePct)) : 0
              return (
                <div
                  key={d.date}
                  className="relative flex flex-1 cursor-default items-end justify-center"
                  onMouseEnter={() => setHover(i)}
                  onMouseLeave={() => setHover(null)}
                >
                  {hover === i && <div className="absolute inset-y-0 inset-x-1 rounded-sm bg-(--color-surface-3)/60" />}
                  <div
                    className={cn('relative w-full max-w-6 rounded-t bg-brand-500 transition-[height] duration-300', i !== todayIdx && 'opacity-70')}
                    style={{height: h}}
                  />
                  {i === todayIdx && d.dosePct > 0 && (
                    <span
                      className="absolute text-[10px] font-semibold tabular-nums text-(--color-fg)"
                      style={{bottom: h + 4}}
                    >
                      {fmtPct(d.dosePct)}
                    </span>
                  )}
                </div>
              )
            })}
          </div>

          {hover != null && (
            <Tooltip day={days[hover]} isToday={hover === todayIdx} index={hover} count={days.length} />
          )}
        </div>
      </div>

      {/* sumbu X */}
      <div className="flex gap-2">
        <div className="w-8 shrink-0" />
        <div className="flex flex-1">
          {days.map((d, i) => (
            <span
              key={d.date}
              className={cn('flex-1 text-center text-[10px]', i === todayIdx ? 'font-semibold text-(--color-fg)' : 'text-(--color-muted)')}
            >
              {dayLabel(d.date, i === todayIdx)}
            </span>
          ))}
        </div>
      </div>
    </div>
  )
}

function Tooltip({day, isToday, index, count}) {
  const zone = zoneOfSpl(day.leqDb, day.listenMin <= 0)
  // Tooltip menempel ke kolom, digeser ke dalam supaya tidak keluar area chart.
  const leftPct = ((index + 0.5) / count) * 100
  const shift = index === 0 ? '0%' : index === count - 1 ? '-100%' : '-50%'
  return (
    <div
      className="pointer-events-none absolute top-0 z-10 w-44 rounded-md border border-(--color-border-strong) bg-(--color-surface) p-2.5 text-xs shadow-lg"
      style={{left: `${leftPct}%`, transform: `translateX(${shift})`}}
    >
      <div className="mb-1 font-semibold capitalize text-(--color-fg)">{dayLabel(day.date, isToday, 'long')}</div>
      {day.listenMin > 0 ? (
        <div className="flex flex-col gap-0.5 text-(--color-secondary)">
          <Row label="Durasi" value={fmtMinutes(day.listenMin)} />
          <Row
            label="Rata-rata"
            value={
              <span className="flex items-center gap-1.5">
                <span className={cn('size-2 rounded-full', ZONE_STYLE[zone].dot)} />~{Math.round(day.leqDb)} dB
              </span>
            }
          />
          <Row label="Jatah mingguan" value={fmtPct(day.dosePct)} />
        </div>
      ) : (
        <div className="text-(--color-muted)">Tidak ada audio tercatat.</div>
      )}
    </div>
  )
}

function Row({label, value}) {
  return (
    <div className="flex justify-between gap-2">
      <span>{label}</span>
      <span className="tabular-nums text-(--color-fg)">{value}</span>
    </div>
  )
}

function DayTable({days}) {
  return (
    <details className="group rounded-md border border-(--color-border) bg-(--color-surface-2)">
      <summary className="flex cursor-pointer list-none items-center justify-between px-3 py-2.5 text-xs font-medium text-(--color-secondary) select-none">
        Lihat sebagai tabel
        <span className="text-(--color-muted) transition-transform group-open:rotate-90">›</span>
      </summary>
      <table className="w-full border-t border-(--color-border) text-xs">
        <thead className="text-(--color-muted)">
          <tr>
            <th className="px-3 py-2 text-left font-medium">Hari</th>
            <th className="px-3 py-2 text-right font-medium">Durasi</th>
            <th className="px-3 py-2 text-right font-medium">Rata-rata</th>
            <th className="px-3 py-2 text-right font-medium">Jatah</th>
          </tr>
        </thead>
        <tbody className="tabular-nums text-(--color-secondary)">
          {[...days].reverse().map((d, i) => (
            <tr key={d.date} className="border-t border-(--color-border)">
              <td className="px-3 py-1.5 capitalize text-(--color-fg)">{dayLabel(d.date, i === 0, 'long')}</td>
              <td className="px-3 py-1.5 text-right">{d.listenMin > 0 ? fmtMinutes(d.listenMin) : '—'}</td>
              <td className="px-3 py-1.5 text-right">{d.listenMin > 0 ? `~${Math.round(d.leqDb)} dB` : '—'}</td>
              <td className="px-3 py-1.5 text-right">{d.listenMin > 0 ? fmtPct(d.dosePct) : '—'}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </details>
  )
}

export function ExposurePanel({targetSpl}) {
  const {data, error} = useExposure()

  if (!data) {
    return <div className="py-10 text-center text-xs text-(--color-muted)">{error || 'Memuat catatan paparan…'}</div>
  }

  const {today, week, days} = data
  const todayZone = zoneOfSpl(today.leqDb, today.listenMin <= 0)

  return (
    <div className="flex flex-col gap-5">
      <WeeklyHero week={week} targetSpl={targetSpl} />

      <div className="grid grid-cols-3 gap-2">
        <StatTile label="Hari ini didengar" value={fmtMinutes(today.listenMin)} />
        <StatTile
          label="Rata-rata hari ini"
          value={today.listenMin > 0 ? `~${Math.round(today.leqDb)} dB` : '—'}
          sub={today.listenMin > 0 ? ZONE_STYLE[todayZone].label : null}
        />
        <StatTile label="Jatah terpakai hari ini" value={fmtPct(today.dosePct)} sub={`patokan ≤ ${Math.round(DAILY_SHARE)}%/hari`} />
      </div>

      <DailyChart days={days} />
      <DayTable days={days} />

      <p className="text-[11px] leading-relaxed text-(--color-muted)">
        Dihitung seperti fitur Headphone Audio Level di iPhone: 100% = rata-rata {WEEKLY_REF.spl} dB selama{' '}
        {WEEKLY_REF.hours} jam dalam 7 hari (WHO). Tiap +3 dB jatahnya habis 2× lebih cepat. Hanya waktu saat ada audio
        terdengar yang dihitung. Angkanya perkiraan — kalibrasi perangkat di tab Pengaturan supaya lebih akurat.
      </p>
    </div>
  )
}
