import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { BulkEditSheet } from './bulk-edit-sheet'
import type { Transaction } from '../schemas/transaction.schemas'
// jsdom lacks ResizeObserver; radix Select/Sheet measurement needs it.
class ResizeObserverStub {
  observe() {}
  unobserve() {}
  disconnect() {}
}
globalThis.ResizeObserver ??= ResizeObserverStub as unknown as typeof ResizeObserver
Element.prototype.scrollIntoView = Element.prototype.scrollIntoView ?? (() => {})
Element.prototype.hasPointerCapture = Element.prototype.hasPointerCapture ?? (() => false)
Element.prototype.releasePointerCapture = Element.prototype.releasePointerCapture ?? (() => {})




const state = vi.hoisted(() => ({
  current: {
    isPending: false,
    mutate: vi.fn(),
  },
  wallets: [] as { id: string; name: string }[],
}))

vi.mock('../hooks/use-transactions', () => ({
  useBulkUpdateTransactions: () => state.current,
}))

vi.mock('../hooks/use-wallets', () => ({
  useWallets: () => ({ data: state.wallets }),
}))

vi.mock('../hooks/use-categories', () => ({
  useCategories: () => ({
    data: [
      { name: 'food', active: true },
      { name: 'groceries', active: true },
    ],
  }),
}))

const selected: Transaction[] = [
  {
    id: 'tx-1',
    amountCents: 12345,
    currency: 'PHP',
    category: 'food',
    description: 'Jollibee',
    type: 'expense',
    walletId: 'w-1',
    transactionDate: '2026-08-14',
    createdAt: '2026-08-14T10:00:00Z',
    revision: 2,
    imported: true,
    importProvider: 'gcash_pdf',
  },
  {
    id: 'tx-2',
    amountCents: 5000,
    currency: 'PHP',
    category: 'transport',
    description: 'Grab',
    type: 'expense',
    walletId: 'w-2',
    transactionDate: '2026-08-15',
    createdAt: '2026-08-15T10:00:00Z',
    revision: 3,
    imported: false,
  },
]

describe('BulkEditSheet', () => {
  beforeEach(() => {
    state.current = { isPending: false, mutate: vi.fn() }
    state.wallets = [
      { id: 'w-1', name: 'Cash' },
      { id: 'w-2', name: 'Bank' },
    ]
  })

  it('does not submit until a field is enabled', async () => {
    const user = userEvent.setup()
    render(<BulkEditSheet selected={selected} open onOpenChange={() => {}} />)

    const save = screen.getByRole('button', { name: /Update 2 transactions/ }) as HTMLButtonElement
    expect(save.disabled).toBe(true)

    await user.click(screen.getByRole('switch', { name: 'Change type' }))
    expect(save.disabled).toBe(false)
  })

  it('sends only enabled fields with per-row revisions', async () => {
    const user = userEvent.setup()
    render(<BulkEditSheet selected={selected} open onOpenChange={() => {}} />)

    await user.click(screen.getByRole('switch', { name: 'Change category' }))
    await user.click(screen.getByRole('combobox'))
    await user.click(screen.getByText('groceries'))
    await user.click(screen.getByRole('switch', { name: 'Change wallet' }))
    await user.click(screen.getByRole('combobox', { name: /Wallet/i }))
    // Radix Select renders a hidden native option plus the visible item span.
    const banks = screen.getAllByText('Bank')
    await user.click(banks[banks.length - 1] as Element)

    await user.click(screen.getByRole('button', { name: /Update 2 transactions/ }))

    await waitFor(() => {
      expect(state.current.mutate).toHaveBeenCalledWith(
        {
          items: [
            { id: 'tx-1', revision: 2 },
            { id: 'tx-2', revision: 3 },
          ],
          patch: { category: 'groceries', walletId: 'w-2' },
        },
        expect.anything(),
      )
    })
  })

  it('reports the selection count in title and button', () => {
    render(<BulkEditSheet selected={selected} open onOpenChange={() => {}} />)
    expect(screen.getByRole('heading', { name: 'Edit 2 transactions' })).toBeInTheDocument()
    expect(screen.getByText(/fields you leave off stay untouched/i)).toBeInTheDocument()
  })
})
