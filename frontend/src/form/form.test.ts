import { describe, expect, it } from 'vitest'
import { Operation } from '@/api/constants'
import type { FieldError } from '@/api/types'
import {
  EMPTY_VALUES,
  type FieldErrors,
  firstInvalidInput,
  type FormValues,
  inputForField,
  inputsOf,
  mapApiFieldErrors,
  toOperands,
  validateForm,
} from '@/form/form'

/** Form values with the given inputs filled in, the rest empty. */
function formWith(filled: Partial<FormValues>): FormValues {
  return { ...EMPTY_VALUES, ...filled }
}

describe('inputsOf', () => {
  const cases = [
    { name: 'two-operand operation', operation: Operation.Divide, want: ['leftOperand', 'rightOperand', 'precision'] },
    { name: 'square root takes one operand', operation: Operation.Sqrt, want: ['leftOperand', 'precision'] },
  ] as const

  it.each(cases)('$name', ({ operation, want }) => {
    expect(inputsOf(operation)).toEqual(want)
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

describe('validateForm', () => {
  const cases: { name: string; operation: Operation; values: FormValues; want: FieldErrors }[] = [
    {
      name: 'valid two operands, full precision',
      operation: Operation.Add,
      values: formWith({ leftOperand: '1.5', rightOperand: '2' }),
      want: {},
    },
    {
      name: 'every problem is reported at once',
      operation: Operation.Divide,
      values: formWith({ leftOperand: '', rightOperand: 'abc', precision: '16' }),
      want: {
        leftOperand: 'error.field.required',
        rightOperand: 'error.field.invalid.type',
        precision: 'error.field.out.of.range',
      },
    },
    {
      name: 'operand beyond float64',
      operation: Operation.Multiply,
      values: formWith({ leftOperand: '1e309', rightOperand: '1' }),
      want: { leftOperand: 'error.field.out.of.range' },
    },
    {
      name: 'square root ignores the right input',
      operation: Operation.Sqrt,
      values: formWith({ leftOperand: '9', rightOperand: 'not used' }),
      want: {},
    },
    {
      name: 'square root still needs its operand',
      operation: Operation.Sqrt,
      values: formWith({}),
      want: { leftOperand: 'error.field.required' },
    },
  ]

  it.each(cases)('$name', ({ operation, values, want }) => {
    expect(validateForm(operation, values)).toEqual(want)
  })
})

describe('toOperands', () => {
  const cases = [
    {
      name: 'two operands, full precision',
      operation: Operation.Power,
      values: formWith({ leftOperand: ' 2 ', rightOperand: '10' }),
      want: { binary: true, operation: Operation.Power, left: 2, right: 10, precision: null },
    },
    {
      name: 'precision zero is kept',
      operation: Operation.Divide,
      values: formWith({ leftOperand: '1', rightOperand: '3', precision: '0' }),
      want: { binary: true, operation: Operation.Divide, left: 1, right: 3, precision: 0 },
    },
    {
      name: 'square root has no right operand',
      operation: Operation.Sqrt,
      values: formWith({ leftOperand: '2.25', rightOperand: 'ignored', precision: '2' }),
      want: { binary: false, operation: Operation.Sqrt, left: 2.25, precision: 2 },
    },
  ] as const

  it.each(cases)('$name', ({ operation, values, want }) => {
    expect(toOperands(operation, values)).toEqual(want)
  })
})

describe('firstInvalidInput', () => {
  const cases: { name: string; operation: Operation; errors: FieldErrors; want: string | null }[] = [
    { name: 'no errors', operation: Operation.Add, errors: {}, want: null },
    {
      name: 'form order, not insertion order',
      operation: Operation.Add,
      errors: { precision: 'error.field.out.of.range', rightOperand: 'error.field.required' },
      want: 'rightOperand',
    },
    {
      name: 'an input the operation does not show is skipped',
      operation: Operation.Sqrt,
      errors: { rightOperand: 'error.field.required', precision: 'error.field.invalid.type' },
      want: 'precision',
    },
  ]

  it.each(cases)('$name', ({ operation, errors, want }) => {
    expect(firstInvalidInput(operation, errors)).toBe(want)
  })
})

describe('mapApiFieldErrors', () => {
  const cases: {
    name: string
    operation: Operation
    errors: FieldError[]
    want: { fieldErrors: FieldErrors; unmatched: string | null }
  }[] = [
    {
      name: 'each field error lands on its input',
      operation: Operation.Add,
      errors: [
        { field: 'leftOperand', code: 'error.field.required' },
        { field: 'rightOperand', code: 'error.field.invalid.type' },
      ],
      want: {
        fieldErrors: { leftOperand: 'error.field.required', rightOperand: 'error.field.invalid.type' },
        unmatched: null,
      },
    },
    {
      name: 'square root operand maps to the left input',
      operation: Operation.Sqrt,
      errors: [{ field: 'operand', code: 'error.field.out.of.range' }],
      want: { fieldErrors: { leftOperand: 'error.field.out.of.range' }, unmatched: null },
    },
    {
      name: 'first code per input wins',
      operation: Operation.Add,
      errors: [
        { field: 'precision', code: 'error.field.invalid.type' },
        { field: 'precision', code: 'error.field.out.of.range' },
      ],
      want: { fieldErrors: { precision: 'error.field.invalid.type' }, unmatched: null },
    },
    {
      name: 'a field the form does not show is unmatched, first one wins',
      operation: Operation.Add,
      errors: [
        { field: 'leftOperand', code: 'error.field.required' },
        { field: 'aaa', code: 'error.field.unknown' },
        { field: 'zzz', code: 'error.field.required' },
      ],
      want: { fieldErrors: { leftOperand: 'error.field.required' }, unmatched: 'error.field.unknown' },
    },
    {
      name: 'an unmatched error before a field error does not hide it',
      operation: Operation.Add,
      errors: [
        { field: 'zzz', code: 'error.field.unknown' },
        { field: 'rightOperand', code: 'error.field.required' },
      ],
      want: { fieldErrors: { rightOperand: 'error.field.required' }, unmatched: 'error.field.unknown' },
    },
    {
      name: 'square root reports a right operand error for the whole form',
      operation: Operation.Sqrt,
      errors: [{ field: 'rightOperand', code: 'error.field.unknown' }],
      want: { fieldErrors: {}, unmatched: 'error.field.unknown' },
    },
    {
      name: 'no errors',
      operation: Operation.Divide,
      errors: [],
      want: { fieldErrors: {}, unmatched: null },
    },
  ]

  it.each(cases)('$name', ({ operation, errors, want }) => {
    expect(mapApiFieldErrors(operation, errors)).toEqual(want)
  })
})
