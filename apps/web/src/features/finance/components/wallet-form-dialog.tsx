import { useState } from 'react'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Plus } from 'lucide-react'
import { useCreateWallet, useUpdateWallet } from '../hooks/use-wallets'
import type { Wallet } from '../schemas/wallet.schemas'

interface WalletFormDialogProps {
  trigger?: React.ReactNode
  wallet?: Wallet
  open?: boolean
  onOpenChange?: (open: boolean) => void
}

const WALLET_KINDS = [
  { value: 'cash', label: 'Cash' },
  { value: 'bank', label: 'Bank' },
  { value: 'ewallet', label: 'E-Wallet' },
] as const

export function WalletFormDialog({ trigger, wallet, open, onOpenChange }: WalletFormDialogProps) {
  const [name, setName] = useState(wallet?.name ?? '')
  const [kind, setKind] = useState<string>(wallet?.kind ?? 'cash')
  const [openingBalance, setOpeningBalance] = useState(wallet ? String(wallet.openingBalanceCents / 100) : '')

  const createWallet = useCreateWallet()
  const updateWallet = useUpdateWallet()
  const isPending = createWallet.isPending || updateWallet.isPending

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!name) return

    const data = {
      name,
      kind: kind as 'cash' | 'bank' | 'ewallet',
      openingBalanceCents: Math.round(parseFloat(openingBalance || '0') * 100),
    }

    if (wallet) {
      updateWallet.mutate({ id: wallet.id, data }, { onSuccess: () => onOpenChange?.(false) })
    } else {
      createWallet.mutate(data, { onSuccess: () => onOpenChange?.(false) })
    }
  }

  return (
    <Dialog open={open} onOpenChange={(o) => { onOpenChange?.(o) }}>
      <DialogTrigger asChild>
        {trigger ?? (
          <Button size="sm" className="gap-2">
            <Plus className="h-4 w-4" />
            Add Wallet
          </Button>
        )}
      </DialogTrigger>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle className="text-sm">{wallet ? 'Edit Wallet' : 'New Wallet'}</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="wallet-name" className="text-xs">Name</Label>
            <Input id="wallet-name" value={name} onChange={(e) => setName(e.target.value)} className="text-sm" placeholder="Main Account" required />
          </div>
          <div className="space-y-2">
            <Label htmlFor="wallet-kind" className="text-xs">Type</Label>
            <Select value={kind} onValueChange={setKind}>
              <SelectTrigger id="wallet-kind" className="text-sm">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {WALLET_KINDS.map((k) => (
                  <SelectItem key={k.value} value={k.value}>{k.label}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            <Label htmlFor="wallet-balance" className="text-xs">Opening Balance (PHP)</Label>
            <Input id="wallet-balance" type="number" step="0.01" min="0" value={openingBalance} onChange={(e) => setOpeningBalance(e.target.value)} className="text-sm" placeholder="0.00" />
          </div>
          <Button type="submit" className="w-full text-sm" disabled={isPending}>
            {isPending ? 'Saving...' : wallet ? 'Update Wallet' : 'Create Wallet'}
          </Button>
        </form>
      </DialogContent>
    </Dialog>
  )
}
