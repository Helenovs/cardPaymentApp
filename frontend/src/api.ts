const API_URL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'

export type Payment = {
  id: string
  amount: number
  currency: string
  status: string
  customerEmail?: string
  paymentMethod?: string
  lastError?: string
  createdAt: string
  updatedAt: string
  lastEventType?: string
}

export type AppConfig = { publishableKey: string; amount: number; currency: string }

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${API_URL}${path}`, init)
  const body = await response.json().catch(() => ({}))
  if (!response.ok) throw new Error(body.error ?? `Request failed (${response.status})`)
  return body as T
}

export const getConfig = () => request<AppConfig>('/api/config')
export const createPaymentIntent = (email: string) =>
  request<{ clientSecret: string; paymentIntentId: string }>('/api/payment-intents', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email }),
  })
export const getPayments = () => request<{ payments: Payment[] }>('/api/payments?limit=100')

