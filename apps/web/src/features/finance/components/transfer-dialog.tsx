import { useState } from 'react'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { ArrowRightLeft } from 'lucide-react'
import { useCreateTransfer } from '../hooks/use-transfers'
import { useWallets } from '../hooks/use-wallets'
import type { Wallet } from '../schemas/wallet.schemas'
import { todayLocalStr } from '@/shared/lib/utils'

interface TransferDialogProps {
  trigger?: React.ReactNode
}

export function TransferDialog({ trigger }: TransferDialogProps) {
  const [open, setOpen] = useState(false)
  const [fromWalletId, setFromWalletId] = useState('')
  const [toWalletId, setToWalletId] = useState('')
  const [amount, setAmount] = useState('')
  const [description, setDescription] = useState('')
  const [transferDate, setTransferDate] = useState(todayLocalStr())
  // One key per form session: a double-submit or a queued replay reuses it, so
  // the server dedupes instead of moving money twice.
  const [idempotencyKey, setIdempotencyKey] = useState(() => crypto.randomUUID())

  const { data: wallets } = useWallets()
  const createTransfer = useCreateTransfer()
  const isPending = createTransfer.isPending

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    const cents = Math.round(parseFloat(amount) * 100)
    if (!fromWalletId || !toWalletId || !cents) return

    createTransfer.mutate(
      { fromWalletId, toWalletId, amountCents: cents, description, transferDate, idempotencyKey },
      {
        onSuccess: () => {
          setIdempotencyKey(crypto.randomUUID())
          setOpen(false)
          setFromWalletId('')
          setToWalletId('')
          setAmount('')
          setDescription('')
        },
      },
    )
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {trigger || (
          <Button variant="outline" size="sm" className="gap-2">
            <ArrowRightLeft className="h-4 w-4" />
            Transfer
          </Button>
        )}
      </DialogTrigger>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle className="text-sm">Transfer Between Wallets</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label className="text-xs">From Wallet</Label>
            <Select value={fromWalletId} onValueChange={setFromWalletId}>
              <SelectTrigger className="text-sm">
                <SelectValue placeholder="Select source" />
              </SelectTrigger>
              <SelectContent>
                {(wallets ?? []).map((w: Wallet) => (
                  <SelectItem key={w.id} value={w.id}>{w.name}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <Label className="text-xs">To Wallet</Label>
            <Select value={toWalletId} onValueChange={setToWalletId}>
              <SelectTrigger className="text-sm">
                <SelectValue placeholder="Select destination" />
              </SelectTrigger>
              <SelectContent>
                {(wallets ?? []).map((w: Wallet) => (
                  <SelectItem key={w.id} value={w.id}>{w.name}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <Label htmlFor="transfer-amount" className="text-xs">Amount (PHP)</Label>
            <Input id="transfer-amount" type="number" step="0.01" min="0.01" value={amount} onChange={(e) => setAmount(e.target.value)} className="text-sm" placeholder="1,000" required />
          </div>
          <div className="space-y-2">
            <Label htmlFor="transfer-desc" className="text-xs">Description (optional)</Label>
            <Input id="transfer-desc" value={description} onChange={(e) => setDescription(e.target.value)} className="text-sm" placeholder="Savings transfer" />
          </div>
          <div className="space-y-2">
            <Label htmlFor="transfer-date" className="text-xs">Date</Label>
            <Input id="transfer-date" type="date" value={transferDate} onChange={(e) => setTransferDate(e.target.value)} className="text-sm" required />
          </div>
          <Button type="submit" className="w-full text-sm" disabled={isPending}>
            {isPending ? 'Sending...' : 'Transfer'}
          </Button>
        </form>
      </DialogContent>
    </Dialog>
  )
}
