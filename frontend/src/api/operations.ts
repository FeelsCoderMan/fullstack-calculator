import { Operation, RequestField } from './constants'
import type { OperationRequest } from './types'

/** Number of operands an operation takes. */
export const Arity = {
  Unary: 1,
  Binary: 2,
} as const
export type Arity = (typeof Arity)[keyof typeof Arity]

/** The form's inputs. Named after the binary request fields they fill. */
export const InputField = {
  Left: RequestField.LeftOperand,
  Right: RequestField.RightOperand,
  Precision: RequestField.Precision,
} as const
export type InputField = (typeof InputField)[keyof typeof InputField]

export function arityOf(operation: Operation): Arity {
  return operation === Operation.Sqrt ? Arity.Unary : Arity.Binary
}

/** Inputs the form shows for an operation, in tab order. */
export function inputsOf(operation: Operation): InputField[] {
  return arityOf(operation) === Arity.Unary
    ? [InputField.Left, InputField.Precision]
    : [InputField.Left, InputField.Right, InputField.Precision]
}

/** Parsed, client-validated form values. */
export interface Operands {
  left: number
  /** `NaN` for one-operand operations, which never send it. */
  right: number
  /** `null` = full precision (the field is omitted from the request). */
  precision: number | null
}

/** Builds the request body for an operation. `precision` is omitted when not set. */
export function buildRequest(operation: Operation, { left, right, precision }: Operands): OperationRequest {
  const rounding = precision === null ? {} : { precision }
  if (arityOf(operation) === Arity.Unary) {
    return { operand: left, ...rounding }
  }
  return { leftOperand: left, rightOperand: right, ...rounding }
}

/**
 * Maps a request field reported by the API back to the form input that produced it,
 * or `undefined` when no input matches (e.g. an `error.field.unknown` property).
 */
export function inputForField(operation: Operation, field: string): InputField | undefined {
  if (arityOf(operation) === Arity.Unary) {
    // The single operand is sent as `operand` but typed into the left input.
    if (field === RequestField.Operand) return InputField.Left
    return field === RequestField.Precision ? InputField.Precision : undefined
  }
  return inputsOf(operation).find((input) => input === field)
}
