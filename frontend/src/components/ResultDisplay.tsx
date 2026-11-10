import { useId } from 'react'
import { arityOf, Arity } from '../api/operations'
import { StatusKind, type Status } from '../hooks/useCalculator'
import { errorMessage, formatNumber, operationText, t } from '../i18n'
import styles from './ResultDisplay.module.css'

interface ResultDisplayProps {
  status: Status
}

/**
 * The calculator's display. Its content is a live region, so screen readers
 * announce each result or error without the user moving focus.
 */
export function ResultDisplay({ status }: ResultDisplayProps) {
  const headingId = useId()
  const loading = status.kind === StatusKind.Loading

  return (
    <section className={styles.display} aria-labelledby={headingId}>
      <h2 id={headingId} className={styles.heading}>
        {t.result.heading}
      </h2>
      {/* <output> has the implicit role "status"; aria-live makes the announcement explicit. */}
      <output className={styles.content} aria-live="polite" aria-atomic="true" aria-busy={loading}>
        <StatusContent status={status} />
      </output>
    </section>
  )
}

function StatusContent({ status }: ResultDisplayProps) {
  switch (status.kind) {
    case StatusKind.Idle:
      return <span className={styles.placeholder}>{t.result.idle}</span>
    case StatusKind.Loading:
      return (
        <span className={styles.placeholder}>
          <span className={styles.spinner} aria-hidden="true" />
          {t.result.loading}
        </span>
      )
    case StatusKind.Success: {
      const { operation, operands, result } = status
      const right = arityOf(operation) === Arity.Binary ? formatNumber(operands.right) : undefined
      return (
        <>
          <span className={styles.expression}>
            {operationText(operation).expression(formatNumber(operands.left), right)} {t.result.equals}
          </span>
          <span className={styles.value}>{formatNumber(result)}</span>
        </>
      )
    }
    case StatusKind.Failure:
      return (
        <>
          <span className={styles.errorHeading}>{t.result.errorHeading}</span>
          <span className={styles.error}>{errorMessage(status.code)}</span>
          {status.requestId && <span className={styles.reference}>{t.result.requestId(status.requestId)}</span>}
        </>
      )
  }
}
