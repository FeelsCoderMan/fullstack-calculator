import type { Operation } from '../api/constants'
import { en, type OperationText } from './en'

/** The active messages. Only English exists; a second locale would be chosen here. */
export const t = en

export function operationText(operation: Operation): OperationText {
  return t.operations[operation]
}

function lookup(table: Record<string, string>, code: string): string | undefined {
  return Object.hasOwn(table, code) ? table[code] : undefined
}

/** Message for a top-level error code, falling back to a generic text for unknown codes. */
export function errorMessage(code: string): string {
  return lookup(t.errors, code) ?? lookup(t.fieldErrors, code) ?? t.genericError
}

/** Message for a field error code, shown next to its input. */
export function fieldErrorMessage(code: string): string {
  return lookup(t.fieldErrors, code) ?? t.genericError
}

/** Formats a number for display: the shortest form that round-trips, as JS prints it. */
export function formatNumber(value: number): string {
  return value.toString()
}
