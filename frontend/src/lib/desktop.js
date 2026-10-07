// Perilaku "aplikasi desktop" untuk WebView: zoom hanya lewat keyboard,
// tanpa zoom dari scroll/pinch, tanpa drag gambar/teks.
//
// Wails mematikan accelerator browser (termasuk Ctrl+/Ctrl-) dan zoom bawaan
// WebView2 dimatikan di main.go, jadi zoom diimplementasikan sendiri pakai CSS zoom.

const ZOOM_STEPS = [0.67, 0.75, 0.8, 0.9, 1, 1.1, 1.25, 1.5, 1.75, 2]
const ZOOM_KEY = 'limiterdB.zoom'

function loadZoom() {
  try {
    const z = Number(localStorage.getItem(ZOOM_KEY))
    return ZOOM_STEPS.includes(z) ? z : 1
  } catch {
    return 1
  }
}

function applyZoom(root, z) {
  root.style.zoom = z === 1 ? '' : String(z)
  try {
    localStorage.setItem(ZOOM_KEY, String(z))
  } catch {}
}

export function installDesktopBehavior(root) {
  let zoom = loadZoom()
  applyZoom(root, zoom)

  function step(dir) {
    const i = ZOOM_STEPS.indexOf(zoom)
    zoom = ZOOM_STEPS[Math.min(ZOOM_STEPS.length - 1, Math.max(0, i + dir))]
    applyZoom(root, zoom)
  }

  window.addEventListener('keydown', (e) => {
    if (!e.ctrlKey || e.altKey || e.metaKey) return
    if (e.key === '+' || e.key === '=' || e.code === 'NumpadAdd') step(1)
    else if (e.key === '-' || e.key === '_' || e.code === 'NumpadSubtract') step(-1)
    else if (e.key === '0' || e.code === 'Numpad0') applyZoom(root, (zoom = 1))
    else return
    e.preventDefault()
  })

  // Cadangan kalau zoom bawaan WebView2 masih aktif: Ctrl+scroll dan pinch
  // touchpad (dikirim sebagai wheel + ctrlKey) tidak boleh men-zoom.
  window.addEventListener('wheel', (e) => e.ctrlKey && e.preventDefault(), {passive: false})

  // Gambar dan teks tidak bisa di-drag keluar seperti di halaman web.
  window.addEventListener('dragstart', (e) => e.preventDefault())
}
