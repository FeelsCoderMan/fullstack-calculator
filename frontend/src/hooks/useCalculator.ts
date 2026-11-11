import { useCallback, useEffect, useRef, useState } from 'react'
import { calculate, type CalculationOutcome, failure, isAbortError, OutcomeKind } from '@/api/client'
import { ClientErrorCode, Operation } from '@/api/constants'
import { buildRequest } from '@/api/operations'
import {
  EMPTY_VALUES,
  type FieldErrors,
  firstInvalidInput,
  type FormValues,
  InputField,
  mapApiFieldErrors,
  toOperands,
  validateForm,
} from '@/form/form'
import { IDLE, type Status, StatusKind } from '@/form/status'
import { formatOperand } from '@/utils/validation'

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

  // Keeps the last result on screen (and out of the live region's way); only a
  // cancelled request must leave the loading state.
  const leaveLoading = useCallback(() => {
    setStatus((prev) => (prev.kind === StatusKind.Loading ? IDLE : prev))
  }, [])

  const setOperation = useCallback(
    (next: Operation) => {
      cancel()
      setOperationState(next)
      setFieldErrors({})
      leaveLoading()
    },
    [cancel, leaveLoading],
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
    // A new submit always replaces the request in flight, valid or not.
    cancel()
    const clientErrors = validateForm(operation, values)
    if (Object.keys(clientErrors).length > 0) {
      setFieldErrors(clientErrors)
      leaveLoading()
      return firstInvalidInput(operation, clientErrors)
    }

    const controller = new AbortController()
    inFlight.current = controller
    const operands = toOperands(operation, values)
    setFieldErrors({})
    setStatus({ kind: StatusKind.Loading })

    let outcome: CalculationOutcome
    try {
      outcome = await calculate(operation, buildRequest(operands), controller.signal)
    } catch (err) {
      if (isAbortError(err)) return null
      outcome = failure(ClientErrorCode.UnexpectedResponse, null)
    }
    if (controller.signal.aborted) return null
    inFlight.current = null

    switch (outcome.kind) {
      case OutcomeKind.Success: {
        const { result } = outcome
        setStatus({ kind: StatusKind.Success, operands, result })
        // Carry the result into the first number and clear the second, ready for the
        // next operand. Precision stays as chosen.
        setValues((prev) => ({
          ...prev,
          [InputField.Left]: formatOperand(result),
          [InputField.Right]: EMPTY_VALUES[InputField.Right],
        }))
        return operands.binary ? InputField.Right : InputField.Left
      }
      case OutcomeKind.FieldErrors: {
        const { fieldErrors: apiErrors, unmatched } = mapApiFieldErrors(operation, outcome.errors)
        setFieldErrors(apiErrors)
        // A field the form doesn't show can only be reported for the whole form.
        setStatus(
          unmatched === null ? IDLE : { kind: StatusKind.Failure, code: unmatched, requestId: outcome.requestId },
        )
        return firstInvalidInput(operation, apiErrors)
      }
      case OutcomeKind.Failure:
        setStatus({ kind: StatusKind.Failure, code: outcome.code, requestId: outcome.requestId })
        return null
    }
  }, [cancel, leaveLoading, operation, values])

  return { operation, values, fieldErrors, status, setOperation, setValue, reset, submit }
}
