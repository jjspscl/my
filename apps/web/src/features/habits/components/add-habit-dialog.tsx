import { useState } from 'react'
import { useForm, useWatch } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Check, Plus } from 'lucide-react'
import { cn } from '@/shared/lib/utils'
import { PALETTE_TOKENS, type PaletteToken } from '@/shared/theme/palette'

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
import { useCreateHabit } from '../hooks/use-habits'
import { CreateHabitSchema, type CreateHabit } from '../schemas/habit.schemas'

interface AddHabitDialogProps {
  trigger?: React.ReactNode
}

export function AddHabitDialog({ trigger }: AddHabitDialogProps) {
  const [open, setOpen] = useState(false)
  const createHabit = useCreateHabit()

  const form = useForm<CreateHabit>({
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    resolver: zodResolver(CreateHabitSchema) as any,
    defaultValues: {
      name: '',
      color: 'blue',
      frequency: 'daily',
      targetPerWeek: 1,
    },
  })

  const onSubmit = (values: CreateHabit) => {
    createHabit.mutate(values, {
      onSuccess: () => {
        setOpen(false)
        form.reset()
      },
    })
  }

  const frequency = useWatch({ control: form.control, name: 'frequency' })

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {trigger || (
          <Button size="sm" className="gap-2">
            <Plus className="h-4 w-4" />
            Add Habit
          </Button>
        )}
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>New Habit</DialogTitle>
        </DialogHeader>

        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4 pt-2">
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel className="text-xs text-muted-foreground">Name</FormLabel>
                  <FormControl>
                    <Input placeholder="Read, Exercise, Meditate..." autoFocus {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="color"
              render={({ field }) => (
                <FormItem>
                  <FormLabel className="text-xs text-muted-foreground">Color</FormLabel>
                  <FormControl>
                    <div className="flex gap-1.5 flex-wrap">
                      {PALETTE_TOKENS.map((t) => (
                        <button
                          key={t}
                          type="button"
                          className={cn(
                            'h-7 w-7 rounded-full flex items-center justify-center transition-transform',
                            field.value === t && 'ring-2 ring-offset-1 ring-foreground scale-110',
                          )}
                          style={{ backgroundColor: `var(--palette-${t})` }}
                          onClick={() => field.onChange(t as PaletteToken)}
                        >
                          {field.value === t && <Check className="h-3.5 w-3.5 text-white" />}
                        </button>
                      ))}
                    </div>
                  </FormControl>
                </FormItem>
              )}
            />

            <div className="flex gap-3">
              <FormField
                control={form.control}
                name="frequency"
                render={({ field }) => (
                  <FormItem className="flex-1">
                    <FormLabel className="text-xs text-muted-foreground">Frequency</FormLabel>
                    <Select value={field.value} onValueChange={field.onChange}>
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        <SelectItem value="daily">Daily</SelectItem>
                        <SelectItem value="weekly">Weekly</SelectItem>
                      </SelectContent>
                    </Select>
                  </FormItem>
                )}
              />

              {frequency === 'weekly' && (
                <FormField
                  control={form.control}
                  name="targetPerWeek"
                  render={({ field }) => (
                    <FormItem className="w-24">
                      <FormLabel className="text-xs text-muted-foreground">Times/week</FormLabel>
                      <FormControl>
                        <Input
                          type="number"
                          min="1"
                          max="7"
                          {...field}
                          onChange={(e) => field.onChange(parseInt(e.target.value) || 1)}
                        />
                      </FormControl>
                    </FormItem>
                  )}
                />
              )}
            </div>

            <Button
              type="submit"
              className="w-full"
              size="sm"
              disabled={createHabit.isPending}
            >
              {createHabit.isPending ? 'Creating...' : 'Create Habit'}
            </Button>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
