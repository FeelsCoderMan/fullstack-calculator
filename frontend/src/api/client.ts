import {
  ABORT_ERROR_NAME,
  ApiPath,
  ClientErrorCode,
  Header,
  HttpMethod,
  MediaType,
  type Operation,
} from '@/api/constants'
import type { ErrorResponse, FieldError, OperationRequest, ResultResponse } from '@/api/types'

/** What a calculation request ended with. Components never see raw responses. */
export const OutcomeKind = {
  Success: 'success',
  FieldErrors: 'fieldErrors',
  Failure: 'failure',
} as const

export type CalculationOutcome =
  | { kind: typeof OutcomeKind.Success; result: number }
  | { kind: typeof OutcomeKind.FieldErrors; errors: FieldError[]; requestId: string | null }
  | {
      kind: typeof OutcomeKind.Failure
      /** An API `ErrorCode`, a `ClientErrorCode`, or a code this client doesn't know yet. */
      code: string
      /** `X-Request-ID` of the response, to quote when reporting a problem. */
      requestId: string | null
    }

/**
 * Sends one calculation to the backend and classifies the answer.
 *
 * It never throws for HTTP or network problems: those come back as a `Failure`
 * outcome with a code the UI can translate. It only rethrows an `AbortError`
 * when `signal` cancels the request, so callers can ignore superseded requests.
 */
export async function calculate(
  operation: Operation,
  body: OperationRequest,
  signal?: AbortSignal,
): Promise<CalculationOutcome> {
  let response: Response
  try {
    response = await fetch(ApiPath[operation], {
      method: HttpMethod.Post,
      headers: { [Header.ContentType]: MediaType.JSON, [Header.Accept]: MediaType.JSON },
      body: JSON.stringify(body),
      signal,
    })
  } catch (err) {
    rethrowAbort(err)
    return failure(ClientErrorCode.Network, null)
  }

  const requestId = response.headers.get(Header.RequestID)
  let payload: unknown
  try {
    payload = await response.json()
  } catch (err) {
    rethrowAbort(err)
    // Not JSON: typically a proxy page (502/503/504) while the backend is down.
    const code = isServerError(response.status) ? ClientErrorCode.Unavailable : ClientErrorCode.UnexpectedResponse
    return failure(code, requestId)
  }

  if (response.ok) {
    return isResultResponse(payload)
      ? { kind: OutcomeKind.Success, result: payload.result }
      : failure(ClientErrorCode.UnexpectedResponse, requestId)
  }
  // A 422 is either a bare array of field errors or a {code} object.
  if (isFieldErrorList(payload)) {
    return { kind: OutcomeKind.FieldErrors, errors: payload, requestId }
  }
  if (isErrorResponse(payload)) {
    return failure(payload.code, requestId)
  }
  return failure(ClientErrorCode.UnexpectedResponse, requestId)
}

/** A failed outcome with `code`, as `calculate` returns it. */
export function failure(code: string, requestId: string | null): CalculationOutcome {
  return { kind: OutcomeKind.Failure, code, requestId }
}

/** True when `err` is the rejection of a request cancelled with an `AbortSignal`. */
export function isAbortError(err: unknown): boolean {
  return err instanceof DOMException && err.name === ABORT_ERROR_NAME
}

function rethrowAbort(err: unknown): void {
  if (isAbortError(err)) throw err
}

const SERVER_ERROR_MIN = 500

function isServerError(status: number): boolean {
  return status >= SERVER_ERROR_MIN
}

function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function isResultResponse(value: unknown): value is ResultResponse {
  return isObject(value) && typeof value.result === 'number' && Number.isFinite(value.result)
}

// The code is only checked to be a string: an unknown code still gets the
// generic message instead of being treated as a broken response.
function isErrorResponse(value: unknown): value is ErrorResponse {
  return isObject(value) && typeof value.code === 'string'
}

function isFieldErrorList(value: unknown): value is FieldError[] {
  return (
    Array.isArray(value) &&
    value.length > 0 &&
    value.every((entry) => isObject(entry) && typeof entry.field === 'string' && typeof entry.code === 'string')
  )
}
