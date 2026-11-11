// Aliases for the types generated from backend/api/openapi.yaml (npm run gen:api).
// The spec is the source of truth: never hand-write a type it already defines.
import type { components } from '@/api/schema.gen'

type Schemas = components['schemas']

export type ErrorCode = Schemas['ErrorCode']
export type FieldErrorCode = Schemas['FieldErrorCode']
export type FieldError = Schemas['FieldError']
export type BinaryOperationRequest = Schemas['BinaryOperationRequest']
/** Body of a one-operand operation (`sqrt`): `{operand, precision?}`. */
export type UnaryOperationRequest = Schemas['UnaryOperationRequest']
export type ResultResponse = Schemas['ResultResponse']
export type ErrorResponse = Schemas['ErrorResponse']

export type OperationRequest = BinaryOperationRequest | UnaryOperationRequest
