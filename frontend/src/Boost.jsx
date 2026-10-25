import {useEffect, useState} from 'react'
import {BoostInfo, LinkEqualizerAPO} from '../wailsjs/go/main/App'
import {BrowserOpenURL} from '../wailsjs/runtime/runtime'
import {Button} from '@/components/button'
import {Field} from '@/components/field'
import {Slider} from '@/components/slider'

const EQAPO_URL = 'https://sourceforge.net/projects/equalizerapo/'
// Cek ulang Equalizer APO berkala: user bisa memasangnya / menyetujui UAC
// sementara jendela terbuka.
const INFO_POLL_MS = 2000

const boostToDb = (pct) => (pct <= 100 ? 0 : 20 * Math.log10(pct / 100))

// BoostCard: volume speaker di atas 100% (seperti "over-amplification" di
// Ubuntu) lewat Equalizer APO. Hanya tampil saat device bukan headphone.
export function BoostCard({speakerBoost, onBoostChange, status, onError}) {
  const [info, setInfo] = useState(null)
  const [linking, setLinking] = useState(false)

  useEffect(() => {
    let alive = true
    const load = () =>
      BoostInfo()
        .then((v) => alive && setInfo(v))
        .catch(() => {})
    load()
    const id = setInterval(load, INFO_POLL_MS)
    return () => {
      alive = false
      clearInterval(id)
    }
  }, [])

  function link() {
    setLinking(true)
    LinkEqualizerAPO()
      .then(() => BoostInfo().then(setInfo))
      .catch((e) => onError(String(e)))
      .finally(() => setLinking(false))
  }

  return (
    <section className="flex flex-col gap-3 rounded-md border border-(--color-border) bg-(--color-surface-2) p-4">
      <div className="flex items-baseline justify-between">
        <h2 className="text-sm font-semibold text-(--color-fg)">Boost volume speaker</h2>
        <span className="text-xs text-(--color-muted)">sampai 200%, khusus speaker</span>
      </div>

      {!info ? null : !info.installed ? (
        <div className="flex flex-col gap-2 text-xs text-(--color-secondary)">
          <span>
            Windows membatasi volume di 100%. Untuk lebih keras, limiterdB memakai{' '}
            <span className="text-(--color-fg)">Equalizer APO</span> (gratis, open-source):
          </span>
          <ol className="ml-4 list-decimal space-y-0.5 text-[11px] text-(--color-muted)">
            <li>Install Equalizer APO.</li>
            <li>Di jendela Configurator, centang speaker kamu, lalu restart Windows.</li>
            <li>Buka limiterdB lagi — slider boost muncul di sini.</li>
          </ol>
          <div>
            <Button onClick={() => BrowserOpenURL(EQAPO_URL)}>Download Equalizer APO</Button>
          </div>
        </div>
      ) : !info.linked ? (
        <div className="flex flex-col gap-2 text-xs text-(--color-secondary)">
          <span>
            Equalizer APO terpasang. Hubungkan sekali supaya limiterdB bisa mengatur boost — Windows mungkin meminta izin
            admin.
          </span>
          <div>
            <Button variant="primary" disabled={linking} onClick={link}>
              {linking ? 'Menghubungkan…' : 'Hubungkan Equalizer APO'}
            </Button>
          </div>
        </div>
      ) : (
        <>
          <Field
            label="Volume maksimum speaker"
            hint={`${Math.round(speakerBoost)}%${speakerBoost > 100 ? ` · +${boostToDb(speakerBoost).toFixed(1)} dB` : ''}`}
          >
            <Slider value={speakerBoost} onValueChange={onBoostChange} min={100} max={200} step={5} />
          </Field>
          {speakerBoost > 100 && (
            <span className="text-[11px] text-warning">
              Lagu yang sudah keras bisa terdengar pecah (clipping) — turunkan boost kalau suaranya jadi kasar.
            </span>
          )}
          <span className="text-[11px] text-(--color-muted)">
            Boost otomatis mati saat pindah ke headphone/headset dan saat limiterdB ditutup. Kalau tidak ada efek, buka
            Equalizer APO Configurator dan pastikan speaker ini dicentang.
          </span>
        </>
      )}

      {status?.boostError && info?.linked && <span className="text-[11px] text-danger">{status.boostError}</span>}
    </section>
  )
}
