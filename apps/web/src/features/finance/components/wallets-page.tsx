import { useState } from 'react'
import { Trash2, Landmark, Wallet as WalletIcon, Smartphone } from 'lucide-react'
import { motion } from 'motion/react'

import { Button } from '@/components/ui/button'
import { useWallets, useArchiveWallet } from '../hooks/use-wallets'
import { WalletFormDialog } from './wallet-form-dialog'
import { TransferDialog } from './transfer-dialog'
import type { Wallet } from '../schemas/wallet.schemas'

function formatCents(cents: number): string {
  return `₱${(cents / 100).toLocaleString('en-PH', { minimumFractionDigits: 2 })}`
}

function walletIcon(kind: string) {
  switch (kind) {
    case 'bank':
      return <Landmark className="h-4 w-4" />
    case 'ewallet':
      return <Smartphone className="h-4 w-4" />
    default:
      return <WalletIcon className="h-4 w-4" />
  }
}

function WalletCard({ wallet, onArchive, onEdit }: { wallet: Wallet; onArchive: (id: string) => void; onEdit?: (w: Wallet) => void }) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 4 }}
      animate={{ opacity: 1, y: 0 }}
      className="border rounded-md p-3 space-y-3"
    >
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2 min-w-0">
          {walletIcon(wallet.kind)}
          <span className="text-sm font-medium truncate">{wallet.name}</span>
          {wallet.isDefault && (
            <span className="text-[10px] text-muted-foreground bg-muted px-1.5 py-0.5 rounded">Default</span>
          )}
        </div>
        <div className="flex items-center gap-1">
          {onEdit && (
            <Button variant="ghost" size="sm" className="h-7 text-xs" onClick={() => onEdit(wallet)}>
              Edit
            </Button>
          )}
          <Button variant="ghost" size="icon" className="h-7 w-7" onClick={() => onArchive(wallet.id)}>
            <Trash2 className="h-3 w-3 text-muted-foreground hover:text-destructive" />
          </Button>
        </div>
      </div>

      <div className="flex items-center gap-2 text-xs">
        <span className="text-muted-foreground capitalize">{wallet.kind}</span>
        <span className="text-muted-foreground">·</span>
        <span className="text-muted-foreground">{wallet.currency}</span>
      </div>

      <div className="flex items-center justify-between">
        <p className="text-xs text-muted-foreground">Balance</p>
        <p className={`text-sm font-bold tabular-nums ${wallet.balanceCents >= 0 ? 'text-green-600' : 'text-destructive'}`}>
          {formatCents(wallet.balanceCents)}
        </p>
      </div>

      <div className="grid grid-cols-2 gap-2 text-xs">
        <div>
          <p className="text-muted-foreground">Opening</p>
          <p className="font-medium tabular-nums">{formatCents(wallet.openingBalanceCents)}</p>
        </div>
        <div>
          <p className="text-muted-foreground">Income</p>
          <p className="font-medium tabular-nums text-green-600">{formatCents(wallet.incomeCents)}</p>
        </div>
        <div>
          <p className="text-muted-foreground">Expenses</p>
          <p className="font-medium tabular-nums text-destructive">{formatCents(wallet.expenseCents)}</p>
        </div>
        <div>
          <p className="text-muted-foreground">Transfers</p>
          <p className="font-medium tabular-nums">{formatCents(wallet.incomingTransferCents - wallet.outgoingTransferCents)}</p>
        </div>
      </div>
    </motion.div>
  )
}

export function WalletsPage() {
  const { data: wallets, isLoading } = useWallets()
  const archiveWallet = useArchiveWallet()
  const [editingWallet, setEditingWallet] = useState<Wallet | null>(null)
  const [editingOpen, setEditingOpen] = useState(false)

  const handleArchive = (id: string) => {
    if (confirm('Archive this wallet? It can be restored later.')) {
      archiveWallet.mutate(id)
    }
  }

  const handleEdit = (w: Wallet) => {
    setEditingWallet(w)
    setEditingOpen(true)
  }

  return (
    <div className="space-y-4">
      {/* Actions */}
      <div className="flex items-center gap-2">
        <WalletFormDialog />
        <TransferDialog />
      </div>

      {/* Edit dialog */}
      {editingWallet && (
        <WalletFormDialog
          wallet={editingWallet}
          open={editingOpen}
          onOpenChange={(o) => { setEditingOpen(o); if (!o) setEditingWallet(null) }}
        />
      )}

      {/* Wallets list */}
      <div className="space-y-3">
        {isLoading ? (
          <p className="text-sm text-muted-foreground">Loading...</p>
        ) : !wallets || wallets.length === 0 ? (
          <div className="flex flex-col items-center gap-2 py-12 text-center">
            <WalletIcon className="h-8 w-8 text-muted-foreground" />
            <p className="text-sm text-muted-foreground">No wallets yet</p>
            <p className="text-xs text-muted-foreground">Create your first wallet to start tracking finances</p>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            {wallets.map((w) => (
              <WalletCard key={w.id} wallet={w} onArchive={handleArchive} onEdit={handleEdit} />
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
