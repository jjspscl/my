import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'

import { Button } from '@/components/ui/button'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Form } from '@/components/ui/form'
import { Badge } from '@/components/ui/badge'
import {
  TransactionFormFields,
  TransactionFormSchema,
  amountToCents,
  centsToAmount,
  type TransactionFormValues,
} from './transaction-form-fields'
import { useUpdateTransaction } from '../hooks/use-transactions'
import { useWallets } from '../hooks/use-wallets'
import type { Transaction } from '../schemas/transaction.schemas'

interface EditTransactionSheetProps {
  transaction: Transaction | null
  open: boolean
  onOpenChange: (open: boolean) => void
}

// Edit surface for one transaction: responsive sheet (full-screen on mobile),
// prefilled values, explicit save, and a warning when the record came from an
// import (the statement entry stays immutable).
export function EditTransactionSheet({
  transaction,
  open,
  onOpenChange,
}: EditTransactionSheetProps) {
  const updateTx = useUpdateTransaction()
  const { data: wallets } = useWallets()

  const form = useForm<TransactionFormValues>({
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    resolver: zodResolver(TransactionFormSchema) as any,
    values: transaction
      ? {
          amount: centsToAmount(transaction.amountCents),
          category: transaction.category,
          description: transaction.description,
          type: transaction.type,
          walletId: transaction.walletId,
          transactionDate: transaction.transactionDate,
        }
      : undefined,
    defaultValues: {
      amount: '',
      category: '',
      description: '',
      type: 'expense',
      walletId: '',
      transactionDate: '',
    },
  })

  const handleOpenChange = (next: boolean) => {
    if (!next && form.formState.isDirty) {
      if (!window.confirm('Discard unsaved changes?')) return
    }
    onOpenChange(next)
  }

  const onSubmit = (values: TransactionFormValues) => {
    if (!transaction) return
    updateTx.mutate(
      {
        id: transaction.id,
        revision: transaction.revision,
        data: {
          amountCents: amountToCents(values.amount),
          category: values.category,
          description: values.description,
          type: values.type,
          walletId: values.walletId,
          transactionDate: values.transactionDate,
        },
      },
      {
        onSuccess: () => {
          onOpenChange(false)
          form.reset()
        },
      },
    )
  }

  return (
    <Sheet open={open} onOpenChange={handleOpenChange}>
      <SheetContent side="right" className="w-full gap-6 overflow-y-auto sm:max-w-lg">
        <SheetHeader>
          <SheetTitle>Edit transaction</SheetTitle>
          <SheetDescription>
            {transaction?.imported ? (
              <span className="flex flex-col gap-2">
                <Badge variant="outline" className="w-fit gap-1 text-xs">
                  Imported{transaction.importProvider ? ` from ${transaction.importProvider}` : ''}
                </Badge>
                <span>
                  Changes apply to this transaction only. The original statement
                  entry is kept unchanged.
                </span>
              </span>
            ) : (
              'Update the details below. Saving may affect balances and bill matching.'
            )}
          </SheetDescription>
        </SheetHeader>

        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <TransactionFormFields control={form.control} wallets={wallets} />
            <SheetFooter>
              <Button
                type="submit"
                className="w-full sm:w-auto"
                disabled={updateTx.isPending}
              >
                {updateTx.isPending ? 'Saving...' : 'Save changes'}
              </Button>
            </SheetFooter>
          </form>
        </Form>
      </SheetContent>
    </Sheet>
  )
}
