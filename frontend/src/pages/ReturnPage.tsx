import { useEffect, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'

export default function ReturnPage() {
  const [params] = useSearchParams()
  const [status, setStatus] = useState('processing')
  useEffect(() => { setStatus(params.get('redirect_status') ?? 'processing') }, [params])
  const success = status === 'succeeded'
  return <section className="return-page card"><span className={`result-icon ${success ? 'ok' : ''}`}>{success ? '✓' : '…'}</span><h1>{success ? 'Payment received' : 'Payment processing'}</h1><p>{success ? 'Thanks — your payment was successful and a receipt is on its way.' : 'Your payment is being confirmed. The report will update when Stripe sends the result.'}</p><div className="return-actions"><Link className="primary-button link-button" to="/report">View payment report</Link><Link className="text-link" to="/">Make another payment</Link></div></section>
}

