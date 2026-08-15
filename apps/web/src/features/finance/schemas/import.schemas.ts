import { z } from 'zod'

// ---- Parser contracts ----

export const GcashParsedRowSchema = z.object({
  dateTime: z.string(),
  description: z.string(),
  referenceNo: z.string(),
  debit: z.string(),
  credit: z.string(),
  balance: z.string(),
})
export type GcashParsedRow = z.infer<typeof GcashParsedRowSchema>

export const GcashWarningSchema = z.object({
  rowIndex: z.number().int().optional(),
  code: z.enum(['both-sides', 'no-side', 'bad-reference', 'unparseable-date', 'balance-gap']),
  message: z.string(),
})
export type GcashWarning = z.infer<typeof GcashWarningSchema>

export const GcashParsedStatementSchema = z.object({
  rows: z.array(GcashParsedRowSchema),
  totalDebitCents: z.number().int(),
  totalCreditCents: z.number().int(),
  startingBalanceCents: z.number().int().nullable(),
  endingBalanceCents: z.number().int().nullable(),
  warnings: z.array(GcashWarningSchema),
})
export type GcashParsedStatement = z.infer<typeof GcashParsedStatementSchema>

// ---- Review draft contract (client-side, pre-import) ----

export const ImportDraftKindSchema = z.enum(['expense', 'income', 'transfer_out', 'transfer_in'])
export type ImportDraftKind = z.infer<typeof ImportDraftKindSchema>

export const ImportRowDraftSchema = z.object({
  sourceReference: z.string(),
  occurredAt: z.string(), // RFC3339
  amountCents: z.number().int().positive(),
  kind: ImportDraftKindSchema,
  category: z.string(),
  description: z.string(),
  counterparty: z.string().optional(),
  counterWalletId: z.string().optional(),
  excluded: z.boolean().default(false),
  // Parser/LLM provenance shown in the review table.
  suggestedKind: ImportDraftKindSchema.optional(),
  suggestedCategory: z.string().optional(),
  confidence: z.number().min(0).max(1).optional(),
  rationale: z.string().optional(),
})
export type ImportRowDraft = z.infer<typeof ImportRowDraftSchema>

// ---- Create import (backend contract) ----

// Transfers must carry a counter wallet; income/expense must NOT carry one.
// The API rejects both violations; the contract catches them client-side.
const CreateImportRowBase = z.object({
  sourceReference: z.string(),
  occurredAt: z.string(),
  amountCents: z.number().int().positive(),
  kind: ImportDraftKindSchema,
  category: z.string(),
  description: z.string(),
  counterparty: z.string().optional(),
  counterWalletId: z.string().optional(),
})

export const CreateImportRowSchema = CreateImportRowBase.superRefine((value, ctx) => {
  const isTransfer = value.kind === 'transfer_out' || value.kind === 'transfer_in'
  if (isTransfer && !value.counterWalletId) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      message: 'transfer rows require a counter wallet',
      path: ['counterWalletId'],
    })
  }
  if (!isTransfer && value.counterWalletId) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      message: 'non-transfer rows cannot carry a counter wallet',
      path: ['counterWalletId'],
    })
  }
})
export type CreateImportRow = z.infer<typeof CreateImportRowSchema>

const CreateImportBase = z.object({
  provider: z.literal('gcash_pdf'),
  fileFingerprint: z.string().regex(/^[0-9a-f]{64}$/i, 'Invalid file fingerprint'),
  statementFrom: z.string(),
  statementTo: z.string(),
  walletId: z.string().optional(),
  createWallet: z
    .object({
      name: z.string().min(1),
      openingBalanceCents: z.number().int().min(0),
    })
    .optional(),
  openingBalanceCents: z.number().int().min(0),
  endingBalanceCents: z.number().int().min(0),
  reconciliation: z.enum(['ok', 'mismatch', 'unknown']),
  rows: z.array(CreateImportRowSchema).max(2000),
})

// Exactly one target must be selected: an existing wallet or a new one.
export const CreateImportSchema = CreateImportBase.superRefine((value, ctx) => {
  if (!value.walletId && !value.createWallet) {
    ctx.addIssue({
      code: z.ZodIssueCode.custom,
      message: 'select a wallet or create a new one',
      path: ['walletId'],
    })
  }
})
export type CreateImport = z.infer<typeof CreateImportSchema>

export const ImportSummarySchema = z.object({
  total: z.number().int(),
  imported: z.number().int(),
  duplicates: z.number().int(),
  excluded: z.number().int(),
  errors: z.number().int(),
  transactions: z.number().int(),
  transfers: z.number().int(),
  incomeCents: z.number().int(),
  expenseCents: z.number().int(),
  replay: z.boolean(),
})
export type ImportSummary = z.infer<typeof ImportSummarySchema>

export const ImportEntrySchema = z.object({
  id: z.string(),
  sourceReference: z.string(),
  occurredAt: z.string(),
  amountCents: z.number().int(),
  kind: ImportDraftKindSchema,
  category: z.string(),
  description: z.string(),
  counterparty: z.string().optional(),
  counterWalletId: z.string().optional(),
  outcome: z.string(),
  entityType: z.string().optional(),
  entityId: z.string().optional(),
})
export type ImportEntry = z.infer<typeof ImportEntrySchema>

export const ImportBatchSchema = z.object({
  id: z.string(),
  provider: z.string(),
  fileFingerprint: z.string(),
  statementFrom: z.string(),
  statementTo: z.string(),
  walletId: z.string().optional(),
  createdWalletId: z.string().optional(),
  openingBalanceCents: z.number().int(),
  endingBalanceCents: z.number().int(),
  reconciliation: z.string(),
  status: z.string(),
  summary: ImportSummarySchema,
  createdAt: z.string(),
  rolledBackAt: z.string().optional(),
  entries: z.array(ImportEntrySchema).optional(),
})
export type ImportBatch = z.infer<typeof ImportBatchSchema>

export const ImportBatchListSchema = z.object({
  data: z.array(ImportBatchSchema),
})
export type ImportBatchList = z.infer<typeof ImportBatchListSchema>
