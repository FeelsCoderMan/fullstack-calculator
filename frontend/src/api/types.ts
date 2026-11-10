// Aliases for the types generated from backend/api/openapi.yaml (npm run gen:api).
// The spec is the source of truth: never hand-write a type it already defines.
import type { components } from './schema.gen'

type Schemas = components['schemas']

export type ErrorCode = Schemas['ErrorCode']
export type FieldErrorCode = Schemas['FieldErrorCode']
export type FieldError = Schemas['FieldError']
export type BinaryOperationRequest = Schemas['BinaryOperationRequest']
export type ResultResponse = Schemas['ResultResponse']
export type ErrorResponse = Schemas['ErrorResponse']

/**
 * Request body of a one-operand operation (`sqrt`).
 *
 * Provisional: the backend has not designed sqrt's request yet, so this shape is
 * not in the spec. Replace it with the generated type once the spec defines it.
 */
export interface UnaryOperationRequest {
  operand: number
  precision?: number | null
}

export type OperationRequest = BinaryOperationRequest | UnaryOperationRequest
