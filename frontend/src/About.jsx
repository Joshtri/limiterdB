import {cn} from '@/lib/cn'
import {fmtDb, LIMITER_PARAMS} from '@/lib/levels'
import {
  DEFAULT_MAX_SPL,
  fmtSafeTime,
  KIND_LABEL,
  LIMIT_SPL,
  REFERENCES,
  SAFE_SPL,
  ZONE_STYLE,
  zoneOfSpl,
} from '@/lib/hearing'

function Section({title, children}) {
  return (
    <section className="flex flex-col gap-2">
      <h2 className="text-sm font-semibold text-(--color-fg)">{title}</h2>
      <div className="flex flex-col gap-2 text-[13px] leading-relaxed text-(--color-secondary)">{children}</div>
    </section>
  )
}

function Formula({children}) {
  return (
    <div className="rounded-md border border-(--color-border) bg-(--color-surface-2) px-3 py-2 font-mono text-xs text-(--color-fg)">
      {children}
    </div>
  )
}

function Step({n, title, children}) {
  return (
    <div className="flex gap-3 rounded-md border border-(--color-border) bg-(--color-surface-2) p-3">
      <span className="flex size-5 shrink-0 items-center justify-center rounded-full bg-brand-500 text-[11px] font-semibold text-white">
        {n}
      </span>
      <div className="flex flex-col gap-0.5">
        <span className="text-xs font-semibold text-(--color-fg)">{title}</span>
        <span className="text-xs text-(--color-secondary)">{children}</span>
      </div>
    </div>
  )
}

function B({children}) {
  return <strong className="text-(--color-fg)">{children}</strong>
}

export function About({targetSpl, maxSpl, thresholdDb, deviceKind}) {
  const {tickMs, attack, releaseStep, minVolume} = LIMITER_PARAMS
  const releasePerSec = (releaseStep * 100 * 1000) / tickMs // % gain per detik

  return (
    <div className="flex flex-col gap-6">
      <Section title="Apa yang dilakukan limiterdB?">
        <p>
          Kamu cukup menentukan <B>batas dengar</B>: seberapa keras suara boleh sampai ke telinga, dalam dB SPL (satuan
          yang sama dengan sound meter). limiterdB lalu mengatur volume Windows terus-menerus supaya lagu apa pun —
          keras atau pelan — tidak melewati batas itu.
        </p>
        <p>
          Volumenya <B>tidak konstan</B>: dihitung ulang setiap {tickMs} ms dari kerasnya audio yang sedang diputar.
          Lagu keras diturunkan, lagu pelan dibiarkan. Perangkat output juga diikuti otomatis — ganti headset atau
          speaker di Windows, limiterdB pindah sendiri dalam ±0,5 detik.
        </p>
        <p>
          limiterdB dibuat untuk <B>jalan terus di latar belakang</B>, seperti fitur bawaan di iPhone: menutup jendela
          hanya menyembunyikannya ke system tray. Warna ikon tray menunjukkan kondisi saat ini (hijau/kuning/merah;
          cincin = limiter mati), dan bisa diatur supaya jalan otomatis saat Windows menyala.
        </p>
      </Section>

      <Section title="Catatan paparan (seperti iPhone)">
        <p>
          Selain kerasnya saat ini, yang penting adalah <B>dosis</B>: keras × lama. limiterdB mencatat setiap detik ada
          audio terdengar, lalu menjumlahkan energinya per hari. Patokannya sama dengan Headphone Audio Level di
          iPhone (WHO): <B>rata-rata 80 dB selama 40 jam per 7 hari = 100%</B>.
        </p>
        <Formula>dosis = Σ detik × 10^((dB − 80) / 10)  ÷  (40 jam)</Formula>
        <p>
          Karena aturan 3 dB, jatah 40 jam itu menyusut cepat kalau lebih keras: 83 dB → 20 jam, 86 dB → 10 jam, 90 dB →
          4 jam per minggu. Supaya seminggu tidak lewat 100%, rata-rata per hari sebaiknya ≤ 14%. Riwayatnya bisa dilihat
          di tab <em>Paparan</em> (disimpan 30 hari).
        </p>
      </Section>

      <Section title="Berapa desibel yang aman?">
        <p>
          Risiko kerusakan pendengaran bergantung pada <B>seberapa keras</B> dan <B>berapa lama</B>. Patokan yang
          dipakai:
        </p>
        <ul className="list-disc space-y-1 pl-5">
          <li>
            <B>WHO</B>: rata-rata {SAFE_SPL} dB aman untuk ±40 jam per minggu (dewasa).
          </li>
          <li>
            <B>NIOSH</B>: {LIMIT_SPL} dB maksimal 8 jam per hari, dan waktu amannya <B>separuh setiap +3 dB</B>.
          </li>
        </ul>
        <div className="flex flex-col divide-y divide-(--color-border) rounded-md border border-(--color-border) bg-(--color-surface-2) px-3">
          {REFERENCES.map((r) => {
            const style = ZONE_STYLE[zoneOfSpl(r.spl)]
            return (
              <div key={r.spl} className="flex items-center gap-3 py-1.5 text-xs">
                <span className={cn('size-2 shrink-0 rounded-full', style.dot)} />
                <span className="w-12 shrink-0 font-semibold tabular-nums text-(--color-fg)">{r.spl} dB</span>
                <span className="flex-1 text-(--color-secondary)">{r.label}</span>
                <span className="shrink-0 tabular-nums text-(--color-muted)">{fmtSafeTime(r.spl)}</span>
              </div>
            )
          })}
        </div>
        <p>
          Warna meter mengikuti tabel ini: <span className="text-success">hijau ≤ {SAFE_SPL} dB</span>,{' '}
          <span className="text-warning">
            kuning {SAFE_SPL}–{LIMIT_SPL} dB
          </span>
          , <span className="text-danger">merah &gt; {LIMIT_SPL} dB</span>. Batasnya absolut — sama untuk semua lagu dan
          perangkat.
        </p>
      </Section>

      <Section title="dBFS vs dB SPL — kenapa perlu kalibrasi">
        <p>
          Windows hanya tahu <B>dBFS</B>: seberapa besar sinyal digital (0 dBFS = paling besar). Windows tidak tahu
          seberapa keras sinyal itu keluar dari headset kamu — itu tergantung perangkatnya. Jembatannya adalah{' '}
          <B>kerasnya perangkat di volume 100%</B>:
        </p>
        <Formula>dB SPL ≈ kerasnya perangkat + level lagu (dBFS) + volume Windows (dB)</Formula>
        <p>Tanpa kalibrasi dipakai perkiraan per jenis perangkat (dideteksi otomatis dari Windows):</p>
        <div className="grid grid-cols-2 gap-2 text-xs sm:grid-cols-3">
          {Object.entries(DEFAULT_MAX_SPL).map(([kind, spl]) => (
            <div
              key={kind}
              className={cn(
                'rounded-md border bg-(--color-surface-2) p-2',
                kind === deviceKind ? 'border-brand-500' : 'border-(--color-border)'
              )}
            >
              <div className="font-semibold text-(--color-fg)">{KIND_LABEL[kind]}</div>
              <div className="text-(--color-muted)">~{spl} dB di volume 100%</div>
            </div>
          ))}
        </div>
        <p>
          Perangkat bisa berbeda ±10 dB dari perkiraan ini. Supaya angkanya pas, buka{' '}
          tab <em>Pengaturan</em>, ukur pakai aplikasi sound meter di HP, lalu masukkan angkanya.
          Kalibrasi disimpan per perangkat.
        </p>
      </Section>

      <Section title="Cara kerjanya">
        <div className="flex flex-col gap-2">
          <Step n={1} title="Ikuti perangkat default">
            Tiap 0,5 detik limiterdB mengecek perangkat output default Windows. Kalau berganti (atau dicabut), koneksi
            lama dilepas dan perangkat baru dibuka beserta kalibrasinya.
          </Step>
          <Step n={2} title="Batas dengar → threshold">
            Threshold dalam dBFS dihitung otomatis: <B>batas dengar − kerasnya perangkat</B>. Sekarang:{' '}
            {targetSpl} − {maxSpl} = <B>{fmtDb(thresholdDb, 0)} dBFS</B>.
          </Step>
          <Step n={3} title={`Ukur (tiap ${tickMs} ms)`}>
            Windows memberi nilai peak audio yang sedang diputar, diukur <em>sebelum</em> volume master. Itu kerasnya
            lagu itu sendiri.
          </Step>
          <Step n={4} title="Hitung volume target">
            Volume = 10^((threshold − level) / 20), dibatasi antara {fmtDb(20 * Math.log10(minVolume), 0)} dB (tidak
            pernah mute total) dan volume maksimum.
          </Step>
          <Step n={5} title={`Turun cepat, naik pelan`}>
            Terlalu keras: tiap tick volume menutup {Math.round(attack * 100)}% jarak ke target (±100–200 ms). Aman:
            naik pelan ±{releasePerSec.toFixed(0)}% per detik supaya tidak naik-turun bergelombang.
          </Step>
        </div>
      </Section>

      <Section title="Contoh: headphone (~100 dB), batas dengar 70 dB">
        <p>Threshold = 70 − 100 = −30 dBFS.</p>
        <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
          <div className="rounded-md border border-(--color-border) bg-(--color-surface-2) p-3 text-xs">
            <div className="font-semibold text-(--color-fg)">Lagu keras, level −6 dBFS</div>
            <div className="mt-1 text-(--color-secondary)">Tanpa limiter: 100 − 6 = ~94 dB (aman ±1 jam)</div>
            <div className="font-mono text-(--color-secondary)">volume = −30 − (−6) = −24 dB</div>
            <div className="mt-1 text-(--color-fg)">→ terdengar ~70 dB</div>
          </div>
          <div className="rounded-md border border-(--color-border) bg-(--color-surface-2) p-3 text-xs">
            <div className="font-semibold text-(--color-fg)">Lagu pelan, level −35 dBFS</div>
            <div className="mt-1 text-(--color-secondary)">Tanpa limiter: 100 − 35 = ~65 dB</div>
            <div className="font-mono text-(--color-secondary)">volume = −30 − (−35) = +5 dB → maks. 0 dB</div>
            <div className="mt-1 text-(--color-fg)">→ dibiarkan, terdengar ~65 dB</div>
          </div>
        </div>
      </Section>

      <Section title="Batasan">
        <ul className="list-disc space-y-1 pl-5">
          <li>
            Angka dB SPL adalah <B>perkiraan</B>. Tanpa kalibrasi bisa meleset ±10 dB; setelah kalibrasi biasanya
            dalam beberapa dB.
          </li>
          <li>
            Perkiraannya cenderung sedikit <em>lebih tinggi</em> dari yang sebenarnya (dihitung dari peak, bukan
            rata-rata energi), jadi meleset ke arah yang lebih aman.
          </li>
          <li>
            Kalau kamu mengubah volume langsung di headset/speaker (tombol fisik yang terpisah dari Windows), kalibrasi
            jadi tidak cocok — kalibrasi ulang.
          </li>
          <li>
            limiterdB mengatur volume master Windows, bukan memproses audio langsung. Ada jeda ±{tickMs}–{tickMs * 2}{' '}
            ms, jadi hentakan yang sangat mendadak bisa lolos sesaat.
          </li>
          <li>Ini alat bantu, bukan alat ukur medis.</li>
        </ul>
      </Section>
    </div>
  )
}
