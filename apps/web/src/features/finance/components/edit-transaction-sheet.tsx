import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { motion } from 'motion/react'

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
import { useMotionPreset } from '@/shared/lib/motion'
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
  const preset = useMotionPreset()

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
      <SheetContent side="right" className="w-full gap-0 p-0 sm:max-w-lg">
        <SheetHeader className="px-6 py-5 pr-12">
          <SheetTitle>Edit transaction</SheetTitle>
          {transaction?.imported && (
            <motion.div
              initial={preset.item.initial}
              animate={preset.item.animate}
              className="flex flex-col gap-1.5"
            >
              <Badge variant="outline" className="w-fit gap-1 text-xs">
                Imported
                {transaction.importProvider ? ` from ${transaction.importProvider}` : ''}
              </Badge>
            </motion.div>
          )}
          <SheetDescription>
            {transaction?.imported
              ? 'Changes apply to this transaction only. The original statement entry is kept unchanged.'
              : 'Update the details below. Saving may affect balances and bill matching.'}
          </SheetDescription>
        </SheetHeader>

        <Form {...form}>
          <form
            onSubmit={form.handleSubmit(onSubmit)}
            className="flex min-h-0 flex-1 flex-col"
          >
            <motion.div
              variants={preset.container}
              initial="initial"
              animate="animate"
              className="flex-1 overflow-y-auto px-6 pb-6"
            >
              <TransactionFormFields control={form.control} wallets={wallets} />
            </motion.div>

            <SheetFooter className="border-t px-6 py-4">
              <Button type="submit" size="sm" disabled={updateTx.isPending}>
                {updateTx.isPending ? 'Saving...' : 'Save changes'}
              </Button>
            </SheetFooter>
          </form>
        </Form>
      </SheetContent>
    </Sheet>
  )
}
