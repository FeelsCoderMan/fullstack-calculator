import { describe, expect, it } from 'vitest'
import { parseOperand, parsePrecision, validateOperand, validatePrecision } from './validation'

describe('validateOperand', () => {
  const cases = [
    { name: 'empty input', input: '', want: 'error.field.required' },
    { name: 'only whitespace', input: '   ', want: 'error.field.required' },
    { name: 'integer', input: '42', want: null },
    { name: 'negative decimal', input: '-4.5', want: null },
    { name: 'explicit plus sign', input: '+3', want: null },
    { name: 'leading dot', input: '.5', want: null },
    { name: 'trailing dot', input: '4.', want: null },
    { name: 'exponent form', input: '1e-3', want: null },
    { name: 'upper-case exponent', input: '2.5E10', want: null },
    { name: 'surrounding whitespace', input: '  7  ', want: null },
    { name: 'zero', input: '0', want: null },
    { name: 'negative zero', input: '-0', want: null },
    { name: 'largest float64', input: '1.7976931348623157e308', want: null },
    { name: 'letters', input: 'abc', want: 'error.field.invalid.type' },
    { name: 'two dots', input: '1.2.3', want: 'error.field.invalid.type' },
    { name: 'comma as decimal separator', input: '1,5', want: 'error.field.invalid.type' },
    { name: 'hexadecimal', input: '0x10', want: 'error.field.invalid.type' },
    { name: 'Infinity literal', input: 'Infinity', want: 'error.field.invalid.type' },
    { name: 'NaN literal', input: 'NaN', want: 'error.field.invalid.type' },
    { name: 'lone sign', input: '-', want: 'error.field.invalid.type' },
    { name: 'lone dot', input: '.', want: 'error.field.invalid.type' },
    { name: 'exponent without digits', input: '1e', want: 'error.field.invalid.type' },
    { name: 'space inside the number', input: '1 000', want: 'error.field.invalid.type' },
    { name: 'beyond float64', input: '1e309', want: 'error.field.out.of.range' },
    { name: 'beyond float64, negative', input: '-1e309', want: 'error.field.out.of.range' },
  ] as const

  it.each(cases)('$name', ({ input, want }) => {
    expect(validateOperand(input)).toBe(want)
  })
})

describe('parseOperand', () => {
  const cases = [
    { name: 'decimal', input: '-4.5', want: -4.5 },
    { name: 'surrounding whitespace', input: ' 12 ', want: 12 },
    { name: 'exponent form', input: '1e-3', want: 0.001 },
  ] as const

  it.each(cases)('$name', ({ input, want }) => {
    expect(parseOperand(input)).toBe(want)
  })
})

describe('validatePrecision', () => {
  const cases = [
    { name: 'empty means full precision', input: '', want: null },
    { name: 'lowest value', input: '0', want: null },
    { name: 'highest value', input: '15', want: null },
    { name: 'above the range', input: '16', want: 'error.field.out.of.range' },
    { name: 'negative', input: '-1', want: 'error.field.out.of.range' },
    { name: 'decimal', input: '2.5', want: 'error.field.invalid.type' },
    { name: 'not a number', input: 'two', want: 'error.field.invalid.type' },
  ] as const

  it.each(cases)('$name', ({ input, want }) => {
    expect(validatePrecision(input)).toBe(want)
  })
})

describe('parsePrecision', () => {
  const cases = [
    { name: 'empty means full precision', input: '', want: null },
    { name: 'number of places', input: '2', want: 2 },
  ] as const

  it.each(cases)('$name', ({ input, want }) => {
    expect(parsePrecision(input)).toBe(want)
  })
})
