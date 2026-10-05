import { describe, it, expect } from 'vitest'
import { parseLoras, sameLoras, buildLorasOverride, sanitizeWeight } from '../loraOverride.js'

describe('parseLoras', () => {
  it('parses a valid loras JSON string', () => {
    expect(parseLoras('[{"name":"x","weight":0.4},{"name":"y","weight":1}]'))
      .toEqual([{ name: 'x', weight: 0.4 }, { name: 'y', weight: 1 }])
  })

  it('returns [] for empty string', () => {
    expect(parseLoras('')).toEqual([])
    expect(parseLoras(null)).toEqual([])
  })

  it('returns [] for "[]"', () => {
    expect(parseLoras('[]')).toEqual([])
  })

  it('returns [] for garbage input', () => {
    expect(parseLoras('not json')).toEqual([])
    expect(parseLoras('{"name":"x"}')).toEqual([])
  })

  it('drops entries without a name and sanitizes weights', () => {
    expect(parseLoras('[{"name":"","weight":0.4},{"weight":0.4},{"name":"x","weight":9}]'))
      .toEqual([{ name: 'x', weight: 2 }])
  })
})

describe('sameLoras', () => {
  it('matches identical lists', () => {
    expect(sameLoras([{ name: 'x', weight: 0.4 }], [{ name: 'x', weight: 0.4 }])).toBe(true)
  })

  it('detects weight, name and length differences', () => {
    expect(sameLoras([{ name: 'x', weight: 0.4 }], [{ name: 'x', weight: 0.5 }])).toBe(false)
    expect(sameLoras([{ name: 'x', weight: 0.4 }], [{ name: 'y', weight: 0.4 }])).toBe(false)
    expect(sameLoras([{ name: 'x', weight: 0.4 }], [])).toBe(false)
  })

  it('treats empty lists as equal', () => {
    expect(sameLoras([], [])).toBe(true)
  })
})

describe('buildLorasOverride', () => {
  it('returns null when nothing was touched', () => {
    expect(buildLorasOverride([{ name: 'x', weight: 0.4 }], [{ name: 'x', weight: 0.4 }], false)).toBeNull()
    expect(buildLorasOverride([], [], false)).toBeNull()
  })

  it('returns "[]" when ignoreAll is set', () => {
    expect(buildLorasOverride([{ name: 'x', weight: 0.4 }], [{ name: 'x', weight: 0.4 }], true)).toBe('[]')
  })

  it('returns new JSON when a weight was changed', () => {
    expect(buildLorasOverride([{ name: 'x', weight: 0.4 }], [{ name: 'x', weight: 0.8 }], false))
      .toBe('[{"name":"x","weight":0.8}]')
  })

  it('returns "[]" when all rows were removed manually', () => {
    expect(buildLorasOverride([{ name: 'x', weight: 0.4 }], [], false)).toBe('[]')
  })

  it('ignores rows with empty names and falls back to null', () => {
    expect(buildLorasOverride([], [{ name: '', weight: 0.6 }], false)).toBeNull()
  })
})

describe('sanitizeWeight', () => {
  it('defaults to 0.6 on empty or garbage input', () => {
    expect(sanitizeWeight('')).toBe(0.6)
    expect(sanitizeWeight(null)).toBe(0.6)
    expect(sanitizeWeight(undefined)).toBe(0.6)
    expect(sanitizeWeight('abc')).toBe(0.6)
    expect(sanitizeWeight(-1)).toBe(0.6)
  })

  it('clamps to 0..2 and rounds to hundredths', () => {
    expect(sanitizeWeight(0)).toBe(0)
    expect(sanitizeWeight(2)).toBe(2)
    expect(sanitizeWeight(5)).toBe(2)
    expect(sanitizeWeight(0.66666)).toBe(0.67)
    expect(sanitizeWeight('1.5')).toBe(1.5)
  })
})

test('parseLoras trims names', () => {
  expect(parseLoras('[{"name":"  x  ","weight":0.5}]')).toEqual([{ name: 'x', weight: 0.5 }])
})

test('buildLorasOverride: edited back to original -> null', () => {
  const original = [{ name: 'a', weight: 0.5 }]
  const edited = [{ name: 'a', weight: 0.9 }]
  expect(buildLorasOverride(original, edited, false)).toBe(JSON.stringify([{ name: 'a', weight: 0.9 }]))
  expect(buildLorasOverride(original, original, false)).toBeNull()
})
