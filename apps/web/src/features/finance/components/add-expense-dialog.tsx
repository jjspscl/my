import { useState } from 'react'
import { useForm, useWatch } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Plus } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { CategoryCombobox } from './category-combobox'
import { randomUUID } from '@/shared/lib/uuid'
import { useCreateTransaction } from '../hooks/use-transactions'
import { useWallets } from '../hooks/use-wallets'
import { type CreateTransaction } from '../schemas/transaction.schemas'
import type { Wallet } from '../schemas/wallet.schemas'
import { todayLocalStr } from '@/shared/lib/utils'
import { z } from 'zod'

// Form schema extends CreateTransactionSchema but uses string for amount (input field)
const FormSchema = z.object({
  amount: z.string().min(1, 'Amount is required').refine(
    (v) => parseFloat(v) > 0,
    'Amount must be positive',
  ),
  category: z.string().min(1, 'Category is required'),
  description: z.string().default(''),
  type: z.enum(['expense', 'income']),
  walletId: z.string(),
  transactionDate: z.string().min(1),
})
type FormValues = z.infer<typeof FormSchema>

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

  const form = useForm<FormValues>({
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    resolver: zodResolver(FormSchema) as any,
    defaultValues: {
      amount: '',
      category: '',
      description: '',
      type: defaultType,
      walletId: '',
      transactionDate: todayLocalStr(),
    },
  })

  const formType = useWatch({ control: form.control, name: 'type' })

  const onSubmit = (values: FormValues) => {
    const data: CreateTransaction = {
      amountCents: Math.round(parseFloat(values.amount) * 100),
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
            <div className="flex gap-2">
              <FormField
                control={form.control}
                name="amount"
                render={({ field }) => (
                  <FormItem className="flex-1">
                    <FormLabel className="text-xs text-muted-foreground">Amount (PHP)</FormLabel>
                    <FormControl>
                      <Input
                        type="number"
                        step="0.01"
                        min="0"
                        placeholder="0.00"
                        autoFocus
                        {...field}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="type"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel className="text-xs text-muted-foreground">Type</FormLabel>
                    <Select value={field.value} onValueChange={field.onChange}>
                      <FormControl>
                        <SelectTrigger className="w-[110px]">
                          <SelectValue />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        <SelectItem value="expense">Expense</SelectItem>
                        <SelectItem value="income">Income</SelectItem>
                      </SelectContent>
                    </Select>
                  </FormItem>
                )}
              />
            </div>

            {wallets && wallets.length > 0 && (
              <FormField
                control={form.control}
                name="walletId"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel className="text-xs text-muted-foreground">Wallet</FormLabel>
                    <Select value={field.value} onValueChange={field.onChange}>
                      <FormControl>
                        <SelectTrigger className="text-sm">
                          <SelectValue placeholder="Select wallet" />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        {wallets.map((w: Wallet) => (
                          <SelectItem key={w.id} value={w.id}>{w.name}</SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    <FormMessage />
                  </FormItem>
                )}
              />
            )}

            <FormField
              control={form.control}
              name="category"
              render={({ field }) => (
                <FormItem>
                  <FormLabel className="text-xs text-muted-foreground">Category</FormLabel>
                  <FormControl>
                    <CategoryCombobox value={field.value} onChange={field.onChange} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="description"
              render={({ field }) => (
                <FormItem>
                  <FormLabel className="text-xs text-muted-foreground">Description (optional)</FormLabel>
                  <FormControl>
                    <Input placeholder="Coffee, lunch..." {...field} />
                  </FormControl>
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="transactionDate"
              render={({ field }) => (
                <FormItem>
                  <FormLabel className="text-xs text-muted-foreground">Date</FormLabel>
                  <FormControl>
                    <Input type="date" {...field} />
                  </FormControl>
                </FormItem>
              )}
            />

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
