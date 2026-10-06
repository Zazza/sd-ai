export function splitHighlight(text, query) {
  const str = String(text ?? '')
  const q = String(query ?? '').trim()
  if (!q) return [{ text: str, hit: false }]
  const needle = q.toLowerCase()
  const lower = str.toLowerCase()
  const segments = []
  let i = 0
  while (i < str.length) {
    const at = lower.indexOf(needle, i)
    if (at === -1) {
      segments.push({ text: str.slice(i), hit: false })
      break
    }
    if (at > i) segments.push({ text: str.slice(i, at), hit: false })
    segments.push({ text: str.slice(at, at + needle.length), hit: true })
    i = at + needle.length
  }
  return segments
}
