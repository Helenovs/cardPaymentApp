import { useEffect, useState } from 'react'
import { Elements, PaymentElement, useElements, useStripe } from '@stripe/react-stripe-js'
import { loadStripe, type Stripe } from '@stripe/stripe-js'
import { createPaymentIntent, getConfig, type AppConfig } from '../api'

const appearance = {
  theme: 'stripe' as const,
  variables: { colorPrimary: '#6759e8', colorText: '#202236', borderRadius: '10px', fontFamily: 'Inter, system-ui, sans-serif' },
}

function PaymentForm({ email, amount, currency }: { email: string; amount: number; currency: string }) {
  const stripe = useStripe()
  const elements = useElements()
  const [submitting, setSubmitting] = useState(false)
  const [message, setMessage] = useState('')
  const price = new Intl.NumberFormat('en-AU', { style: 'currency', currency }).format(amount / 100)

  async function submit(event: React.FormEvent) {
    event.preventDefault()
    if (!stripe || !elements) return
    setSubmitting(true); setMessage('')
    const { error } = await stripe.confirmPayment({
      elements,
      confirmParams: { return_url: `${window.location.origin}/payment-complete`, receipt_email: email },
    })
    if (error) setMessage(error.message ?? 'Payment could not be completed.')
    setSubmitting(false)
  }

  return (
    <form onSubmit={submit} className="payment-form">
      <PaymentElement options={{ layout: 'tabs' }} />
      {message && <div className="alert error" role="alert">{message}</div>}
      <button className="primary-button" disabled={!stripe || submitting}>{submitting ? 'Processing…' : `Pay ${price}`}</button>
      <p className="security-note"><span>⌁</span> Encrypted and secure payment</p>
    </form>
  )
}

export default function CheckoutPage() {
  const [config, setConfig] = useState<AppConfig | null>(null)
  const [stripePromise, setStripePromise] = useState<PromiseLike<Stripe | null> | null>(null)
  const [email, setEmail] = useState('')
  const [clientSecret, setClientSecret] = useState('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => { getConfig().then((value) => { setConfig(value); setStripePromise(loadStripe(value.publishableKey)) }).catch((err) => setError(err.message)).finally(() => setLoading(false)) }, [])

  async function continueToPayment(event: React.FormEvent) {
    event.preventDefault(); setLoading(true); setError('')
    try { setClientSecret((await createPaymentIntent(email)).clientSecret) }
    catch (err) { setError(err instanceof Error ? err.message : 'Could not start payment') }
    finally { setLoading(false) }
  }

  return (
    <section className="checkout-layout">
      <div className="checkout-copy">
        <p className="eyebrow">CHECKOUT</p>
        <h1>A simpler way<br />to <em>pay.</em></h1>
        <p>Fast, secure, and delightfully straightforward. Your card information stays protected with Stripe.</p>
        <div className="feature"><span>✓</span><div><strong>Protected payment</strong><small>Industry-standard encryption</small></div></div>
        <div className="feature"><span>↗</span><div><strong>Instant confirmation</strong><small>See your result right away</small></div></div>
      </div>
      <div className="card payment-card">
        <div className="card-heading"><div><small>ORDER TOTAL</small><strong>{config ? new Intl.NumberFormat('en-AU', { style: 'currency', currency: config.currency }).format(config.amount / 100) : '—'}</strong></div><span>AUD</span></div>
        {loading && !config ? <div className="loading">Preparing secure checkout…</div> : !clientSecret ? (
          <form onSubmit={continueToPayment}>
            <label htmlFor="email">Email for receipt</label>
            <input id="email" type="email" required autoComplete="email" placeholder="you@example.com" value={email} onChange={(event) => setEmail(event.target.value)} />
            {error && <div className="alert error" role="alert">{error}</div>}
            <button className="primary-button" disabled={loading || !config}>{loading ? 'Starting…' : 'Continue to payment'}</button>
          </form>
        ) : stripePromise && config ? (
          <Elements stripe={stripePromise} options={{ clientSecret, appearance }}>
            <PaymentForm email={email} amount={config.amount} currency={config.currency} />
          </Elements>
        ) : <div className="alert error">{error || 'Stripe is not configured.'}</div>}
      </div>
    </section>
  )
}

