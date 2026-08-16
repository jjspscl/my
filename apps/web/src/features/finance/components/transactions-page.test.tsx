import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { TransactionsPage } from './transactions-page'
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
  bulk: {
    isPending: false,
    mutate: vi.fn(),
  },
  transactions: [] as Transaction[],
}))

vi.mock('../hooks/use-transactions', () => ({
  useTransactions: () => ({ data: state.transactions, isLoading: false, isError: false }),
  useDeleteTransaction: () => state.current,
  useUpdateTransaction: () => ({ isPending: false, mutate: vi.fn() }),
  useBulkUpdateTransactions: () => ({ isPending: false, mutate: vi.fn() }),
  useBulkDeleteTransactions: () => state.bulk,
}))

vi.mock('@/shared/sync/network-status', () => ({
  useNetworkStatus: () => true,
}))

vi.mock('./edit-transaction-sheet', () => ({
  EditTransactionSheet: () => null,
}))

vi.mock('./bulk-edit-sheet', () => ({
  BulkEditSheet: () => null,
}))

const tx: Transaction = {
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

describe('TransactionsPage delete flow', () => {
  beforeEach(() => {
    state.current = { isPending: false, mutate: vi.fn() }
    state.transactions = [tx]
  })

  it('marks imported transactions in the list', () => {
    render(<TransactionsPage />)
    expect(screen.getByText('imported')).toBeInTheDocument()
  })

  it('opens a transaction-specific confirmation and warns about imports', async () => {
    const user = userEvent.setup()
    render(<TransactionsPage />)

    await user.click(screen.getByRole('button', { name: /Actions for Jollibee/ }))
    await user.click(screen.getByRole('menuitem', { name: /Delete transaction/ }))

    expect(
      screen.getByRole('heading', { name: 'Delete this transaction?' }),
    ).toBeInTheDocument()
    // Amount and description appear in both the table row and the dialog;
    // the dialog must render its own copies.
    expect(screen.getAllByText(/123\.45/).length).toBeGreaterThanOrEqual(1)
    expect(screen.getAllByText(/Jollibee/).length).toBeGreaterThanOrEqual(1)
    expect(screen.getByText(/imported statement/i)).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Delete transaction' }))
    expect(state.current.mutate).toHaveBeenCalledWith(
      { id: 'tx-1', revision: 2 },
      expect.anything(),
    )
  })
})

describe('TransactionsPage bulk selection', () => {
  beforeEach(() => {
    state.bulk = { isPending: false, mutate: vi.fn() }
    state.transactions = [
      tx,
      { ...tx, id: 'tx-2', description: 'Grab ride', imported: false },
    ]
  })

  it('shows the bulk bar after selecting rows and deletes with revisions', async () => {
    const user = userEvent.setup()
    render(<TransactionsPage />)

    await user.click(screen.getByRole('checkbox', { name: 'Select Jollibee on 2026-08-14' }))
    await user.click(screen.getByRole('checkbox', { name: 'Select Grab ride on 2026-08-14' }))

    expect(screen.getByText('2 selected')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /^Delete$/ }))
    expect(
      screen.getByRole('heading', { name: 'Delete 2 transactions?' }),
    ).toBeInTheDocument()
    // Selection includes an imported row, so the warning appears.
    expect(screen.getByText(/includes imported transactions/i)).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Delete 2 transactions' }))
    expect(state.bulk.mutate).toHaveBeenCalledWith(
      {
        items: [
          { id: 'tx-1', revision: 2 },
          { id: 'tx-2', revision: 2 },
        ],
      },
      expect.anything(),
    )
  })

  it('tri-state header checkbox selects and clears all visible rows', async () => {
    const user = userEvent.setup()
    render(<TransactionsPage />)

    await user.click(screen.getByRole('checkbox', { name: 'Select all visible transactions' }))
    expect(screen.getByText('2 selected')).toBeInTheDocument()

    await user.click(screen.getByRole('checkbox', { name: 'Select Jollibee on 2026-08-14' }))
    expect(screen.getByText('1 selected')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Clear selection' }))
    // The bar fades out via AnimatePresence; wait for it to leave the DOM.
    await waitFor(() => {
      expect(screen.queryByText(/selected/)).not.toBeInTheDocument()
    })
  })
})
