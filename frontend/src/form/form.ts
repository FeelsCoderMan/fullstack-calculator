import { type Operation, RequestField } from '@/api/constants'
import { isBinary, type Operands } from '@/api/operations'
import type { FieldError } from '@/api/types'
import {
  FULL_PRECISION,
  parseOperand,
  parsePrecision,
  validateOperand,
  validatePrecision,
} from '@/utils/validation'

// The calculator form as data: its inputs, their raw values and errors, and the pure
// rules that turn them into operands or map API field errors back onto them. React
// state lives in `useCalculator`; this module has no React in it, so it is unit-tested
// directly.

/** The form's inputs. Named after the binary request fields they fill. */
export const InputField = {
  Left: RequestField.LeftOperand,
  Right: RequestField.RightOperand,
  Precision: RequestField.Precision,
} as const
export type InputField = (typeof InputField)[keyof typeof InputField]

/** Raw text of each input, exactly as typed. */
export type FormValues = Record<InputField, string>

/** Field error code per input; an input without an entry is valid. */
export type FieldErrors = Partial<Record<InputField, string>>

export const EMPTY_VALUES: FormValues = {
  [InputField.Left]: '',
  [InputField.Right]: '',
  [InputField.Precision]: FULL_PRECISION,
}

/** Inputs the form shows for an operation, in tab order. */
export function inputsOf(operation: Operation): InputField[] {
  return isBinary(operation)
    ? [InputField.Left, InputField.Right, InputField.Precision]
    : [InputField.Left, InputField.Precision]
}

/**
 * Maps a request field reported by the API back to the form input that produced it,
 * or `undefined` when no input matches (e.g. an `error.field.unknown` property).
 */
export function inputForField(operation: Operation, field: string): InputField | undefined {
  if (!isBinary(operation)) {
    // The single operand is sent as `operand` but typed into the left input.
    if (field === RequestField.Operand) return InputField.Left
    return field === RequestField.Precision ? InputField.Precision : undefined
  }
  return inputsOf(operation).find((input) => input === field)
}

/** Client-side validation of the inputs `operation` uses, with the backend's field codes. */
export function validateForm(operation: Operation, values: FormValues): FieldErrors {
  const errors: FieldErrors = {}
  for (const input of inputsOf(operation)) {
    const code = input === InputField.Precision ? validatePrecision(values[input]) : validateOperand(values[input])
    if (code !== null) errors[input] = code
  }
  return errors
}

/** Parses values that passed `validateForm`. A one-operand operation ignores the right input. */
export function toOperands(operation: Operation, values: FormValues): Operands {
  const left = parseOperand(values[InputField.Left])
  const precision = parsePrecision(values[InputField.Precision])
  if (isBinary(operation)) {
    return { binary: true, operation, left, right: parseOperand(values[InputField.Right]), precision }
  }
  return { binary: false, operation, left, precision }
}

/** The first input in form order that has an error, or `null`. */
export function firstInvalidInput(operation: Operation, errors: FieldErrors): InputField | null {
  return inputsOf(operation).find((input) => errors[input] !== undefined) ?? null
}

/**
 * Maps the field errors of a 422 response onto the form. The first code per input
 * wins. An error for a field the form doesn't show (e.g. `error.field.unknown`) can
 * only be reported for the whole form: the first such code is returned as `unmatched`.
 */
export function mapApiFieldErrors(
  operation: Operation,
  errors: readonly FieldError[],
): { fieldErrors: FieldErrors; unmatched: string | null } {
  const fieldErrors: FieldErrors = {}
  let unmatched: string | null = null
  for (const { field, code } of errors) {
    const input = inputForField(operation, field)
    if (input === undefined) unmatched ??= code
    else fieldErrors[input] ??= code
  }
  return { fieldErrors, unmatched }
}
