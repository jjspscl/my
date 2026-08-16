import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { EditTransactionSheet } from './edit-transaction-sheet'
import type { Transaction } from '../schemas/transaction.schemas'

// Controllable mutation + wallet query state.
const state = vi.hoisted(() => ({
  current: {
    isPending: false,
    mutate: vi.fn(),
  },
  wallets: [] as { id: string; name: string }[],
}))

vi.mock('../hooks/use-transactions', () => ({
  useUpdateTransaction: () => state.current,
}))

vi.mock('../hooks/use-wallets', () => ({
  useWallets: () => ({ data: state.wallets }),
}))

vi.mock('../hooks/use-categories', () => ({
  useCategories: () => ({ data: ['food', 'groceries'] }),
}))

const importedTx: Transaction = {
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
}

const manualTx: Transaction = { ...importedTx, imported: false }

describe('EditTransactionSheet', () => {
  beforeEach(() => {
    state.current = { isPending: false, mutate: vi.fn() }
    state.wallets = [
      { id: 'w-1', name: 'Cash' },
      { id: 'w-2', name: 'Bank' },
    ]
    vi.spyOn(window, 'confirm').mockReturnValue(true)
  })

  it('prefills every editable field from the transaction', () => {
    render(
      <EditTransactionSheet transaction={manualTx} open onOpenChange={() => {}} />,
    )
    const amount = screen.getByLabelText('Amount (PHP)') as HTMLInputElement
    expect(amount.value).toBe('123.45')
    const date = screen.getByLabelText('Date') as HTMLInputElement
    expect(date.value).toBe('2026-08-14')
    const desc = screen.getByLabelText('Description (optional)') as HTMLInputElement
    expect(desc.value).toBe('Jollibee')
  })

  it('sends the patch with the current revision', async () => {
    const user = userEvent.setup()
    render(
      <EditTransactionSheet transaction={manualTx} open onOpenChange={() => {}} />,
    )

    const desc = screen.getByLabelText('Description (optional)')
    await user.clear(desc)
    await user.type(desc, 'Lunch with team')
    await user.click(screen.getByRole('button', { name: 'Save changes' }))

    await waitFor(() => {
      expect(state.current.mutate).toHaveBeenCalledWith(
        {
          id: 'tx-1',
          revision: 2,
          data: expect.objectContaining({
            description: 'Lunch with team',
            amountCents: 12345,
            type: 'expense',
            walletId: 'w-1',
            transactionDate: '2026-08-14',
          }),
        },
        expect.anything(),
      )
    })
  })

  it('shows the imported badge and immutable-statement note for imported transactions', () => {
    render(
      <EditTransactionSheet transaction={importedTx} open onOpenChange={() => {}} />,
    )
    expect(screen.getByText(/Imported/)).toBeInTheDocument()
    expect(screen.getByText(/original statement entry is kept unchanged/i)).toBeInTheDocument()
  })

  it('does not show the import warning for manual transactions', () => {
    render(
      <EditTransactionSheet transaction={manualTx} open onOpenChange={() => {}} />,
    )
    expect(screen.queryByText(/original statement entry/i)).not.toBeInTheDocument()
  })
})

Element.prototype.scrollIntoView = Element.prototype.scrollIntoView ?? (() => {})
Element.prototype.hasPointerCapture = Element.prototype.hasPointerCapture ?? (() => false)
Element.prototype.releasePointerCapture = Element.prototype.releasePointerCapture ?? (() => {})

