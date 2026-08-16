import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { TransactionsPage } from './transactions-page'
import type { Transaction } from '../schemas/transaction.schemas'

const state = vi.hoisted(() => ({
  current: {
    isPending: false,
    mutate: vi.fn(),
  },
  transactions: [] as Transaction[],
}))

vi.mock('../hooks/use-transactions', () => ({
  useTransactions: () => ({ data: state.transactions, isLoading: false, isError: false }),
  useDeleteTransaction: () => state.current,
  useUpdateTransaction: () => ({ isPending: false, mutate: vi.fn() }),
}))

vi.mock('@/shared/sync/network-status', () => ({
  useNetworkStatus: () => true,
}))

vi.mock('./edit-transaction-sheet', () => ({
  EditTransactionSheet: () => null,
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
