import { afterEach, describe, expect, it, vi } from 'vitest'
import { createPaymentIntent, getPayments } from './api'

describe('payment API', () => {
  afterEach(() => vi.restoreAllMocks())

  it('creates a payment intent with the receipt email', async () => {
    const fetchMock = vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({ clientSecret: 'secret', paymentIntentId: 'pi_1' }), { status: 201 }))
    await expect(createPaymentIntent('buyer@example.com')).resolves.toEqual({ clientSecret: 'secret', paymentIntentId: 'pi_1' })
    expect(fetchMock).toHaveBeenCalledWith('http://localhost:8080/api/payment-intents', expect.objectContaining({ body: JSON.stringify({ email: 'buyer@example.com' }) }))
  })

  it('returns an empty report', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({ payments: [] }), { status: 200 }))
    await expect(getPayments()).resolves.toEqual({ payments: [] })
  })

  it('surfaces API errors', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response(JSON.stringify({ error: 'Redis unavailable' }), { status: 503 }))
    await expect(getPayments()).rejects.toThrow('Redis unavailable')
  })
})

