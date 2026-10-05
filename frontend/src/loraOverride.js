export function sanitizeWeight(v) {
  if (v === '' || v === null || v === undefined) return 0.6
  const n = Number(v)
  if (!Number.isFinite(n) || n < 0) return 0.6
  return Math.round(Math.min(n, 2) * 100) / 100
}

function normRows(rows) {
  if (!Array.isArray(rows)) return []
  const out = []
  for (const r of rows) {
    if (!r || typeof r.name !== 'string' || !r.name.trim()) continue
    out.push({ name: r.name.trim(), weight: sanitizeWeight(r.weight) })
  }
  return out
}

export function parseLoras(jsonStr) {
  if (!jsonStr) return []
  try {
    return normRows(JSON.parse(jsonStr))
  } catch {
    return []
  }
}

export function sameLoras(a, b) {
  const x = normRows(a)
  const y = normRows(b)
  if (x.length !== y.length) return false
  return x.every((l, i) => l.name === y[i].name && l.weight === y[i].weight)
}

export function buildLorasOverride(original, edited, ignoreAll) {
  if (ignoreAll) return '[]'
  const rows = normRows(edited)
  if (sameLoras(original, rows)) return null
  return JSON.stringify(rows)
}
