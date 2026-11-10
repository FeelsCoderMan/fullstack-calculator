import { useCallback, useEffect, useRef, useState } from 'react'
import { calculate, isAbortError, OutcomeKind } from '../api/client'
import { ClientErrorCode, Operation } from '../api/constants'
import { buildRequest, InputField, inputForField, inputsOf, type Operands } from '../api/operations'
import { parseOperand, parsePrecision, validateOperand, validatePrecision } from '../utils/validation'

/** Raw text of each input, exactly as typed. */
export type FormValues = Record<InputField, string>

/** Field error code per input; an input without an entry is valid. */
export type FieldErrors = Partial<Record<InputField, string>>

export const StatusKind = {
  Idle: 'idle',
  Loading: 'loading',
  Success: 'success',
  Failure: 'failure',
} as const

export type Status =
  | { kind: typeof StatusKind.Idle }
  | { kind: typeof StatusKind.Loading }
  | { kind: typeof StatusKind.Success; operation: Operation; operands: Operands; result: number }
  | { kind: typeof StatusKind.Failure; code: string; requestId: string | null }

const EMPTY_VALUES: FormValues = {
  [InputField.Left]: '',
  [InputField.Right]: '',
  [InputField.Precision]: '',
}

const IDLE: Status = { kind: StatusKind.Idle }

/** Client-side validation of the inputs `operation` uses, with the backend's field codes. */
export function validateForm(operation: Operation, values: FormValues): FieldErrors {
  const errors: FieldErrors = {}
  for (const input of inputsOf(operation)) {
    const code = input === InputField.Precision ? validatePrecision(values[input]) : validateOperand(values[input])
    if (code !== null) errors[input] = code
  }
  return errors
}

function toOperands(operation: Operation, values: FormValues): Operands {
  const usesRight = inputsOf(operation).includes(InputField.Right)
  return {
    left: parseOperand(values[InputField.Left]),
    // A one-operand operation ignores the right input, so it isn't parsed.
    right: usesRight ? parseOperand(values[InputField.Right]) : Number.NaN,
    precision: parsePrecision(values[InputField.Precision]),
  }
}

/**
 * State and actions of the calculator form: the selected operation, the typed
 * values, field errors (client-side or from the API) and the request status.
 *
 * Only the latest request counts: submitting again, or unmounting, aborts the
 * one in flight, so a slow earlier answer can never overwrite a newer one.
 */
export function useCalculator() {
  const [operation, setOperationState] = useState<Operation>(Operation.Add)
  const [values, setValues] = useState<FormValues>(EMPTY_VALUES)
  const [fieldErrors, setFieldErrors] = useState<FieldErrors>({})
  const [status, setStatus] = useState<Status>(IDLE)
  const inFlight = useRef<AbortController | null>(null)

  const cancel = useCallback(() => {
    inFlight.current?.abort()
    inFlight.current = null
  }, [])

  useEffect(() => cancel, [cancel])

  const setOperation = useCallback(
    (next: Operation) => {
      cancel()
      setOperationState(next)
      setFieldErrors({})
      setStatus(IDLE)
    },
    [cancel],
  )

  const setValue = useCallback((input: InputField, value: string) => {
    setValues((prev) => ({ ...prev, [input]: value }))
    // Editing a field clears its error; the others stay until the next submit.
    setFieldErrors((prev) => {
      if (!(input in prev)) return prev
      const next = { ...prev }
      delete next[input]
      return next
    })
  }, [])

  const reset = useCallback(() => {
    cancel()
    setValues(EMPTY_VALUES)
    setFieldErrors({})
    setStatus(IDLE)
  }, [cancel])

  /**
   * Validates and sends the calculation. Resolves to the inputs that are invalid
   * (client-side or according to the API), in form order, so the caller can move
   * focus to the first one. An empty list means there were no field errors.
   */
  const submit = useCallback(async (): Promise<InputField[]> => {
    const clientErrors = validateForm(operation, values)
    if (Object.keys(clientErrors).length > 0) {
      cancel()
      setFieldErrors(clientErrors)
      // Keep the previous result on screen (and out of the live region's way); only a
      // cancelled request must leave the loading state.
      setStatus((prev) => (prev.kind === StatusKind.Loading ? IDLE : prev))
      return invalidInputs(operation, clientErrors)
    }

    cancel()
    const controller = new AbortController()
    inFlight.current = controller
    const operands = toOperands(operation, values)
    setFieldErrors({})
    setStatus({ kind: StatusKind.Loading })

    let outcome
    try {
      outcome = await calculate(operation, buildRequest(operation, operands), controller.signal)
    } catch (err) {
      if (isAbortError(err)) return []
      outcome = { kind: OutcomeKind.Failure, code: ClientErrorCode.UnexpectedResponse, requestId: null } as const
    }
    if (controller.signal.aborted) return []
    inFlight.current = null

    switch (outcome.kind) {
      case OutcomeKind.Success:
        setStatus({ kind: StatusKind.Success, operation, operands, result: outcome.result })
        return []
      case OutcomeKind.FieldErrors: {
        const apiErrors: FieldErrors = {}
        let unmatched: string | null = null
        for (const { field, code } of outcome.errors) {
          const input = inputForField(operation, field)
          if (input === undefined) unmatched ??= code
          else apiErrors[input] ??= code
        }
        setFieldErrors(apiErrors)
        // An error for a field the form doesn't show (e.g. error.field.unknown) can
        // only be reported for the whole form.
        setStatus(
          unmatched === null ? IDLE : { kind: StatusKind.Failure, code: unmatched, requestId: outcome.requestId },
        )
        return invalidInputs(operation, apiErrors)
      }
      case OutcomeKind.Failure:
        setStatus({ kind: StatusKind.Failure, code: outcome.code, requestId: outcome.requestId })
        return []
    }
  }, [cancel, operation, values])

  return { operation, values, fieldErrors, status, setOperation, setValue, reset, submit }
}

function invalidInputs(operation: Operation, errors: FieldErrors): InputField[] {
  return inputsOf(operation).filter((input) => errors[input] !== undefined)
}
