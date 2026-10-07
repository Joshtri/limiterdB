import {useEffect, useRef, useState} from 'react'
import {
  ResetCalibration,
  SetAutoStart,
  SetCalibration,
  SetCeiling,
  SetLimiterEnabled,
  SetTargetSPL,
  Settings,
  Status,
} from '../wailsjs/go/main/App'
import {Switch} from '@/components/switch'
import {Slider} from '@/components/slider'
import {Field} from '@/components/field'
import {Button} from '@/components/button'
import {Tabs} from '@/components/tabs'
import {SplMeter, VolumeBar} from '@/components/meter'
import {About} from './About'
import {ExposurePanel} from './Exposure'
import {cn} from '@/lib/cn'
import {useVisible} from '@/lib/useVisible'
import {fmtDb, smoothDb, volumeToDb} from '@/lib/levels'
import {fmtSafeTime, KIND_LABEL, PRESETS, SAFE_SPL, takeLegacySettings, ZONE_STYLE, zoneOfSpl} from '@/lib/hearing'

const POLL_MS = 100
// Rata-rata ±2 detik untuk angka utama & waktu aman, supaya tidak loncat-loncat.
const AVG_TAU_MS = 2000
// Selisih kecil di atas batas dengar masih dianggap "di batas".
const TARGET_TOLERANCE_DB = 1.5

function useStatus() {
  const visible = useVisible()
  const [snap, setSnap] = useState({status: null, avgSrc: null, avgOut: null, error: ''})
  useEffect(() => {
    if (!visible) return
    const id = setInterval(() => {
      Status()
        .then((s) =>
          setSnap((prev) => {
            const src = s.listening ? s.lastLevel : -100
            const out = s.listening ? s.lastLevel + volumeToDb(s.currentVolume) : -100
            return {
              status: s,
              avgSrc: smoothDb(prev.avgSrc, src, POLL_MS, AVG_TAU_MS),
              avgOut: smoothDb(prev.avgOut, out, POLL_MS, AVG_TAU_MS),
              error: '',
            }
          })
        )
        .catch((e) => setSnap((prev) => ({...prev, error: String(e)})))
    }, POLL_MS)
    return () => clearInterval(id)
  }, [visible])
  return snap
}

function statusMessage({status, silent, srcSpl, heardSpl}) {
  if (!status?.deviceOk) return 'Tidak ada perangkat output aktif. Colok atau pilih perangkat di Windows.'
  if (status.device.muted) return 'Perangkat sedang di-mute di Windows.'
  if (silent) return 'Tidak ada audio yang sedang diputar.'
  const target = Math.round(status.maxSpl + status.threshold)
  const over = heardSpl > target + TARGET_TOLERANCE_DB
  if (!status.enabled) {
    return over
      ? `Limiter mati. Yang kamu dengar di atas batas ${target} dB — nyalakan limiter untuk meredamnya.`
      : `Limiter mati. Masih di bawah batas ${target} dB.`
  }
  if (over) return 'Lonjakan terdeteksi — limiter sedang menurunkan volume…'
  if (srcSpl > target + TARGET_TOLERANCE_DB) {
    return `Lagu ini keras (~${Math.round(srcSpl)} dB kalau tidak dibatasi). Limiter menahannya di ~${Math.round(heardSpl)} dB.`
  }
  return 'Lagu cukup pelan, volume tidak perlu diturunkan.'
}

function StatusBanner({zone, spl, showSpl, message}) {
  const style = ZONE_STYLE[zone]
  const tint = {
    silent: 'border-(--color-border) bg-(--color-surface-2)',
    safe: 'border-success/40 bg-success/10',
    warn: 'border-warning/40 bg-warning/10',
    danger: 'border-danger/40 bg-danger/10',
  }[zone]

  return (
    <div className={cn('flex items-center gap-4 rounded-md border p-4 transition-colors', tint)}>
      <div className="flex min-w-20 flex-col items-center">
        <span className={cn('text-2xl font-semibold tabular-nums', style.text)}>{showSpl ? `~${Math.round(spl)}` : '—'}</span>
        <span className="text-[10px] text-(--color-muted)">dB SPL</span>
      </div>
      <div className="flex flex-1 flex-col gap-0.5">
        <span className={cn('flex items-center gap-2 text-sm font-semibold', style.text)}>
          <span className={cn('size-2.5 rounded-full', style.dot, zone !== 'silent' && 'animate-pulse')} />
          {style.label}
          {showSpl && <span className="font-normal text-(--color-secondary)">· aman didengar {fmtSafeTime(spl)}</span>}
        </span>
        <span className="text-xs text-(--color-secondary)">{message}</span>
      </div>
    </div>
  )
}

function DeviceBar({status}) {
  if (!status?.deviceOk) {
    return (
      <div className="rounded-md border border-danger/40 bg-danger/10 px-3 py-2 text-xs text-danger">
        Tidak ada perangkat output.
      </div>
    )
  }
  const d = status.device
  return (
    <div className="flex items-center justify-between gap-3 rounded-md border border-(--color-border) bg-(--color-surface-2) px-3 py-2 text-xs">
      <div className="flex min-w-0 items-center gap-2">
        <span className="size-2 shrink-0 rounded-full bg-success" title="Terhubung" />
        <span className="truncate font-medium text-(--color-fg)">{d.name || 'Perangkat tanpa nama'}</span>
        <span className="shrink-0 text-(--color-muted)">· {KIND_LABEL[d.kind] ?? d.kind}</span>
        {d.muted && <span className="shrink-0 font-semibold text-warning">· Mute</span>}
      </div>
      <span className="shrink-0 text-(--color-muted)">
        {status.calibrated ? 'dikalibrasi' : 'perkiraan default'} · ikut Windows otomatis
      </span>
    </div>
  )
}

function PresetPicker({targetSpl, onChange}) {
  return (
    <div className="grid grid-cols-3 gap-2">
      {PRESETS.map((p) => {
        const active = Math.round(targetSpl) === p.spl
        return (
          <button
            key={p.id}
            type="button"
            onClick={() => onChange(p.spl)}
            className={cn(
              'flex cursor-pointer flex-col gap-0.5 rounded-md border p-2.5 text-left transition-colors',
              active
                ? 'border-brand-500 bg-brand-500/15'
                : 'border-(--color-border) bg-(--color-surface-2) hover:border-(--color-border-strong)'
            )}
          >
            <span className="flex items-baseline justify-between">
              <span className="text-xs font-semibold text-(--color-fg)">{p.label}</span>
              <span className="text-xs tabular-nums text-(--color-secondary)">{p.spl} dB</span>
            </span>
            <span className="text-[11px] leading-snug text-(--color-muted)">{p.desc}</span>
          </button>
        )
      })}
    </div>
  )
}

function LimiterPanel({status, avgSrc, avgOut, targetSpl, onTargetChange}) {
  const levelDb = status?.lastLevel ?? -100
  const volume = status?.currentVolume ?? 0
  const maxSpl = status?.maxSpl ?? 95
  const ceiling = status?.ceiling ?? 1
  const enabled = status?.enabled ?? false
  const listening = status?.listening ?? false
  const reachableSpl = maxSpl + (status?.threshold ?? 0)

  // dB SPL ≈ kerasnya perangkat + level lagu (dBFS) + gain volume (dB)
  const heardSpl = maxSpl + levelDb + volumeToDb(volume)
  const srcSpl = maxSpl + levelDb + volumeToDb(enabled ? ceiling : volume)
  const avgHeardSpl = maxSpl + (avgOut ?? -100)
  const avgSrcSpl = maxSpl + (avgSrc ?? -100) + volumeToDb(enabled ? ceiling : volume)
  const silentAvg = !listening || avgOut == null
  const zone = zoneOfSpl(avgHeardSpl, silentAvg)

  return (
    <div className="flex flex-col gap-5">
      <DeviceBar status={status} />

      <StatusBanner
        zone={zone}
        spl={avgHeardSpl}
        showSpl={!silentAvg}
        message={statusMessage({status, silent: silentAvg, srcSpl: avgSrcSpl, heardSpl: avgHeardSpl})}
      />

      <div className="flex flex-col gap-4 rounded-md border border-(--color-border) bg-(--color-surface-2) p-4">
        <SplMeter label="Tanpa limiter" hint="kerasnya lagu ini" spl={srcSpl} silent={!listening} targetSpl={reachableSpl} />
        <SplMeter
          label="Yang kamu dengar"
          hint="setelah limiter"
          spl={heardSpl}
          silent={!listening}
          targetSpl={reachableSpl}
          showScale
        />
        <VolumeBar volume={volume} ceiling={ceiling} />

        <div className="flex flex-wrap gap-x-4 gap-y-1 text-[11px] text-(--color-muted)">
          <Legend swatch="size-2.5 bg-success" label={`Aman ≤ ${SAFE_SPL} dB`} />
          <Legend swatch="size-2.5 bg-warning" label="Waspada 80–85 dB" />
          <Legend swatch="size-2.5 bg-danger" label="Terlalu keras > 85 dB" />
          <Legend swatch="h-2.5 w-0.5 bg-(--color-fg)" label="Batas dengar kamu" />
        </div>
      </div>

      <div className="flex flex-col gap-3">
        <div className="flex items-baseline justify-between">
          <span className="text-sm font-semibold">Batas dengar</span>
          <span className="text-xs text-(--color-muted)">yang boleh sampai ke telinga</span>
        </div>
        <PresetPicker targetSpl={targetSpl} onChange={onTargetChange} />
        <Field label="Atur sendiri" hint={`${targetSpl} dB SPL · aman ${fmtSafeTime(targetSpl)}`}>
          <Slider value={targetSpl} onValueChange={onTargetChange} min={50} max={90} step={1} />
        </Field>
        {targetSpl > SAFE_SPL && (
          <span className="text-xs text-warning">
            Di atas {SAFE_SPL} dB: aman hanya {fmtSafeTime(targetSpl)}. Pendengaran bisa rusak permanen kalau sering.
          </span>
        )}
        {status && Math.round(reachableSpl) !== targetSpl && (
          <span className="text-xs text-(--color-muted)">
            {reachableSpl < targetSpl
              ? `Perangkat ini maksimal ~${Math.round(maxSpl)} dB, jadi batas efektifnya ~${Math.round(reachableSpl)} dB.`
              : `Batas efektif ~${Math.round(reachableSpl)} dB (rentang limiter maks. 60 dB di bawah kerasnya perangkat).`}
          </span>
        )}
      </div>
    </div>
  )
}

function Card({title, children}) {
  return (
    <section className="flex flex-col gap-3 rounded-md border border-(--color-border) bg-(--color-surface-2) p-4">
      <h2 className="text-sm font-semibold text-(--color-fg)">{title}</h2>
      {children}
    </section>
  )
}

function SettingsPanel({status, avgOut, settings, ceiling, onCeilingChange, onAutoStartChange, onError}) {
  const [measured, setMeasured] = useState('')
  const [maxSplDraft, setMaxSplDraft] = useState(null)
  const draftTimer = useRef(null)

  const device = status?.deviceOk ? status.device : null
  const maxSpl = maxSplDraft ?? status?.maxSpl ?? 95
  const silent = !status?.listening
  const measuredNum = Number(measured.replace(',', '.'))
  const canCalibrate = device && !silent && Number.isFinite(measuredNum) && measuredNum >= 30 && measuredNum <= 130

  function saveMaxSpl(spl) {
    setMaxSplDraft(spl)
    clearTimeout(draftTimer.current)
    draftTimer.current = setTimeout(() => setMaxSplDraft(null), 600)
    SetCalibration(spl).catch((e) => onError(String(e)))
  }

  function calibrate() {
    // measured = maxSpl + avgOut  →  maxSpl = measured − avgOut
    saveMaxSpl(Math.round((measuredNum - avgOut) * 2) / 2)
    setMeasured('')
  }

  return (
    <div className="flex flex-col gap-4">
      <Card title="Jalan di latar belakang">
        <div className="flex items-start justify-between gap-4">
          <div className="flex flex-col gap-0.5">
            <span className="text-xs font-medium text-(--color-fg)">Jalan otomatis saat Windows menyala</span>
            <span className="text-[11px] text-(--color-muted)">Langsung masuk tray tanpa membuka jendela.</span>
          </div>
          <Switch checked={settings?.autoStart ?? false} onCheckedChange={onAutoStartChange} disabled={!settings} id="autostart" />
        </div>
        <div className="rounded-md bg-(--color-surface-3)/50 px-3 py-2 text-[11px] leading-relaxed text-(--color-secondary)">
          Menutup jendela (tombol ✕) <span className="text-(--color-fg)">tidak</span> mematikan limiterdB — aplikasi tetap
          jalan di system tray (pojok kanan bawah) dan terus mencatat paparan. Klik ikon tray untuk membuka lagi, nyalakan/
          matikan limiter, atau <span className="text-(--color-fg)">Keluar</span>.
        </div>
      </Card>

      <Card title="Kalibrasi perangkat">
        <div className="text-xs text-(--color-secondary)">
          {device ? (
            <>
              <span className="text-(--color-fg)">{device.name}</span> · {KIND_LABEL[device.kind] ?? device.kind} ·{' '}
              {status.calibrated ? 'sudah dikalibrasi' : 'masih perkiraan default'}
            </>
          ) : (
            'Tidak ada perangkat output aktif.'
          )}
        </div>
        <Field label="Kerasnya perangkat di volume 100%" hint={`${maxSpl} dB SPL`}>
          <Slider value={maxSpl} onValueChange={saveMaxSpl} min={70} max={125} step={0.5} disabled={!device} />
        </Field>

        <div className="flex flex-col gap-2">
          <span className="text-xs font-medium text-(--color-secondary)">Kalibrasi pakai aplikasi sound meter di HP</span>
          <span className="text-[11px] text-(--color-muted)">
            Putar lagu, taruh mikrofon HP di posisi telinga (untuk headphone: tempelkan ke bantalan telinga), baca angka
            rata-ratanya, lalu masukkan di sini. Perkiraan saat ini:{' '}
            <span className="text-(--color-fg)">{silent ? '—' : `~${Math.round(maxSpl + avgOut)} dB`}</span>
          </span>
          <div className="flex gap-2">
            <input
              type="text"
              inputMode="decimal"
              value={measured}
              onChange={(e) => setMeasured(e.target.value)}
              placeholder="mis. 72"
              className="h-9 w-28 rounded-md border border-(--color-border) bg-(--color-surface) px-3 text-sm text-(--color-fg) placeholder:text-(--color-placeholder) focus-visible:outline-2 focus-visible:outline-brand-500"
            />
            <Button variant="primary" disabled={!canCalibrate} onClick={calibrate}>
              Pakai angka ini
            </Button>
            {status?.calibrated && (
              <Button onClick={() => ResetCalibration().catch((e) => onError(String(e)))}>Reset default</Button>
            )}
          </div>
          {device && silent && <span className="text-[11px] text-warning">Putar audio dulu supaya bisa dikalibrasi.</span>}
        </div>
      </Card>

      <Card title="Limiter">
        <Field label="Volume maksimum" hint={`${fmtDb(volumeToDb(ceiling))} dB — batas atas saat lagu pelan`}>
          <Slider value={ceiling} onValueChange={onCeilingChange} min={0.01} max={1} step={0.01} />
        </Field>
        <div className="text-[11px] text-(--color-muted)">
          Threshold teknis (dihitung otomatis):{' '}
          <span className="text-(--color-fg)">{fmtDb(status?.threshold ?? 0, 0)} dBFS</span> = batas dengar − kerasnya
          perangkat.
        </div>
      </Card>

      {settings?.dataDir && (
        <p className="text-[11px] text-(--color-muted)">
          Pengaturan & catatan paparan disimpan di <span className="selectable break-all text-(--color-secondary)">{settings.dataDir}</span>
        </p>
      )}
    </div>
  )
}

function App() {
  const [tab, setTab] = useState('limiter')
  const [settings, setSettings] = useState(null)
  const [targetSpl, setTargetSpl] = useState(70)
  const [ceiling, setCeiling] = useState(1)
  const [error, setError] = useState('')
  const migrated = useRef(false)

  const {status, avgSrc, avgOut, error: statusError} = useStatus()

  useEffect(() => {
    Settings()
      .then((s) => {
        setSettings(s)
        setTargetSpl(Math.round(s.targetSpl))
        setCeiling(s.ceiling)
      })
      .catch((e) => setError(String(e)))
  }, [])

  // Versi sebelumnya menyimpan batas dengar & kalibrasi di localStorage;
  // pindahkan sekali ke pengaturan backend.
  useEffect(() => {
    if (migrated.current || !settings || !status?.deviceOk) return
    migrated.current = true
    const legacy = takeLegacySettings(status.device.id)
    if (legacy.targetSpl != null) onTargetChange(legacy.targetSpl)
    if (legacy.maxSpl != null) SetCalibration(legacy.maxSpl).catch(() => {})
  }, [settings, status?.deviceOk])

  function onToggle(next) {
    setError('')
    SetLimiterEnabled(next).catch((e) => setError(String(e)))
  }

  function onTargetChange(spl) {
    setTargetSpl(spl)
    SetTargetSPL(spl).catch((e) => setError(String(e)))
  }

  function onCeilingChange(v) {
    setCeiling(v)
    SetCeiling(v).catch((e) => setError(String(e)))
  }

  function onAutoStartChange(on) {
    SetAutoStart(on)
      .then(() => setSettings((s) => ({...s, autoStart: on})))
      .catch((e) => setError(String(e)))
  }

  const enabled = status?.enabled ?? false

  return (
    <Tabs.Root value={tab} onValueChange={setTab} className="h-full">
      <header className="flex shrink-0 items-center justify-between gap-3 border-b border-(--color-border) bg-(--color-bg) px-4 py-2.5">
        <Tabs.List>
          <Tabs.Tab value="limiter">Limiter</Tabs.Tab>
          <Tabs.Tab value="exposure">Paparan</Tabs.Tab>
          <Tabs.Tab value="settings">Pengaturan</Tabs.Tab>
          <Tabs.Tab value="about">About</Tabs.Tab>
        </Tabs.List>
        <Switch
          checked={enabled}
          onCheckedChange={onToggle}
          disabled={!status}
          label={enabled ? 'Aktif' : 'Mati'}
          id="limiter-switch"
        />
      </header>

      <main className="min-h-0 flex-1 overflow-y-auto bg-(--color-surface) [scrollbar-gutter:stable]">
        <div className="mx-auto w-full max-w-2xl p-5">
          <Tabs.Panel value="limiter">
            <LimiterPanel status={status} avgSrc={avgSrc} avgOut={avgOut} targetSpl={targetSpl} onTargetChange={onTargetChange} />
          </Tabs.Panel>

          <Tabs.Panel value="exposure">{tab === 'exposure' && <ExposurePanel targetSpl={targetSpl} />}</Tabs.Panel>

          <Tabs.Panel value="settings">
            <SettingsPanel
              status={status}
              avgOut={avgOut ?? -100}
              settings={settings}
              ceiling={ceiling}
              onCeilingChange={onCeilingChange}
              onAutoStartChange={onAutoStartChange}
              onError={setError}
            />
          </Tabs.Panel>

          <Tabs.Panel value="about">
            <About
              targetSpl={targetSpl}
              maxSpl={status?.maxSpl ?? 95}
              thresholdDb={status?.threshold ?? 0}
              deviceKind={status?.device?.kind}
            />
          </Tabs.Panel>
        </div>
      </main>

      {(error || statusError) && (
        <footer className="selectable shrink-0 border-t border-danger/40 bg-danger/10 px-4 py-1.5 text-xs text-danger">
          {error || statusError}
        </footer>
      )}
    </Tabs.Root>
  )
}

function Legend({swatch, label}) {
  return (
    <span className="flex items-center gap-1.5">
      <span className={cn('rounded-xs', swatch)} />
      {label}
    </span>
  )
}

export default App
