export function parseSeedInput(v) {
  if (v === '' || v === null || v === undefined) return null
  const n = Number(v)
  if (!Number.isFinite(n) || n < 0) return null
  return Math.min(Math.floor(n), Number.MAX_SAFE_INTEGER)
}
