import { describe, expect, it } from 'vitest'
import { Operation } from './constants'
import { buildRequest, inputForField, inputsOf } from './operations'

describe('inputsOf', () => {
  const cases = [
    { name: 'two-operand operation', operation: Operation.Divide, want: ['leftOperand', 'rightOperand', 'precision'] },
    { name: 'square root takes one operand', operation: Operation.Sqrt, want: ['leftOperand', 'precision'] },
  ] as const

  it.each(cases)('$name', ({ operation, want }) => {
    expect(inputsOf(operation)).toEqual(want)
  })
})

describe('buildRequest', () => {
  const cases = [
    {
      name: 'two operands without precision',
      operation: Operation.Add,
      operands: { left: 1.5, right: 2, precision: null },
      want: { leftOperand: 1.5, rightOperand: 2 },
    },
    {
      name: 'two operands with precision',
      operation: Operation.Percentage,
      operands: { left: 20, right: 50, precision: 2 },
      want: { leftOperand: 20, rightOperand: 50, precision: 2 },
    },
    {
      name: 'precision zero is sent',
      operation: Operation.Divide,
      operands: { left: 1, right: 3, precision: 0 },
      want: { leftOperand: 1, rightOperand: 3, precision: 0 },
    },
    {
      name: 'square root sends a single operand',
      operation: Operation.Sqrt,
      operands: { left: 9, right: Number.NaN, precision: 3 },
      want: { operand: 9, precision: 3 },
    },
  ] as const

  it.each(cases)('$name', ({ operation, operands, want }) => {
    expect(buildRequest(operation, operands)).toEqual(want)
  })
})

describe('inputForField', () => {
  const cases = [
    { name: 'left operand', operation: Operation.Add, field: 'leftOperand', want: 'leftOperand' },
    { name: 'right operand', operation: Operation.Add, field: 'rightOperand', want: 'rightOperand' },
    { name: 'precision', operation: Operation.Add, field: 'precision', want: 'precision' },
    { name: 'unknown property', operation: Operation.Add, field: 'zzz', want: undefined },
    { name: 'binary operation has no operand field', operation: Operation.Add, field: 'operand', want: undefined },
    { name: 'square root operand maps to the left input', operation: Operation.Sqrt, field: 'operand', want: 'leftOperand' },
    { name: 'square root precision', operation: Operation.Sqrt, field: 'precision', want: 'precision' },
    { name: 'square root has no right operand', operation: Operation.Sqrt, field: 'rightOperand', want: undefined },
  ] as const

  it.each(cases)('$name', ({ operation, field, want }) => {
    expect(inputForField(operation, field)).toBe(want)
  })
})
