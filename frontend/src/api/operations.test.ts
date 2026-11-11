import { describe, expect, it } from 'vitest'
import { Operation } from '@/api/constants'
import { buildRequest, isBinary, type Operands } from '@/api/operations'
import type { OperationRequest } from '@/api/types'

describe('isBinary', () => {
  const cases = [
    { name: 'add', operation: Operation.Add, want: true },
    { name: 'subtract', operation: Operation.Subtract, want: true },
    { name: 'multiply', operation: Operation.Multiply, want: true },
    { name: 'divide', operation: Operation.Divide, want: true },
    { name: 'power', operation: Operation.Power, want: true },
    { name: 'percentage', operation: Operation.Percentage, want: true },
    { name: 'square root takes one operand', operation: Operation.Sqrt, want: false },
  ] as const

  it.each(cases)('$name', ({ operation, want }) => {
    expect(isBinary(operation)).toBe(want)
  })
})

describe('buildRequest', () => {
  const cases: { name: string; operands: Operands; want: OperationRequest }[] = [
    {
      name: 'two operands without precision',
      operands: { binary: true, operation: Operation.Add, left: 1.5, right: 2, precision: null },
      want: { leftOperand: 1.5, rightOperand: 2 },
    },
    {
      name: 'two operands with precision',
      operands: { binary: true, operation: Operation.Percentage, left: 20, right: 50, precision: 2 },
      want: { leftOperand: 20, rightOperand: 50, precision: 2 },
    },
    {
      name: 'precision zero is sent',
      operands: { binary: true, operation: Operation.Divide, left: 1, right: 3, precision: 0 },
      want: { leftOperand: 1, rightOperand: 3, precision: 0 },
    },
    {
      name: 'square root sends a single operand',
      operands: { binary: false, operation: Operation.Sqrt, left: 9, precision: 3 },
      want: { operand: 9, precision: 3 },
    },
    {
      name: 'square root without precision',
      operands: { binary: false, operation: Operation.Sqrt, left: 2.25, precision: null },
      want: { operand: 2.25 },
    },
  ]

  it.each(cases)('$name', ({ operands, want }) => {
    expect(buildRequest(operands)).toEqual(want)
  })
})
