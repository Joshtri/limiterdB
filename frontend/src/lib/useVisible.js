import {useEffect, useState} from 'react'

// true selama jendela terlihat. Saat limiterdB disembunyikan ke tray,
// polling dari UI berhenti — backend tetap jalan sendiri.
export function useVisible() {
  const [visible, setVisible] = useState(() => document.visibilityState !== 'hidden')
  useEffect(() => {
    const onChange = () => setVisible(document.visibilityState !== 'hidden')
    document.addEventListener('visibilitychange', onChange)
    return () => document.removeEventListener('visibilitychange', onChange)
  }, [])
  return visible
}
