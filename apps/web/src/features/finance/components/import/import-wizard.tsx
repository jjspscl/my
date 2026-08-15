import { useMemo, useState } from 'react'
import { AlertTriangle, Bot, CheckCircle2, FileUp, Loader2, ShieldCheck } from 'lucide-react'
import { toast } from 'sonner'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

import { GcashParseError, parseGcashPdf, type GcashParsedStatement } from '../../lib/gcash-pdf-parser'
import { classifyRow, type DraftKind } from '../../lib/gcash-classifier'
import { parseCents } from '../../lib/gcash-text'
import { useCreateImport, useImports, useRollbackImport } from '../../hooks/use-imports'
import { useWallets } from '../../hooks/use-wallets'
import { useRunAnalysis } from '@/features/intelligence/hooks/use-intelligence'
import { sha256Hex } from '@/shared/lib/sha256'
import type { CreateImport, ImportBatch, ImportRowDraft } from '../../schemas/import.schemas'
import {
  confidenceBucket,
  type AnalysisResult,
  type Suggestion,
} from '@/features/intelligence/schemas/intelligence.schemas'
import type { Wallet } from '../../schemas/wallet.schemas'
import { formatCents } from '../../lib/format'
import { AiAnalysisCard } from '@/features/intelligence/components/ai-analysis-card'

const MAX_FILE_BYTES = 20 * 1024 * 1024

type Step = 'upload' | 'wallet' | 'review' | 'result'

interface WizardState {
  step: Step
  file: File | null
  password: string
  parsing: boolean
  statement: GcashParsedStatement | null
  fingerprint: string
  drafts: ImportRowDraft[]
  walletId: string
  createWallet: boolean
  newWalletName: string
  openingBalanceCents: number
  importing: boolean
  analyzing: boolean
  analysis: AnalysisResult | null
  result: ImportBatch | null
}

const initial: WizardState = {
  step: 'upload',
  file: null,
  password: '',
  parsing: false,
  statement: null,
  fingerprint: '',
  drafts: [],
  walletId: '',
  createWallet: false,
  newWalletName: 'GCash',
  openingBalanceCents: 0,
  importing: false,
  analyzing: false,
  analysis: null,
  result: null,
}

/** "2026-07-01 09:30 AM" -> RFC3339 in the device's local timezone. */
function toRfc3339Local(dateTime: string): string {
  const m = dateTime.match(/^(\d{4})-(\d{2})-(\d{2})\s+(\d{1,2}):(\d{2})\s+([AP])M$/)
  if (!m) return new Date().toISOString()
  const [, y, mo, d, h, mi, ap] = m
  let hour = Number(h)
  if (ap === 'P' && hour !== 12) hour += 12
  if (ap === 'A' && hour === 12) hour = 0
  const date = new Date(Number(y), Number(mo) - 1, Number(d), hour, Number(mi))
  const pad = (n: number) => String(n).padStart(2, '0')
  const offset = -date.getTimezoneOffset()
  const sign = offset >= 0 ? '+' : '-'
  const abs = Math.abs(offset)
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(
    date.getHours(),
  )}:${pad(date.getMinutes())}:00${sign}${pad(Math.floor(abs / 60))}:${pad(abs % 60)}`
}

function buildDrafts(statement: GcashParsedStatement, walletNames: string[]): ImportRowDraft[] {
  return statement.rows.map((row) => {
    const debitCents = parseCents(row.debit)
    const creditCents = parseCents(row.credit)
    const amountCents = debitCents ?? creditCents ?? 0
    const classification = classifyRow(row, walletNames)

    const kind: DraftKind = creditCents !== null && debitCents === null
      ? (classification.kind === 'income' ? 'income' : 'transfer_in')
      : (classification.kind === 'transfer_out' ? 'transfer_out' : 'expense')

    return {
      sourceReference: row.referenceNo || `no-ref-${row.dateTime}`,
      occurredAt: toRfc3339Local(row.dateTime),
      amountCents,
      kind,
      category: classification.category,
      description: row.description || row.dateTime,
      counterparty: classification.counterparty,
      counterWalletId: '',
      excluded: false,
      suggestedKind: classification.kind,
      suggestedCategory: classification.category,
    }
  })
}

const KIND_LABELS: Record<DraftKind, string> = {
  expense: 'Expense',
  income: 'Income',
  transfer_out: 'Transfer out',
  transfer_in: 'Transfer in',
}

export function ImportWizard() {
  const [state, setState] = useState<WizardState>(initial)
  const set = (patch: Partial<WizardState>) => setState((s) => ({ ...s, ...patch }))

  const { data: wallets, isLoading: walletsLoading } = useWallets()
  const createImport = useCreateImport()
  const { data: imports } = useImports()
  const rollback = useRollbackImport()
  const runAnalysis = useRunAnalysis()

  const phpWallets = useMemo(
    () => (wallets ?? []).filter((w) => w.currency === 'PHP' && !w.archivedAt),
    [wallets],
  )

  const selectedDrafts = state.drafts.filter((d) => !d.excluded)
  const netCents = useMemo(
    () =>
      selectedDrafts.reduce((sum, d) => {
        const signed = d.kind === 'expense' || d.kind === 'transfer_out' ? -d.amountCents : d.amountCents
        return sum + signed
      }, 0),
    [selectedDrafts],
  )

  const handleFile = async (file: File) => {
    if (file.size > MAX_FILE_BYTES) {
      toast.error('Statement is larger than 20 MB.')
      return
    }
    set({ file, step: 'upload', statement: null, drafts: [], result: null })
  }

  const handleParse = async () => {
    if (!state.file) return
    set({ parsing: true })
    try {
      const buffer = await state.file.arrayBuffer()
      // Hash FIRST: pdf.js transfers ownership of the buffer to its worker
      // (detaching it), so anything that still needs the bytes must run
      // before parsing starts. Hashing first also stays correct on insecure
      // origins where the @noble/hashes fallback loads lazily.
      const fingerprint = await sha256Hex(buffer)
      const statement = await parseGcashPdf(buffer, state.password || undefined)
      const drafts = buildDrafts(statement, (wallets ?? []).map((w) => w.name))
      const ending = statement.endingBalanceCents ?? 0
      const opening = ending - netOf(drafts)
      set({
        statement,
        fingerprint,
        drafts,
        step: 'wallet',
        openingBalanceCents: Math.max(0, opening),
        walletId: phpWallets.find((w) => /gcash/i.test(w.name))?.id ?? phpWallets[0]?.id ?? '',
      })
    } catch (err) {
      if (err instanceof GcashParseError) {
        toast.error(err.message)
      } else {
        toast.error(err instanceof Error ? err.message : 'Could not parse the statement.')
      }
    } finally {
      set({ parsing: false })
    }
  }

  const handleImport = async () => {
    if (!state.statement || state.walletId === '' && !state.createWallet) return
    set({ importing: true })
    try {
      const rows = state.drafts
        .filter((d) => !d.excluded)
        .map((d) => ({
          sourceReference: d.sourceReference,
          occurredAt: d.occurredAt,
          amountCents: d.amountCents,
          kind: d.kind,
          category: d.category,
          description: d.description,
          counterparty: d.counterparty,
          counterWalletId: d.kind === 'transfer_out' || d.kind === 'transfer_in' ? d.counterWalletId || undefined : undefined,
        }))

      const payload: CreateImport = {
        provider: 'gcash_pdf',
        fileFingerprint: state.fingerprint,
        statementFrom: state.statement.rows[0]?.dateTime.slice(0, 10) ?? '',
        statementTo: state.statement.rows[state.statement.rows.length - 1]?.dateTime.slice(0, 10) ?? '',
        walletId: state.createWallet ? undefined : state.walletId,
        createWallet: state.createWallet
          ? { name: state.newWalletName || 'GCash', openingBalanceCents: state.openingBalanceCents }
          : undefined,
        openingBalanceCents: state.openingBalanceCents,
        endingBalanceCents: state.statement.endingBalanceCents ?? 0,
        reconciliation: state.statement.warnings.some((w) => w.code === 'balance-gap') ? 'mismatch' : 'ok',
        rows,
      }

      const result = await createImport.mutateAsync(payload)
      set({ result, step: 'result' })
    } catch {
      // toast handled by the hook
    } finally {
      set({ importing: false })
    }
  }

  /**
   * Run the confidence-gated AI analysis over the current drafts and apply
   * suggestions: preselect (>= 0.90) fields are applied directly, everything
   * else is shown as provenance hints with their confidence scores.
   */
  const handleAnalyze = async () => {
    const rows = state.drafts
      .filter((d) => !d.excluded)
      .slice(0, 50)
      .map((d) => ({
        sourceReference: d.sourceReference,
        description: d.description,
        amountCents: d.amountCents,
        kind: d.kind,
      }))
    if (rows.length === 0) return
    set({ analyzing: true })
    try {
      const result = await runAnalysis.mutateAsync({ scopeId: state.fingerprint, rows })
      applySuggestions(result, state.drafts, phpWallets, (drafts) => set({ drafts }))
      set({ analysis: result })
      toast.success(
        `${result.suggestions.length} suggestion${result.suggestions.length === 1 ? '' : 's'} ready — high-confidence ones were applied; the rest await your review.`,
      )
    } catch {
      // toast handled by the hook
    } finally {
      set({ analyzing: false })
    }
  }

  const suggestionMap = useMemo(() => {
    const map: Record<string, Suggestion[]> = {}
    for (const s of state.analysis?.suggestions ?? []) {
      ;(map[s.targetKey] ??= []).push(s)
    }
    return map
  }, [state.analysis])

  const reset = () => setState(initial)

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2 text-xs text-muted-foreground">
        <ShieldCheck className="h-3.5 w-3.5" />
        Parsing happens on this device — the PDF and password never leave your browser.
      </div>

      {/* Step indicators */}
      <div className="flex items-center gap-2">
        {(['upload', 'wallet', 'review', 'result'] as Step[]).map((s, i) => (
          <div key={s} className="flex items-center gap-2">
            <Badge variant={state.step === s ? 'default' : 'outline'} className="uppercase text-[10px]">
              {i + 1}. {s}
            </Badge>
            {i < 3 && <span className="text-muted-foreground/40">—</span>}
          </div>
        ))}
      </div>

      {state.step === 'upload' && (
        <UploadStep
          file={state.file}
          password={state.password}
          parsing={state.parsing}
          onFile={handleFile}
          onPassword={(password) => set({ password })}
          onParse={handleParse}
        />
      )}

      {state.step === 'wallet' && state.statement && (
        <WalletStep
          wallets={phpWallets}
          walletsLoading={walletsLoading}
          walletId={state.walletId}
          createWallet={state.createWallet}
          newWalletName={state.newWalletName}
          openingBalanceCents={state.openingBalanceCents}
          statement={state.statement}
          netCents={netCents}
          onChange={(patch) => set(patch)}
          onBack={() => set({ step: 'upload' })}
          onNext={() => set({ step: 'review' })}
        />
      )}

      {state.step === 'review' && state.statement && (
        <ReviewStep
          drafts={state.drafts}
          wallets={phpWallets}
          statement={state.statement}
          suggestionsByRef={suggestionMap}
          analyzing={state.analyzing}
          onAnalyze={handleAnalyze}
          onDrafts={(drafts) => set({ drafts })}
          onBack={() => set({ step: 'wallet' })}
          onImport={handleImport}
          importing={state.importing}
        />
      )}

      {state.step === 'result' && state.result && (
        <ResultStep batch={state.result} onReset={reset} />
      )}

      <ImportsHistory imports={imports ?? []} onRollback={(id) => rollback.mutate(id)} />

      <AiAnalysisCard />
    </div>
  )
}

/**
 * Apply analysis suggestions to drafts. Fields with calibrated confidence
 * >= 0.90 are preselected (applied directly, still editable); lower scores
 * are recorded as provenance hints shown in the review table.
 */
function applySuggestions(
  result: AnalysisResult,
  drafts: ImportRowDraft[],
  wallets: Wallet[],
  onDrafts: (d: ImportRowDraft[]) => void,
) {
  const walletByName = new Map(wallets.map((w) => [w.name.toLowerCase(), w.id]))
  const next = drafts.map((draft) => {
    const updated = { ...draft }
    const suggestions = result.suggestions.filter((s) => s.targetKey === draft.sourceReference)

    for (const s of suggestions) {
      const preselect = confidenceBucket(s.confidence) === 'preselect'
      if (s.field === 'category') {
        if (preselect) {
          updated.category = s.value
        } else {
          updated.suggestedCategory = s.value
          updated.confidence = s.confidence
          updated.rationale = s.rationale
        }
      } else if (s.field === 'transfer') {
        const walletId = walletByName.get(s.value.toLowerCase())
        if (preselect && walletId) {
          updated.kind = s.value.toLowerCase().includes('wallet') && updated.kind === 'transfer_out'
            ? 'transfer_out'
            : updated.kind === 'income' || updated.kind === 'transfer_in'
              ? 'transfer_in'
              : 'transfer_out'
          updated.counterWalletId = walletId
        } else {
          updated.confidence = Math.max(updated.confidence ?? 0, s.confidence)
          updated.rationale = updated.rationale || s.rationale
        }
      } else if (s.field === 'merchant' && preselect && s.value) {
        updated.description = s.value
      }
    }
    return updated
  })
  onDrafts(next)
}

// ---- upload ----

function UploadStep({
  file,
  password,
  parsing,
  onFile,
  onPassword,
  onParse,
}: {
  file: File | null
  password: string
  parsing: boolean
  onFile: (f: File) => void
  onPassword: (p: string) => void
  onParse: () => void
}) {
  return (
    <div className="space-y-4">
      <label className="flex cursor-pointer flex-col items-center justify-center gap-2 rounded-lg border border-dashed p-8 text-center hover:bg-muted/50">
        <FileUp className="h-8 w-8 text-muted-foreground" />
        <span className="text-sm font-medium">{file ? file.name : 'Choose a GCash statement PDF'}</span>
        {file && <span className="text-xs text-muted-foreground">{(file.size / 1024 / 1024).toFixed(1)} MB</span>}
        <input
          type="file"
          accept="application/pdf,.pdf"
          className="hidden"
          onChange={(e) => {
            const f = e.target.files?.[0]
            if (f) onFile(f)
          }}
        />
      </label>

      <div className="space-y-1.5">
        <Label htmlFor="pdf-password" className="text-xs">Password (if the statement is encrypted)</Label>
        <Input
          id="pdf-password"
          type="password"
          value={password}
          onChange={(e) => onPassword(e.target.value)}
          className="max-w-sm text-sm"
          placeholder="Statement password"
        />
      </div>

      <Button size="sm" onClick={onParse} disabled={!file || parsing}>
        {parsing && <Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" />}
        {parsing ? 'Parsing…' : 'Parse statement'}
      </Button>
    </div>
  )
}

// ---- wallet ----

function WalletStep({
  wallets,
  walletsLoading,
  walletId,
  createWallet,
  newWalletName,
  openingBalanceCents,
  statement,
  netCents,
  onChange,
  onBack,
  onNext,
}: {
  wallets: Wallet[]
  walletsLoading: boolean
  walletId: string
  createWallet: boolean
  newWalletName: string
  openingBalanceCents: number
  statement: GcashParsedStatement
  netCents: number
  onChange: (patch: Partial<{ walletId: string; createWallet: boolean; newWalletName: string; openingBalanceCents: number }>) => void
  onBack: () => void
  onNext: () => void
}) {
  const ending = statement.endingBalanceCents ?? 0
  const expectedBalance = ending
  const statementHasWarnings = statement.warnings.length > 0

  return (
    <div className="space-y-4">
      {statementHasWarnings && (
        <Alert variant="default" className="border-amber-500/50">
          <AlertTriangle className="h-4 w-4 text-amber-600" />
          <AlertTitle className="text-sm">Statement needs attention</AlertTitle>
          <AlertDescription className="text-xs">
            {statement.warnings.length} parser warning{statement.warnings.length === 1 ? '' : 's'} found (see review
            step). Balances may not reconcile.
          </AlertDescription>
        </Alert>
      )}

      <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
        <StatCard label="Transactions" value={String(statement.rows.length)} />
        <StatCard label="Ending balance" value={formatCents(ending)} />
        <StatCard label="Net (selected rows)" value={formatCents(netCents)} />
      </div>

      <div className="space-y-3">
        <div className="flex items-center gap-2">
          <Checkbox
            id="create-wallet"
            checked={createWallet}
            onCheckedChange={(v) => onChange({ createWallet: v === true, walletId: v === true ? '' : walletId })}
          />
          <Label htmlFor="create-wallet" className="text-sm">Create a new GCash wallet for this import</Label>
        </div>

        {createWallet ? (
          <div className="grid max-w-md grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label className="text-xs">Name</Label>
              <Input value={newWalletName} onChange={(e) => onChange({ newWalletName: e.target.value })} className="text-sm" />
            </div>
            <div className="space-y-1.5">
              <Label className="text-xs">Opening balance (PHP)</Label>
              <Input
                type="number"
                min="0"
                step="0.01"
                value={openingBalanceCents / 100}
                onChange={(e) => onChange({ openingBalanceCents: Math.round(parseFloat(e.target.value || '0') * 100) })}
                className="text-sm"
              />
            </div>
          </div>
        ) : (
          <div className="max-w-sm space-y-1.5">
            <Label className="text-xs">Import into wallet</Label>
            {walletsLoading ? (
              <div className="text-xs text-muted-foreground">Loading wallets…</div>
            ) : wallets.length === 0 ? (
              <div className="text-xs text-muted-foreground">No PHP wallets — create one above or in Wallets.</div>
            ) : (
              <Select value={walletId || undefined} onValueChange={(v) => onChange({ walletId: v })}>
                <SelectTrigger className="text-sm"><SelectValue placeholder="Select wallet" /></SelectTrigger>
                <SelectContent>
                  {wallets.map((w) => (
                    <SelectItem key={w.id} value={w.id} className="text-sm">
                      {w.name} — {formatCents(w.balanceCents)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )}
          </div>
        )}

        <p className="text-xs text-muted-foreground">
          Expected balance after import: <span className="font-medium tabular-nums">{formatCents(expectedBalance)}</span>
        </p>
      </div>

      <div className="flex gap-2">
        <Button size="sm" variant="outline" onClick={onBack}>Back</Button>
        <Button size="sm" onClick={onNext} disabled={!createWallet && !walletId}>
          Review rows ({statement.rows.length})
        </Button>
      </div>
    </div>
  )
}

function StatCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border p-3">
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="mt-1 text-lg font-medium tabular-nums">{value}</p>
    </div>
  )
}

// ---- review ----

function ReviewStep({
  drafts,
  wallets,
  statement,
  suggestionsByRef,
  analyzing,
  onAnalyze,
  onDrafts,
  onBack,
  onImport,
  importing,
}: {
  drafts: ImportRowDraft[]
  wallets: Wallet[]
  statement: GcashParsedStatement
  suggestionsByRef: Record<string, Suggestion[]>
  analyzing: boolean
  onAnalyze: () => void
  onDrafts: (d: ImportRowDraft[]) => void
  onBack: () => void
  onImport: () => void
  importing: boolean
}) {
  const [filter, setFilter] = useState<'all' | 'transfers' | 'warnings' | 'excluded'>('all')
  const hasSuggestions = Object.keys(suggestionsByRef).length > 0

  const warningRows = new Set(statement.warnings.map((w) => w.rowIndex))
  const visible = drafts.filter((d) => {
    if (filter === 'transfers') return d.kind === 'transfer_out' || d.kind === 'transfer_in'
    if (filter === 'excluded') return d.excluded
    return true
  })

  const update = (index: number, patch: Partial<ImportRowDraft>) => {
    const next = drafts.map((d, i) => (i === index ? { ...d, ...patch } : d))
    onDrafts(next)
  }

  const selected = drafts.filter((d) => !d.excluded)

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-2">
        {(['all', 'transfers', 'excluded'] as const).map((f) => (
          <Button
            key={f}
            size="sm"
            variant={filter === f ? 'default' : 'outline'}
            className="h-7 px-2 text-xs capitalize"
            onClick={() => setFilter(f)}
          >
            {f} ({f === 'all' ? drafts.length : f === 'transfers' ? drafts.filter((d) => d.kind.includes('transfer')).length : drafts.filter((d) => d.excluded).length})
          </Button>
        ))}
        <Button
          size="sm"
          variant="outline"
          className="h-7 px-2 text-xs"
          onClick={onAnalyze}
          disabled={analyzing || selected.length === 0}
          title="Ask the configured LLM provider to suggest categories, merchants, and transfer mappings for the first 50 rows."
        >
          {analyzing ? <Loader2 className="mr-1 h-3 w-3 animate-spin" /> : <Bot className="mr-1 h-3 w-3" />}
          {analyzing ? 'Analyzing…' : hasSuggestions ? 'Re-analyze' : 'Analyze with AI'}
        </Button>
        <span className="ml-auto text-xs text-muted-foreground">
          {selected.length} of {drafts.length} rows will import ·{' '}
          {formatCents(selected.reduce((s, d) => s + (d.kind === 'expense' || d.kind === 'transfer_out' ? -d.amountCents : d.amountCents), 0))}
        </span>
      </div>

      {hasSuggestions && (
        <p className="text-xs text-muted-foreground">
          <Bot className="mr-1 inline h-3 w-3" />
          AI suggestions are preselect-only: green badges were applied and remain editable; amber badges need your
          review.
        </p>
      )}

      {statement.warnings.length > 0 && (
        <Alert variant="default" className="border-amber-500/50">
          <AlertTriangle className="h-4 w-4 text-amber-600" />
          <AlertTitle className="text-sm">Parser warnings</AlertTitle>
          <AlertDescription className="text-xs space-y-0.5">
            {statement.warnings.map((w, i) => (
              <p key={i}>• {w.message}</p>
            ))}
          </AlertDescription>
        </Alert>
      )}

      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-8"></TableHead>
              <TableHead className="text-xs">Date</TableHead>
              <TableHead className="text-xs">Description</TableHead>
              <TableHead className="text-xs">Type</TableHead>
              <TableHead className="text-xs">Category</TableHead>
              {drafts.some((d) => d.kind.includes('transfer')) && <TableHead className="text-xs">Counter wallet</TableHead>}
              <TableHead className="text-xs text-right">Amount</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {visible.map((draft, i) => {
              const originalIndex = drafts.indexOf(draft)
              const isTransfer = draft.kind === 'transfer_out' || draft.kind === 'transfer_in'
              return (
                <TableRow key={`${draft.sourceReference}-${i}`} className={draft.excluded ? 'opacity-50' : ''}>
                  <TableCell>
                    <Checkbox
                      checked={!draft.excluded}
                      onCheckedChange={(v) => update(originalIndex, { excluded: v !== true })}
                    />
                  </TableCell>
                  <TableCell className="text-xs tabular-nums text-muted-foreground whitespace-nowrap">
                    {draft.occurredAt.slice(0, 16).replace('T', ' ')}
                  </TableCell>
                  <TableCell className="min-w-[180px]">
                    <Input
                      value={draft.description}
                      onChange={(e) => update(originalIndex, { description: e.target.value })}
                      className="h-7 text-xs"
                    />
                    {draft.suggestedKind && draft.suggestedKind !== draft.kind && (
                      <p className="mt-0.5 text-[10px] text-muted-foreground">
                        suggested: {KIND_LABELS[draft.suggestedKind]}
                      </p>
                    )}
                    {draft.suggestedCategory && draft.suggestedCategory !== draft.category && (
                      <p className="mt-0.5 text-[10px] text-muted-foreground">
                        suggested: {draft.suggestedCategory}
                      </p>
                    )}
                    <SuggestionBadges suggestions={suggestionsByRef[draft.sourceReference] ?? []} />
                  </TableCell>
                  <TableCell>
                    <Select
                      value={draft.kind}
                      onValueChange={(v) => update(originalIndex, { kind: v as ImportRowDraft['kind'] })}
                    >
                      <SelectTrigger className="h-7 text-xs w-[120px]"><SelectValue /></SelectTrigger>
                      <SelectContent>
                        {(Object.keys(KIND_LABELS) as DraftKind[]).map((k) => (
                          <SelectItem key={k} value={k} className="text-xs">{KIND_LABELS[k]}</SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </TableCell>
                  <TableCell>
                    <Input
                      value={draft.category}
                      onChange={(e) => update(originalIndex, { category: e.target.value })}
                      className="h-7 text-xs w-[130px]"
                      list="import-categories"
                    />
                  </TableCell>
                  {isTransfer && (
                    <TableCell>
                      <Select
                        value={draft.counterWalletId || undefined}
                        onValueChange={(v) => update(originalIndex, { counterWalletId: v })}
                      >
                        <SelectTrigger className="h-7 text-xs w-[140px]"><SelectValue placeholder="Map wallet" /></SelectTrigger>
                        <SelectContent>
                          {wallets.map((w) => (
                            <SelectItem key={w.id} value={w.id} className="text-xs">{w.name}</SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </TableCell>
                  )}
                  <TableCell className="text-right text-xs tabular-nums">
                    {draft.kind === 'expense' || draft.kind === 'transfer_out' ? '-' : '+'}
                    {formatCents(draft.amountCents)}
                  </TableCell>
                </TableRow>
              )
            })}
          </TableBody>
        </Table>
      </div>

      <datalist id="import-categories">
        {['Food', 'Transport', 'Shopping', 'Utilities', 'Rent', 'Health', 'Subscriptions', 'Load & Data', 'Income', 'Transfer', 'Unclassified'].map((c) => (
          <option key={c} value={c} />
        ))}
      </datalist>

      {warningRows.size > 0 && (
        <p className="text-xs text-muted-foreground">
          {warningRows.size} row{warningRows.size === 1 ? '' : 's'} flagged by the parser — review them before importing.
        </p>
      )}

      <div className="flex gap-2">
        <Button size="sm" variant="outline" onClick={onBack}>Back</Button>
        <Button size="sm" onClick={onImport} disabled={importing || selected.length === 0}>
          {importing && <Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" />}
          {importing ? 'Importing…' : `Import ${selected.length} row${selected.length === 1 ? '' : 's'}`}
        </Button>
      </div>
    </div>
  )
}

// ---- result ----

function ResultStep({ batch, onReset }: { batch: ImportBatch; onReset: () => void }) {
  const s = batch.summary
  return (
    <div className="space-y-4">
      <Alert variant="default" className={s.errors > 0 ? 'border-amber-500/50' : 'border-green-600/40'}>
        {s.errors > 0 ? <AlertTriangle className="h-4 w-4 text-amber-600" /> : <CheckCircle2 className="h-4 w-4 text-green-600" />}
        <AlertTitle className="text-sm">
          {batch.status === 'rolled_back' ? 'Import rolled back' : s.replay ? 'Already imported' : 'Import complete'}
        </AlertTitle>
        <AlertDescription className="text-xs">
          {s.transactions} transactions · {s.transfers} transfers · {s.duplicates} duplicates skipped · {s.excluded} excluded
          {s.errors > 0 && ` · ${s.errors} errors`}
        </AlertDescription>
      </Alert>

      <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
        <StatCard label="Income" value={formatCents(s.incomeCents)} />
        <StatCard label="Expenses" value={formatCents(s.expenseCents)} />
        <StatCard label="Statement range" value={`${batch.statementFrom} → ${batch.statementTo}`} />
        <StatCard label="Reconciliation" value={batch.reconciliation} />
      </div>

      <Button size="sm" onClick={onReset}>Import another statement</Button>
    </div>
  )
}

// ---- history ----

function ImportsHistory({ imports, onRollback }: { imports: ImportBatch[]; onRollback: (id: string) => void }) {
  if (imports.length === 0) return null
  return (
    <div className="space-y-2 pt-2">
      <h3 className="text-sm font-medium">Previous imports</h3>
      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="text-xs">Date</TableHead>
              <TableHead className="text-xs">Range</TableHead>
              <TableHead className="text-xs text-right">Rows</TableHead>
              <TableHead className="text-xs">Status</TableHead>
              <TableHead className="w-20"></TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {imports.map((b) => (
              <TableRow key={b.id}>
                <TableCell className="text-xs text-muted-foreground">{b.createdAt.slice(0, 10)}</TableCell>
                <TableCell className="text-xs text-muted-foreground">{b.statementFrom} → {b.statementTo}</TableCell>
                <TableCell className="text-right text-xs tabular-nums">{b.summary.imported}</TableCell>
                <TableCell>
                  <Badge variant={b.status === 'rolled_back' ? 'outline' : 'default'} className="text-[10px]">
                    {b.status === 'rolled_back' ? 'rolled back' : 'imported'}
                  </Badge>
                </TableCell>
                <TableCell className="text-right">
                  {b.status !== 'rolled_back' && (
                    <Button
                      size="sm"
                      variant="ghost"
                      className="h-6 text-xs text-destructive hover:text-destructive"
                      onClick={() => {
                        if (confirm(`Undo this import? ${b.summary.imported} row(s) will be removed.`)) {
                          onRollback(b.id)
                        }
                      }}
                    >
                      Undo
                    </Button>
                  )}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  )
}

function SuggestionBadges({ suggestions }: { suggestions: Suggestion[] }) {
  if (suggestions.length === 0) return null
  return (
    <div className="mt-0.5 flex flex-wrap gap-1">
      {suggestions.map((s) => {
        const bucket = confidenceBucket(s.confidence)
        const cls =
          bucket === 'preselect'
            ? 'border-green-600/40 text-green-700'
            : bucket === 'review'
              ? 'border-amber-500/50 text-amber-700'
              : 'border-muted text-muted-foreground'
        return (
          <span
            key={s.id}
            title={`${s.field}: ${s.value} — ${s.rationale ?? ''} (${s.evidence.map((e) => e.source).join(', ')})`}
            className={`rounded border px-1 py-px text-[10px] ${cls}`}
          >
            {s.field === 'transfer' ? '↔' : s.field === 'merchant' ? '◈' : '#'} {s.value} · {Math.round(s.confidence * 100)}%
          </span>
        )
      })}
    </div>
  )
}

function netOf(drafts: ImportRowDraft[]): number {
  return drafts.reduce((sum, d) => {
    if (d.excluded) return sum
    const signed = d.kind === 'expense' || d.kind === 'transfer_out' ? -d.amountCents : d.amountCents
    return sum + signed
  }, 0)
}
