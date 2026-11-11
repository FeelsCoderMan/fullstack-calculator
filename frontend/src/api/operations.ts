import { Operation } from '@/api/constants'
import type { OperationRequest } from '@/api/types'

/**
 * Operations that take one operand (`{operand, precision?}`); every other operation
 * takes two (`{leftOperand, rightOperand, precision?}`). The single source of an
 * operation's arity: the types below, `isBinary` and the per-arity operation texts
 * in `src/i18n` all derive from it.
 */
export const UNARY_OPERATIONS = [Operation.Sqrt] as const satisfies readonly Operation[]

export type UnaryOperation = (typeof UNARY_OPERATIONS)[number]
export type BinaryOperation = Exclude<Operation, UnaryOperation>

/** Whether `operation` takes two operands. */
export function isBinary(operation: Operation): operation is BinaryOperation {
  return !(UNARY_OPERATIONS as readonly Operation[]).includes(operation)
}

/**
 * Parsed, client-validated operands of one calculation. A one-operand operation has
 * no `right` at all, so code that reads it must check `binary` first.
 */
export type Operands =
  | { binary: true; operation: BinaryOperation; left: number; right: number; precision: number | null }
  | { binary: false; operation: UnaryOperation; left: number; precision: number | null }

/** Builds the request body. `precision` is omitted when `null` (full precision). */
export function buildRequest(operands: Operands): OperationRequest {
  const rounding = operands.precision === null ? {} : { precision: operands.precision }
  if (!operands.binary) {
    return { operand: operands.left, ...rounding }
  }
  return { leftOperand: operands.left, rightOperand: operands.right, ...rounding }
}
