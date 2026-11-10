import { FieldErrorCode, PRECISION_MAX, PRECISION_MIN } from '../api/constants'
import type { FieldErrorCode as FieldErrorCodeValue } from '../api/types'

/**
 * A plain decimal number, optionally signed and in exponent form: `12`, `-3.5`,
 * `.5`, `4.`, `1e-3`. Stricter than `Number()`, which also accepts `0x10`,
 * `Infinity` and whitespace-only input.
 */
const NUMBER_PATTERN = /^[+-]?(?:\d+\.?\d*|\.\d+)(?:e[+-]?\d+)?$/i
const INTEGER_PATTERN = /^[+-]?\d+$/

/**
 * Validates an operand typed by the user with the same field codes the backend uses.
 * Returns `null` when the input is a valid number.
 */
export function validateOperand(raw: string): FieldErrorCodeValue | null {
  const input = raw.trim()
  if (input === '') return FieldErrorCode.Required
  if (!NUMBER_PATTERN.test(input)) return FieldErrorCode.InvalidType
  // Beyond float64 (about ±1.797e308), like the backend's error.field.out.of.range.
  if (!Number.isFinite(Number(input))) return FieldErrorCode.OutOfRange
  return null
}

/** Parses an operand that passed `validateOperand`. */
export function parseOperand(raw: string): number {
  return Number(raw.trim())
}

/**
 * Validates the optional precision (decimal places). Empty means "full precision"
 * and is valid. Returns `null` when valid.
 */
export function validatePrecision(raw: string): FieldErrorCodeValue | null {
  const input = raw.trim()
  if (input === '') return null
  if (!INTEGER_PATTERN.test(input)) return FieldErrorCode.InvalidType
  const places = Number(input)
  if (places < PRECISION_MIN || places > PRECISION_MAX) return FieldErrorCode.OutOfRange
  return null
}

/** Parses a precision that passed `validatePrecision`; `null` = full precision. */
export function parsePrecision(raw: string): number | null {
  const input = raw.trim()
  return input === '' ? null : Number(input)
}
