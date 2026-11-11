import { SWAGGER_PATH } from '@/api/constants'
import styles from '@/App.module.css'
import { Calculator } from '@/components/calculator/Calculator'
import { t } from '@/i18n'

export default function App() {
  return (
    <div className={styles.page}>
      <header className={styles.header}>
        <h1 className={styles.title}>{t.app.title}</h1>
        <p className={styles.tagline}>{t.app.tagline}</p>
      </header>
      <main className={styles.main}>
        <Calculator />
      </main>
      <footer className={styles.footer}>
        <a className={styles.link} href={SWAGGER_PATH}>
          {t.app.apiDocs}
        </a>
      </footer>
    </div>
  )
}
