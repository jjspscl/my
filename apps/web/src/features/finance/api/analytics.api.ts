import { apiClient } from '@/shared/api/client'
import { z } from 'zod'
import {
  AffordabilitySchema,
  AnomalyReportSchema,
  BillReconciliationSchema,
  BudgetHealthSchema,
  CashFlowSummarySchema,
  CategoryTrendSchema,
  EmergencyFundSchema,
  GoalHealthSchema,
  MonthlyDigestSchema,
  RecurringChargesSummarySchema,
  SavingsRateSchema,
  SpendingSummarySchema,
} from '../schemas/analytics.schemas'

const Data = <T extends z.ZodTypeAny>(schema: T) => z.object({ data: schema })

function qs(params: Record<string, string | number | undefined>): string {
  const search = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== '') search.set(key, String(value))
  }
  const s = search.toString()
  return s ? `?${s}` : ''
}

const base = '/api/v1/finance/analytics'

export async function getSpendingSummary(from?: string, to?: string) {
  const res = await apiClient(`${base}/spending${qs({ from, to })}`, Data(SpendingSummarySchema))
  return res.data
}

export async function getCashFlowSummary(from?: string, to?: string) {
  const res = await apiClient(`${base}/cash-flow${qs({ from, to })}`, Data(CashFlowSummarySchema))
  return res.data
}

export async function getCategoryTrend(category: string, currency: string, months = 6) {
  const res = await apiClient(
    `${base}/category-trend${qs({ category, currency, months })}`,
    Data(CategoryTrendSchema),
  )
  return res.data
}

export async function getBudgetHealth(month?: string) {
  const res = await apiClient(`${base}/budget-health${qs({ month })}`, Data(BudgetHealthSchema))
  return res.data
}

export async function getGoalHealth() {
  const res = await apiClient(`${base}/goal-health`, Data(GoalHealthSchema))
  return res.data
}

export async function getSavingsRate(from?: string, to?: string) {
  const res = await apiClient(`${base}/savings-rate${qs({ from, to })}`, Data(z.array(SavingsRateSchema)))
  return res.data
}

export async function getAnomalies(currency: string, months = 6) {
  const res = await apiClient(`${base}/anomalies${qs({ currency, months })}`, Data(AnomalyReportSchema))
  return res.data
}

export async function getRecurringCharges(currency: string, months = 6) {
  const res = await apiClient(
    `${base}/recurring-charges${qs({ currency, months })}`,
    Data(RecurringChargesSummarySchema),
  )
  return res.data
}

export async function getBillReconciliation(month?: string) {
  const res = await apiClient(
    `${base}/bill-reconciliation${qs({ month })}`,
    Data(BillReconciliationSchema),
  )
  return res.data
}

export async function getEmergencyFund(currency: string, targetMonths = 0) {
  const res = await apiClient(
    `${base}/emergency-fund${qs({ currency, targetMonths })}`,
    Data(EmergencyFundSchema),
  )
  return res.data
}

export async function getAffordability(currency: string, amountCents: number) {
  const res = await apiClient(
    `${base}/affordability${qs({ currency, amountCents })}`,
    Data(AffordabilitySchema),
  )
  return res.data
}

export async function getMonthlyDigest(month?: string) {
  const res = await apiClient(`${base}/digest${qs({ month })}`, Data(MonthlyDigestSchema))
  return res.data
}