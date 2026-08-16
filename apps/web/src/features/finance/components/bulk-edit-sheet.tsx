import { useState } from 'react'
import { motion } from 'motion/react'

import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { CategoryCombobox } from './category-combobox'
import { useBulkUpdateTransactions } from '../hooks/use-transactions'
import { useWallets } from '../hooks/use-wallets'
import { useMotionPreset } from '@/shared/lib/motion'
import type { BulkUpdatePatch, Transaction } from '../schemas/transaction.schemas'

interface BulkEditSheetProps {
  selected: Transaction[]
  open: boolean
  onOpenChange: (open: boolean) => void
}

// Bulk edit surface: each editable field is opt-in via a "change" toggle so
// untouched fields are never sent to the server. Only category, type, and
// wallet are bulk-editable by design.
export function BulkEditSheet({ selected, open, onOpenChange }: BulkEditSheetProps) {
  const bulkUpdate = useBulkUpdateTransactions()
  const { data: wallets } = useWallets()
  const preset = useMotionPreset()

  const [changeCategory, setChangeCategory] = useState(false)
  const [changeType, setChangeType] = useState(false)
  const [changeWallet, setChangeWallet] = useState(false)
  const [category, setCategory] = useState('')
  const [type, setType] = useState<'expense' | 'income'>('expense')
  const [walletId, setWalletId] = useState('')

  const anyEnabled = changeCategory || changeType || changeWallet
  const canSave = anyEnabled && (!changeCategory || category !== '') && (!changeWallet || walletId !== '')

  const handleOpenChange = (next: boolean) => {
    if (!next && bulkUpdate.isPending) return
    onOpenChange(next)
    if (!next) {
      setChangeCategory(false)
      setChangeType(false)
      setChangeWallet(false)
    }
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!canSave || selected.length === 0) return

    const patch: BulkUpdatePatch = {}
    if (changeCategory) patch.category = category
    if (changeType) patch.type = type
    if (changeWallet) patch.walletId = walletId

    bulkUpdate.mutate(
      { items: selected.map((tx) => ({ id: tx.id, revision: tx.revision })), patch },
      {
        onSuccess: () => {
          onOpenChange(false)
          setChangeCategory(false)
          setChangeType(false)
          setChangeWallet(false)
        },
      },
    )
  }

  return (
    <Sheet open={open} onOpenChange={handleOpenChange}>
      <SheetContent side="right" className="w-full gap-0 p-0 sm:max-w-lg">
        <SheetHeader className="px-6 py-5 pr-12">
          <SheetTitle>Edit {selected.length} transactions</SheetTitle>
          <SheetDescription>
            Enable each field you want to change. Fields you leave off stay untouched.
          </SheetDescription>
        </SheetHeader>

        <motion.form
          variants={preset.container}
          initial="initial"
          animate="animate"
          onSubmit={handleSubmit}
          className="flex min-h-0 flex-1 flex-col"
        >
          <div className="flex-1 space-y-5 overflow-y-auto px-6 pb-6">
            <motion.div variants={preset.field} className="space-y-2">
              <div className="flex items-center justify-between">
                <Label htmlFor="bulk-category" className="text-xs">Category</Label>
                <Switch
                  id="bulk-category"
                  checked={changeCategory}
                  onCheckedChange={setChangeCategory}
                  aria-label="Change category"
                />
              </div>
              {changeCategory && (
                <CategoryCombobox value={category} onChange={setCategory} placeholder="Select category" />
              )}
            </motion.div>

            <motion.div variants={preset.field} className="space-y-2">
              <div className="flex items-center justify-between">
                <Label htmlFor="bulk-type" className="text-xs">Type</Label>
                <Switch
                  id="bulk-type"
                  checked={changeType}
                  onCheckedChange={setChangeType}
                  aria-label="Change type"
                />
              </div>
              {changeType && (
                <Select value={type} onValueChange={(v) => setType(v as 'expense' | 'income')}>
                  <SelectTrigger className="w-full text-sm">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="expense">Expense</SelectItem>
                    <SelectItem value="income">Income</SelectItem>
                  </SelectContent>
                </Select>
              )}
            </motion.div>

            <motion.div variants={preset.field} className="space-y-2">
              <div className="flex items-center justify-between">
                <Label htmlFor="bulk-wallet" className="text-xs">Wallet</Label>
                <Switch
                  id="bulk-wallet"
                  checked={changeWallet}
                  onCheckedChange={setChangeWallet}
                  aria-label="Change wallet"
                />
              </div>
              {changeWallet && (
                <Select value={walletId} onValueChange={setWalletId}>
                  <SelectTrigger className="w-full text-sm" aria-label="Wallet">
                    <SelectValue placeholder="Select wallet" />
                  </SelectTrigger>
                  <SelectContent>
                    {(wallets ?? []).map((w) => (
                      <SelectItem key={w.id} value={w.id}>{w.name}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
            </motion.div>
          </div>

          <SheetFooter className="border-t px-6 py-4">
            <Button type="submit" size="sm" disabled={!canSave || bulkUpdate.isPending}>
              {bulkUpdate.isPending ? 'Saving...' : `Update ${selected.length} transaction${selected.length === 1 ? '' : 's'}`}
            </Button>
          </SheetFooter>
        </motion.form>
      </SheetContent>
    </Sheet>
  )
}
