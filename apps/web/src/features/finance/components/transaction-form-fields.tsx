import { type Control } from 'react-hook-form'
import { z } from 'zod'
import { motion } from 'motion/react'

import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { CategoryCombobox } from './category-combobox'
import { useMotionPreset } from '@/shared/lib/motion'
import type { Wallet } from '../schemas/wallet.schemas'
import { todayLocalStr } from '@/shared/lib/utils'

// Shared form schema for create and edit. Amount is a string because the
// input field is free-typed; it is converted to minor units on submit.
export const TransactionFormSchema = z.object({
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
export type TransactionFormValues = z.infer<typeof TransactionFormSchema>

export const emptyFormDefaults = (type: 'expense' | 'income'): TransactionFormValues => ({
  amount: '',
  category: '',
  description: '',
  type,
  walletId: '',
  transactionDate: todayLocalStr(),
})

export function amountToCents(amount: string): number {
  return Math.round(parseFloat(amount) * 100)
}

export function centsToAmount(cents: number): string {
  return (cents / 100).toFixed(2)
}

interface TransactionFormFieldsProps {
  control: Control<TransactionFormValues>
  wallets?: Wallet[]
}

// Field group shared by the add-expense dialog and the edit sheet so both
// surfaces validate and look identical. Groups stagger in on mount.
export function TransactionFormFields({ control, wallets }: TransactionFormFieldsProps) {
  const preset = useMotionPreset()

  return (
    <div className="space-y-5">
      <motion.div variants={preset.field} className="grid grid-cols-[1fr_auto] gap-3">
        <FormField
          control={control}
          name="amount"
          render={({ field }) => (
            <FormItem className="min-w-0">
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
          control={control}
          name="type"
          render={({ field }) => (
            <FormItem>
              <FormLabel className="text-xs text-muted-foreground">Type</FormLabel>
              <Select value={field.value} onValueChange={field.onChange}>
                <FormControl>
                  <SelectTrigger className="w-[120px]">
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
      </motion.div>

      {wallets && wallets.length > 0 && (
        <motion.div variants={preset.field}>
          <FormField
            control={control}
            name="walletId"
            render={({ field }) => (
              <FormItem>
                <FormLabel className="text-xs text-muted-foreground">Wallet</FormLabel>
                <Select value={field.value} onValueChange={field.onChange}>
                  <FormControl>
                    <SelectTrigger className="w-full text-sm">
                      <SelectValue placeholder="Select wallet" />
                    </SelectTrigger>
                  </FormControl>
                  <SelectContent>
                    {wallets.map((w) => (
                      <SelectItem key={w.id} value={w.id}>{w.name}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <FormMessage />
              </FormItem>
            )}
          />
        </motion.div>
      )}

      <motion.div variants={preset.field}>
        <FormField
          control={control}
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
      </motion.div>

      <motion.div variants={preset.field}>
        <FormField
          control={control}
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
      </motion.div>

      <motion.div variants={preset.field}>
        <FormField
          control={control}
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
      </motion.div>
    </div>
  )
}
