import { useState } from 'react'
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
import { useCreateHabit } from '../hooks/use-habits'

interface AddHabitDialogProps {
  trigger?: React.ReactNode
}

export function AddHabitDialog({ trigger }: AddHabitDialogProps) {
  const [open, setOpen] = useState(false)
  const [name, setName] = useState('')
  const [color, setColor] = useState<PaletteToken>('blue')
  const [frequency, setFrequency] = useState<'daily' | 'weekly'>('daily')
  const [targetPerWeek, setTargetPerWeek] = useState('1')

  const createHabit = useCreateHabit()

  const handleSubmit = () => {
    if (!name.trim()) return

    createHabit.mutate(
      {
        name: name.trim(),
        color,
        frequency,
        targetPerWeek: frequency === 'weekly' ? parseInt(targetPerWeek) || 1 : 1,
      },
      {
        onSuccess: () => {
          setOpen(false)
          setName('')
          setColor('blue')
          setFrequency('daily')
          setTargetPerWeek('1')
        },
      },
    )
  }

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

        <div className="space-y-4 pt-2">
          {/* Name */}
          <div>
            <label className="text-xs text-muted-foreground mb-1 block">Name</label>
            <Input
              placeholder="Read, Exercise, Meditate..."
              value={name}
              onChange={(e) => setName(e.target.value)}
              autoFocus
            />
          </div>

          {/* Color picker */}
          <div>
            <label className="text-xs text-muted-foreground mb-1.5 block">Color</label>
            <div className="flex gap-1.5 flex-wrap">
              {PALETTE_TOKENS.map((t) => (
                  <button
                    key={t}
                    type="button"
                    className={cn(
                      'h-7 w-7 rounded-full flex items-center justify-center transition-transform',
                      color === t && 'ring-2 ring-offset-1 ring-foreground scale-110',
                    )}
                    style={{ backgroundColor: `var(--palette-${t})` }}
                  onClick={() => setColor(t as PaletteToken)}
                >
                  {color === t && <Check className="h-3.5 w-3.5 text-white" />}
                </button>
              ))}
            </div>
          </div>

          {/* Frequency */}
          <div className="flex gap-3">
            <div className="flex-1">
              <label className="text-xs text-muted-foreground mb-1 block">Frequency</label>
              <Select
                value={frequency}
                onValueChange={(v: 'daily' | 'weekly') => setFrequency(v)}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="daily">Daily</SelectItem>
                  <SelectItem value="weekly">Weekly</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {frequency === 'weekly' && (
              <div className="w-24">
                <label className="text-xs text-muted-foreground mb-1 block">Times/week</label>
                <Input
                  type="number"
                  min="1"
                  max="7"
                  value={targetPerWeek}
                  onChange={(e) => setTargetPerWeek(e.target.value)}
                />
              </div>
            )}
          </div>

          <Button
            className="w-full"
            size="sm"
            onClick={handleSubmit}
            disabled={!name.trim() || createHabit.isPending}
          >
            {createHabit.isPending ? 'Creating...' : 'Create Habit'}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}