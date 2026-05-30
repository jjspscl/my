import { useState } from 'react'
import { useForm, useFieldArray } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Pencil, Plus, Trash2 } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
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
import { Switch } from '@/components/ui/switch'
import { CategoryCombobox } from './category-combobox'
import { useUpsertBudget } from '../hooks/use-budgets'
import type { BudgetCategorySummary } from '../schemas/budget.schemas'
import { z } from 'zod'

const FormSchema = z.object({
  month: z.string(),
  categories: z.array(
    z.object({
      category: z.string().min(1, 'Required'),
      amount: z.string().min(1, 'Required').refine((v) => parseFloat(v) >= 0, 'Must be >= 0'),
      rolloverEnabled: z.boolean(),
    }),
  ),
})
type FormValues = z.infer<typeof FormSchema>

interface Props {
  month: string
  categories: BudgetCategorySummary[]
}

export function BudgetAllocationDialog({ month, categories }: Props) {
  const [open, setOpen] = useState(false)
  const upsert = useUpsertBudget()

  const form = useForm<FormValues>({
    resolver: zodResolver(FormSchema),
    defaultValues: {
      month,
      categories:
        categories.length > 0
          ? categories.map((c) => ({
              category: c.category,
              amount: (c.allocatedCents / 100).toString(),
              rolloverEnabled: c.rolloverEnabled,
            }))
          : [{ category: '', amount: '', rolloverEnabled: false }],
    },
  })

  const { fields, append, remove } = useFieldArray({
    control: form.control,
    name: 'categories',
  })

  const onSubmit = (values: FormValues) => {
    upsert.mutate(
      {
        month: values.month,
        categories: values.categories.map((c) => ({
          category: c.category,
          allocatedCents: Math.round(parseFloat(c.amount) * 100),
          rolloverEnabled: c.rolloverEnabled,
        })),
      },
      {
        onSuccess: () => setOpen(false),
      },
    )
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="ghost" size="sm" className="gap-1">
          <Pencil className="h-3.5 w-3.5" />
          Edit
        </Button>
      </DialogTrigger>
      <DialogContent className="max-h-[80vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Budget Allocations</DialogTitle>
        </DialogHeader>

        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4 pt-2">
            {fields.map((field, index) => (
              <div key={field.id} className="flex items-end gap-2">
                <FormField
                  control={form.control}
                  name={`categories.${index}.category`}
                  render={({ field: f }) => (
                    <FormItem className="flex-1">
                      {index === 0 && <FormLabel className="text-xs">Category</FormLabel>}
                      <FormControl>
                        <CategoryCombobox value={f.value} onChange={f.onChange} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name={`categories.${index}.amount`}
                  render={({ field: f }) => (
                    <FormItem className="w-28">
                      {index === 0 && <FormLabel className="text-xs">Amount</FormLabel>}
                      <FormControl>
                        <Input type="number" step="0.01" min="0" placeholder="0" {...f} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name={`categories.${index}.rolloverEnabled`}
                  render={({ field: f }) => (
                    <FormItem className="w-10">
                      {index === 0 && <FormLabel className="text-xs">Roll</FormLabel>}
                      <FormControl>
                        <Switch checked={f.value} onCheckedChange={f.onChange} />
                      </FormControl>
                    </FormItem>
                  )}
                />
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  className="h-9 w-9 shrink-0"
                  onClick={() => remove(index)}
                  disabled={fields.length === 1}
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </Button>
              </div>
            ))}

            <Button
              type="button"
              variant="outline"
              size="sm"
              className="w-full gap-1"
              onClick={() => append({ category: '', amount: '', rolloverEnabled: false })}
            >
              <Plus className="h-3.5 w-3.5" /> Add Category
            </Button>

            <Button type="submit" className="w-full" size="sm" disabled={upsert.isPending}>
              {upsert.isPending ? 'Saving...' : 'Save Budget'}
            </Button>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}