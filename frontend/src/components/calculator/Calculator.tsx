import { useRef, type FormEvent, type KeyboardEvent } from 'react'
import { PRECISION_MAX, PRECISION_MIN } from '@/api/constants'
import { Arity, arityOf, InputField } from '@/api/operations'
import { StatusKind, useCalculator } from '@/hooks/useCalculator'
import { operationText, t } from '@/i18n'
import { cx } from '@/utils/dom'
import styles from '@/components/calculator/Calculator.module.css'
import { Field } from '@/components/field/Field'
import { OperationPicker } from '@/components/operation-picker/OperationPicker'
import { ResultDisplay } from '@/components/result-display/ResultDisplay'

/** `KeyboardEvent.key` of the "=" key. */
const EQUALS_KEY = '='

/** Select value meaning "no rounding": `precision` is left out of the request. */
const FULL_PRECISION = ''

const PRECISION_OPTIONS = Array.from({ length: PRECISION_MAX - PRECISION_MIN + 1 }, (_, i) => PRECISION_MIN + i)

type Control = HTMLInputElement | HTMLSelectElement

/** The calculator: a display, the operation picker, operand inputs and actions. */
export function Calculator() {
  const { operation, values, fieldErrors, status, setOperation, setValue, reset, submit } = useCalculator()
  const controls = useRef<Partial<Record<InputField, Control | null>>>({})
  const text = operationText(operation)
  const binary = arityOf(operation) === Arity.Binary
  const loading = status.kind === StatusKind.Loading

  const bind = (input: InputField) => (element: Control | null) => {
    controls.current[input] = element
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const next = await submit()
    if (next !== null) controls.current[next]?.focus()
  }

  // "=" is never part of a number, so in an operand it means "calculate", like Enter.
  function handleOperandKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    if (event.key !== EQUALS_KEY) return
    event.preventDefault()
    event.currentTarget.form?.requestSubmit()
  }

  function handleClear() {
    reset()
    controls.current[InputField.Left]?.focus()
  }

  function operandField(input: InputField, label: string) {
    return (
      <Field label={label} error={fieldErrors[input]}>
        {(control) => (
          <input
            {...control}
            ref={bind(input)}
            type="text"
            autoComplete="off"
            autoCorrect="off"
            spellCheck={false}
            enterKeyHint="go"
            placeholder={t.form.operandPlaceholder}
            value={values[input]}
            onChange={(event) => setValue(input, event.target.value)}
            onKeyDown={handleOperandKeyDown}
          />
        )}
      </Field>
    )
  }

  return (
    <div className={styles.card}>
      <ResultDisplay status={status} />
      <form className={styles.form} onSubmit={handleSubmit} noValidate>
        <OperationPicker value={operation} onChange={setOperation} />

        <div className={cx(styles.operands, binary && styles.binary)}>
          {operandField(InputField.Left, text.leftLabel)}
          {binary && (
            <>
              <span className={styles.symbol} aria-hidden="true">
                {text.symbol}
              </span>
              {operandField(InputField.Right, text.rightLabel ?? text.leftLabel)}
            </>
          )}
        </div>

        <div className={styles.footer}>
          <div className={styles.precision}>
            <Field label={t.form.precisionLabel} hint={t.form.precisionHint} error={fieldErrors[InputField.Precision]}>
              {(control) => (
                <select
                  {...control}
                  ref={bind(InputField.Precision)}
                  value={values[InputField.Precision]}
                  onChange={(event) => setValue(InputField.Precision, event.target.value)}
                >
                  <option value={FULL_PRECISION}>{t.form.precisionFull}</option>
                  {PRECISION_OPTIONS.map((places) => (
                    <option key={places} value={places.toString()}>
                      {t.form.precisionOption(places)}
                    </option>
                  ))}
                </select>
              )}
            </Field>
          </div>

          <div className={styles.actions}>
            <button type="button" className={cx(styles.button, styles.secondary)} onClick={handleClear}>
              {t.form.clear}
            </button>
            {/* Shows "=" like a calculator key; named "Calculate" for assistive tech. */}
            <button
              type="submit"
              className={cx(styles.button, styles.primary, styles.equals)}
              aria-label={t.form.calculate}
              title={t.form.calculate}
              aria-busy={loading}
            >
              {t.form.equals}
            </button>
          </div>
        </div>
      </form>
    </div>
  )
}
