import { z } from 'zod'

// --- Shared primitives ---

export const DateRangeSchema = z.object({
  from: z.string(),
  to: z.string(),
})
export type DateRange = z.infer<typeof DateRangeSchema>

export const AssumptionsSchema = z.array(z.string())

// --- Spending summary ---

export const CurrencySpendingSchema = z.object({
  currency: z.string(),
  totalExpenseCents: z.number().int(),
  byClassification: z.record(z.string(), z.number().int()),
  unclassifiedCents: z.number().int(),
})
export type CurrencySpending = z.infer<typeof CurrencySpendingSchema>

export const SpendingSummarySchema = z.object({
  dateRange: DateRangeSchema,
  currencies: z.array(CurrencySpendingSchema),
  unclassifiedSharePct: z.number(),
  assumptions: AssumptionsSchema,
})
export type SpendingSummary = z.infer<typeof SpendingSummarySchema>

// --- Cash flow summary ---

export const MonthlyCashFlowSchema = z.object({
  month: z.string(),
  currency: z.string(),
  incomeCents: z.number().int(),
  expenseCents: z.number().int(),
  netCents: z.number().int(),
})
export type MonthlyCashFlow = z.infer<typeof MonthlyCashFlowSchema>

export const CurrencyCashFlowSchema = z.object({
  currency: z.string(),
  incomeCents: z.number().int(),
  expenseCents: z.number().int(),
  netCents: z.number().int(),
  monthly: z.array(MonthlyCashFlowSchema),
})
export type CurrencyCashFlow = z.infer<typeof CurrencyCashFlowSchema>

export const CashFlowSummarySchema = z.object({
  dateRange: DateRangeSchema,
  currencies: z.array(CurrencyCashFlowSchema),
  assumptions: AssumptionsSchema,
})
export type CashFlowSummary = z.infer<typeof CashFlowSummarySchema>

// --- Category trend ---

export const CategoryTrendPointSchema = z.object({
  month: z.string(),
  amountCents: z.number().int(),
})
export type CategoryTrendPoint = z.infer<typeof CategoryTrendPointSchema>

export const CategoryTrendSchema = z.object({
  category: z.string(),
  currency: z.string(),
  months: z.array(CategoryTrendPointSchema),
  sampleSize: z.number().int(),
  sufficient: z.boolean(),
  assumptions: AssumptionsSchema,
})
export type CategoryTrend = z.infer<typeof CategoryTrendSchema>

// --- Budget health ---

export const BudgetHealthCategorySchema = z.object({
  category: z.string(),
  allocatedCents: z.number().int(),
  spentCents: z.number().int(),
  remainingCents: z.number().int(),
})
export type BudgetHealthCategory = z.infer<typeof BudgetHealthCategorySchema>

export const BudgetHealthSchema = z.object({
  month: z.string(),
  currency: z.string().optional(),
  hasBudget: z.boolean(),
  totalAllocatedCents: z.number().int(),
  totalSpentCents: z.number().int(),
  totalRemainingCents: z.number().int(),
  unbudgetedSpentCents: z.number().int(),
  categories: z.array(BudgetHealthCategorySchema),
  assumptions: AssumptionsSchema,
})
export type BudgetHealth = z.infer<typeof BudgetHealthSchema>

// --- Goal health ---

export const GoalHealthItemSchema = z.object({
  id: z.string(),
  name: z.string(),
  currency: z.string(),
  targetAmountCents: z.number().int(),
  currentAmountCents: z.number().int(),
  remainingAmountCents: z.number().int(),
  progressPercent: z.number().int(),
  status: z.string(),
  requiredMonthlyCents: z.number().int().optional(),
})
export type GoalHealthItem = z.infer<typeof GoalHealthItemSchema>

export const GoalHealthSchema = z.object({
  goals: z.array(GoalHealthItemSchema),
  assumptions: AssumptionsSchema,
})
export type GoalHealth = z.infer<typeof GoalHealthSchema>

// --- Savings rate ---

export const SavingsRateSchema = z.object({
  currency: z.string(),
  incomeCents: z.number().int(),
  expenseCents: z.number().int(),
  netCents: z.number().int(),
  ratePercent: z.number(),
  zeroIncome: z.boolean(),
  assumptions: AssumptionsSchema,
})
export type SavingsRate = z.infer<typeof SavingsRateSchema>

// --- Anomalies ---

export const AnomalySchema = z.object({
  category: z.string(),
  currency: z.string(),
  month: z.string(),
  amountCents: z.number().int(),
  medianCents: z.number().int(),
  ratio: z.number(),
  explanation: z.string(),
})
export type Anomaly = z.infer<typeof AnomalySchema>

export const AnomalyReportSchema = z.object({
  currency: z.string(),
  months: z.number().int(),
  sufficient: z.boolean(),
  anomalies: z.array(AnomalySchema),
  assumptions: AssumptionsSchema,
})
export type AnomalyReport = z.infer<typeof AnomalyReportSchema>

// --- Recurring charges ---

export const RecurringChargeSchema = z.object({
  category: z.string(),
  currency: z.string(),
  occurrences: z.number().int(),
  distinctMonths: z.number().int(),
  medianCents: z.number().int(),
  status: z.string(),
  billName: z.string().optional(),
  billAmountCents: z.number().int().optional(),
  explanation: z.string(),
})
export type RecurringCharge = z.infer<typeof RecurringChargeSchema>

export const RecurringChargesSummarySchema = z.object({
  currency: z.string(),
  months: z.number().int(),
  charges: z.array(RecurringChargeSchema),
  assumptions: AssumptionsSchema,
})
export type RecurringChargesSummary = z.infer<typeof RecurringChargesSummarySchema>

// --- Bill reconciliation ---

export const BillReconciliationItemSchema = z.object({
  billId: z.string(),
  name: z.string(),
  category: z.string(),
  currency: z.string(),
  expectedCents: z.number().int(),
  paidCents: z.number().int(),
  varianceCents: z.number().int(),
  paidCount: z.number().int(),
  paidWithoutTransactionCount: z.number().int(),
  explanation: z.string(),
})
export type BillReconciliationItem = z.infer<typeof BillReconciliationItemSchema>

export const BillReconciliationSchema = z.object({
  month: z.string(),
  items: z.array(BillReconciliationItemSchema),
  assumptions: AssumptionsSchema,
})
export type BillReconciliation = z.infer<typeof BillReconciliationSchema>

// --- Emergency fund ---

export const EmergencyFundSchema = z.object({
  currency: z.string(),
  liquidBalanceCents: z.number().int(),
  monthlyEssentialCents: z.number().int(),
  monthsOfRunway: z.number(),
  targetRangeMonths: z.tuple([z.number(), z.number()]),
  shortfallToMinCents: z.number().int(),
  shortfallToMaxCents: z.number().int(),
  assumptions: AssumptionsSchema,
})
export type EmergencyFund = z.infer<typeof EmergencyFundSchema>

// --- Affordability ---

export const AffordabilitySchema = z.object({
  currency: z.string(),
  amountCents: z.number().int(),
  liquidBalanceCents: z.number().int(),
  monthlyEssentialCents: z.number().int(),
  upcomingBillsCents: z.number().int(),
  monthlyObligationCents: z.number().int(),
  runwayMonthsBefore: z.number(),
  runwayMonthsAfter: z.number(),
  assumptions: AssumptionsSchema,
})
export type Affordability = z.infer<typeof AffordabilitySchema>

// --- Monthly digest ---

export const DigestCashFlowSchema = z.object({
  present: z.boolean(),
  summary: z.string(),
  currencies: z.array(CurrencyCashFlowSchema),
})
export type DigestCashFlow = z.infer<typeof DigestCashFlowSchema>

export const DigestSpendingSchema = z.object({
  present: z.boolean(),
  summary: z.string(),
  currencies: z.array(CurrencySpendingSchema),
  unclassifiedSharePct: z.number(),
})
export type DigestSpending = z.infer<typeof DigestSpendingSchema>

export const DigestSavingsSchema = z.object({
  present: z.boolean(),
  summary: z.string(),
  rates: z.array(SavingsRateSchema),
})
export type DigestSavings = z.infer<typeof DigestSavingsSchema>

export const DigestRecurringSchema = z.object({
  present: z.boolean(),
  summary: z.string(),
  charges: z.array(RecurringChargeSchema),
})
export type DigestRecurring = z.infer<typeof DigestRecurringSchema>

export const DigestAnomaliesSchema = z.object({
  present: z.boolean(),
  summary: z.string(),
  anomalies: z.array(AnomalySchema),
})
export type DigestAnomalies = z.infer<typeof DigestAnomaliesSchema>

export const DigestEmergencySchema = z.object({
  present: z.boolean(),
  summary: z.string(),
  status: EmergencyFundSchema.optional(),
})
export type DigestEmergency = z.infer<typeof DigestEmergencySchema>

export const MonthlyDigestSchema = z.object({
  month: z.string(),
  cashFlow: DigestCashFlowSchema,
  spending: DigestSpendingSchema,
  savingsRate: DigestSavingsSchema,
  recurring: DigestRecurringSchema,
  anomalies: DigestAnomaliesSchema,
  emergency: DigestEmergencySchema,
  omitted: z.array(z.string()),
  assumptions: AssumptionsSchema,
})
export type MonthlyDigest = z.infer<typeof MonthlyDigestSchema>