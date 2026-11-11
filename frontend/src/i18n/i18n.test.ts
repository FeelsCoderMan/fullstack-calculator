import { describe, expect, it } from 'vitest'
import { Operation } from '@/api/constants'
import { errorMessage, fieldErrorMessage, formatNumber, operationText } from '@/i18n/index'

describe('errorMessage', () => {
  const cases = [
    { name: 'math error', code: 'error.calculation.division.by.zero', want: 'You can’t divide by zero.' },
    {
      name: 'zero to a negative power',
      code: 'error.calculation.zero.negative.power',
      want: 'Zero can’t be raised to a negative power: that divides by zero.',
    },
    { name: 'missing route', code: 'error.request.not.found', want: 'The server doesn’t support this request. Try reloading the page.' },
    { name: 'client-side network failure', code: 'error.client.network', want: 'Can’t reach the server. Check your connection and try again.' },
    { name: 'field code reported for the whole form', code: 'error.field.unknown', want: 'The server did not expect this value.' },
    { name: 'unknown code falls back', code: 'error.something.new', want: 'Something went wrong. Please try again.' },
    { name: 'inherited object key is not a code', code: 'toString', want: 'Something went wrong. Please try again.' },
  ] as const

  it.each(cases)('$name', ({ code, want }) => {
    expect(errorMessage(code)).toBe(want)
  })
})

describe('fieldErrorMessage', () => {
  const cases = [
    { name: 'required', code: 'error.field.required', want: 'Enter a number.' },
    { name: 'invalid type', code: 'error.field.invalid.type', want: 'Enter a valid number, like 12.5 or -3e4.' },
    { name: 'out of range', code: 'error.field.out.of.range', want: 'This number is too large.' },
    { name: 'unknown code falls back', code: 'error.field.new', want: 'Something went wrong. Please try again.' },
  ] as const

  it.each(cases)('$name', ({ code, want }) => {
    expect(fieldErrorMessage(code)).toBe(want)
  })
})

describe('two-operand expression', () => {
  const cases = [
    { name: 'add', operation: Operation.Add, left: '3', right: '4', want: '3 + 4' },
    { name: 'subtract', operation: Operation.Subtract, left: '3', right: '4', want: '3 − 4' },
    { name: 'multiply', operation: Operation.Multiply, left: '3', right: '4', want: '3 × 4' },
    { name: 'divide', operation: Operation.Divide, left: '3', right: '4', want: '3 ÷ 4' },
    { name: 'power', operation: Operation.Power, left: '2', right: '10', want: '2 ^ 10' },
    { name: 'percentage', operation: Operation.Percentage, left: '20', right: '50', want: '20% of 50' },
  ] as const

  it.each(cases)('$name', ({ operation, left, right, want }) => {
    expect(operationText(operation).expression(left, right)).toBe(want)
  })
})

describe('one-operand expression', () => {
  const cases = [{ name: 'square root', operation: Operation.Sqrt, operand: '9', want: '√9' }] as const

  it.each(cases)('$name', ({ operation, operand, want }) => {
    expect(operationText(operation).expression(operand)).toBe(want)
  })
})

describe('formatNumber', () => {
  const cases = [
    { name: 'float noise is shown as is', value: 0.1 + 0.2, want: '0.30000000000000004' },
    { name: 'integer', value: -2, want: '-2' },
    { name: 'very large number', value: 1.7976931348623157e308, want: '1.7976931348623157e+308' },
  ] as const

  it.each(cases)('$name', ({ value, want }) => {
    expect(formatNumber(value)).toBe(want)
  })
})
