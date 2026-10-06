import { describe, it, expect } from 'vitest'
import { splitHighlight } from '../highlight.js'

describe('splitHighlight', () => {
  it('returns single plain segment when query is empty', () => {
    expect(splitHighlight('hello world', '')).toEqual([{ text: 'hello world', hit: false }])
    expect(splitHighlight('hello world', '   ')).toEqual([{ text: 'hello world', hit: false }])
    expect(splitHighlight('', 'q')).toEqual([])
  })

  it('handles null-ish input', () => {
    expect(splitHighlight(null, undefined)).toEqual([{ text: '', hit: false }])
  })

  it('splits on every case-insensitive match', () => {
    expect(splitHighlight('Cat catalog CAT', 'cat')).toEqual([
      { text: 'Cat', hit: true },
      { text: ' ', hit: false },
      { text: 'cat', hit: true },
      { text: 'alog ', hit: false },
      { text: 'CAT', hit: true },
    ])
  })

  it('returns plain segment when there is no match', () => {
    expect(splitHighlight('abc', 'xyz')).toEqual([{ text: 'abc', hit: false }])
  })

  it('returns plain segment when query is longer than text', () => {
    expect(splitHighlight('ab', 'abc')).toEqual([{ text: 'ab', hit: false }])
  })

  it('marks the whole string when it equals the query', () => {
    expect(splitHighlight('idea', 'IDEA')).toEqual([{ text: 'idea', hit: true }])
  })

  it('trims query before matching', () => {
    expect(splitHighlight('a b', ' b ')).toEqual([
      { text: 'a ', hit: false },
      { text: 'b', hit: true },
    ])
  })
})
