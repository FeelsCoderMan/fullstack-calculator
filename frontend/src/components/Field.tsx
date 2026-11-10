import { useId, type ReactNode } from 'react'
import { fieldErrorMessage } from '../i18n'
import { errorId, hintId, idRefs } from '../utils/dom'
import styles from './Field.module.css'

/** Accessibility props a field passes to its control. */
export interface ControlProps {
  id: string
  className: string | undefined
  'aria-invalid': true | undefined
  'aria-describedby': string | undefined
}

interface FieldProps {
  label: string
  /** Field error code; its message is rendered under the control. */
  error?: string
  hint?: string
  /** Renders the control with the ids that tie it to its label, hint and error. */
  children: (control: ControlProps) => ReactNode
}

/** A labelled form control with an optional hint and an error message linked to it. */
export function Field({ label, error, hint, children }: FieldProps) {
  const id = useId()
  const hintElementId = hintId(id)
  const errorElementId = errorId(id)

  return (
    <div className={styles.field}>
      <label className={styles.label} htmlFor={id}>
        {label}
      </label>
      {children({
        id,
        className: styles.control,
        'aria-invalid': error ? true : undefined,
        'aria-describedby': idRefs(hint && hintElementId, error && errorElementId),
      })}
      {hint && (
        <p id={hintElementId} className={styles.hint}>
          {hint}
        </p>
      )}
      {error && (
        <p id={errorElementId} className={styles.error}>
          {fieldErrorMessage(error)}
        </p>
      )}
    </div>
  )
}
