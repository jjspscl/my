import { useState } from 'react'
import { useForm, useWatch } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Plus } from 'lucide-react'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import {
  Form,
} from '@/components/ui/form'
import {
  TransactionFormFields,
  TransactionFormSchema,
  amountToCents,
  emptyFormDefaults,
  type TransactionFormValues,
} from './transaction-form-fields'
import { randomUUID } from '@/shared/lib/uuid'
import { useCreateTransaction } from '../hooks/use-transactions'
import { useWallets } from '../hooks/use-wallets'
import { type CreateTransaction } from '../schemas/transaction.schemas'

interface AddExpenseDialogProps {
  trigger?: React.ReactNode
  defaultType?: 'expense' | 'income'
}

export function AddExpenseDialog({ trigger, defaultType = 'expense' }: AddExpenseDialogProps) {
  const [open, setOpen] = useState(false)
  const createTx = useCreateTransaction()
  const { data: wallets } = useWallets()
  // One key per form session: a double-submit or a queued replay reuses it, so
  // the server dedupes instead of recording the expense twice.
  const [idempotencyKey, setIdempotencyKey] = useState(() => randomUUID())

  const form = useForm<TransactionFormValues>({
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    resolver: zodResolver(TransactionFormSchema) as any,
    defaultValues: emptyFormDefaults(defaultType),
  })

  const formType = useWatch({ control: form.control, name: 'type' })

  const onSubmit = (values: TransactionFormValues) => {
    const data: CreateTransaction = {
      amountCents: amountToCents(values.amount),
      category: values.category,
      description: values.description,
      type: values.type,
      walletId: values.walletId,
      transactionDate: values.transactionDate,
      idempotencyKey,
    }

    createTx.mutate(data, {
      onSuccess: () => {
        setIdempotencyKey(randomUUID())
        setOpen(false)
        form.reset()
      },
    })
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {trigger || (
          <Button variant="outline" size="sm" className="w-full gap-2">
            <Plus className="h-4 w-4" />
            Add expense
          </Button>
        )}
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add {formType === 'expense' ? 'Expense' : 'Income'}</DialogTitle>
        </DialogHeader>

        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4 pt-2">
            <TransactionFormFields control={form.control} wallets={wallets} />

            <Button
              type="submit"
              className="w-full"
              size="sm"
              disabled={createTx.isPending}
            >
              {createTx.isPending ? 'Saving...' : `Add ${formType === 'expense' ? 'Expense' : 'Income'}`}
            </Button>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
