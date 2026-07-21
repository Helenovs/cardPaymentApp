import { useEffect, useMemo, useState } from 'react'
import { getPayments, type Payment } from '../api'

const finalStatuses = new Set(['succeeded', 'canceled'])

function badge(status: string) { return status.replaceAll('_', ' ') }

export default function ReportPage() {
  const [payments, setPayments] = useState<Payment[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [updated, setUpdated] = useState<Date | null>(null)

  async function refresh(silent = false) {
    if (!silent) setLoading(true)
    try { const result = await getPayments(); setPayments(result.payments); setUpdated(new Date()); setError('') }
    catch (err) { setError(err instanceof Error ? err.message : 'Could not load payments') }
    finally { setLoading(false) }
  }

  useEffect(() => { void refresh(); const timer = window.setInterval(() => void refresh(true), 5000); return () => window.clearInterval(timer) }, [])
  const stats = useMemo(() => ({ total: payments.length, succeeded: payments.filter((p) => p.status === 'succeeded').length, pending: payments.filter((p) => !finalStatuses.has(p.status) && p.status !== 'requires_payment_method').length, failed: payments.filter((p) => p.status === 'requires_payment_method' && p.lastError).length }), [payments])

  return (
    <section className="report-page">
      <div className="report-title"><div><p className="eyebrow">REDIS REPORT</p><h1>Payment activity</h1><p>Webhook-backed status from your processed Stripe payments.</p></div><button className="secondary-button" onClick={() => void refresh()} disabled={loading}>↻ Refresh</button></div>
      <div className="stats-grid">
        <div className="stat"><small>TOTAL</small><strong>{stats.total}</strong></div>
        <div className="stat success"><small>SUCCEEDED</small><strong>{stats.succeeded}</strong></div>
        <div className="stat pending"><small>PROCESSING</small><strong>{stats.pending}</strong></div>
        <div className="stat failed"><small>FAILED</small><strong>{stats.failed}</strong></div>
      </div>
      <div className="card report-card">
        <div className="table-meta"><strong>Recent payments</strong><span>{updated ? `Updated ${updated.toLocaleTimeString()}` : 'Loading…'}</span></div>
        {error ? <div className="alert error">{error}</div> : payments.length === 0 && !loading ? <div className="empty"><span>◇</span><h2>No payments yet</h2><p>Completed and attempted payments will appear here.</p></div> : (
          <div className="table-wrap"><table><thead><tr><th>Payment</th><th>Customer</th><th>Date</th><th>Amount</th><th>Status</th></tr></thead><tbody>{payments.map((payment) => <tr key={payment.id}><td><code>{payment.id}</code>{payment.lastError && <small className="row-error">{payment.lastError}</small>}</td><td>{payment.customerEmail || '—'}</td><td>{new Date(payment.createdAt).toLocaleString()}</td><td>{new Intl.NumberFormat('en-AU', { style: 'currency', currency: payment.currency }).format(payment.amount / 100)}</td><td><span className={`status ${payment.status}`}>{badge(payment.status)}</span></td></tr>)}</tbody></table></div>
        )}
      </div>
    </section>
  )
}

