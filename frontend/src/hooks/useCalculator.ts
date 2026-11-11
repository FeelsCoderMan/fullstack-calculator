import { useCallback, useEffect, useRef, useState } from 'react'
import { calculate, isAbortError, OutcomeKind } from '@/api/client'
import { ClientErrorCode, Operation } from '@/api/constants'
import { Arity, arityOf, buildRequest, InputField, inputForField, inputsOf, type Operands } from '@/api/operations'
import { formatOperand, parseOperand, parsePrecision, validateOperand, validatePrecision } from '@/utils/validation'

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
      // Keep the last result visible while the next operation is chosen; only a
      // cancelled request must leave the loading state.
      setStatus((prev) => (prev.kind === StatusKind.Loading ? IDLE : prev))
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
   * Validates and sends the calculation (the "=" key).
   *
   * On success the result carries over into the first number, like on a pocket
   * calculator, so the next calculation continues from it: `15 + 12 = 27`, then
   * `27 × 2 = 54`. The display keeps showing the finished calculation.
   *
   * Resolves to the input that should get focus next: the first invalid one after
   * a field error, the input for the next operand after a success, or `null`.
   */
  const submit = useCallback(async (): Promise<InputField | null> => {
    const clientErrors = validateForm(operation, values)
    if (Object.keys(clientErrors).length > 0) {
      cancel()
      setFieldErrors(clientErrors)
      // Keep the previous result on screen (and out of the live region's way); only a
      // cancelled request must leave the loading state.
      setStatus((prev) => (prev.kind === StatusKind.Loading ? IDLE : prev))
      return firstInvalidInput(operation, clientErrors)
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
      if (isAbortError(err)) return null
      outcome = { kind: OutcomeKind.Failure, code: ClientErrorCode.UnexpectedResponse, requestId: null } as const
    }
    if (controller.signal.aborted) return null
    inFlight.current = null

    switch (outcome.kind) {
      case OutcomeKind.Success: {
        const { result } = outcome
        setStatus({ kind: StatusKind.Success, operation, operands, result })
        // Carry the result into the first number and clear the second, ready for the
        // next operand. Precision stays as chosen.
        setValues((prev) => ({
          ...prev,
          [InputField.Left]: formatOperand(result),
          [InputField.Right]: EMPTY_VALUES[InputField.Right],
        }))
        return arityOf(operation) === Arity.Binary ? InputField.Right : InputField.Left
      }
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
        return firstInvalidInput(operation, apiErrors)
      }
      case OutcomeKind.Failure:
        setStatus({ kind: StatusKind.Failure, code: outcome.code, requestId: outcome.requestId })
        return null
    }
  }, [cancel, operation, values])

  return { operation, values, fieldErrors, status, setOperation, setValue, reset, submit }
}

/** The first input in form order that has an error, or `null`. */
function firstInvalidInput(operation: Operation, errors: FieldErrors): InputField | null {
  return inputsOf(operation).find((input) => errors[input] !== undefined) ?? null
}
