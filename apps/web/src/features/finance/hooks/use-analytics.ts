import { useQuery } from '@tanstack/react-query'
import {
  getAffordability as getAffordabilityApi,
  getAnomalies as getAnomaliesApi,
  getBillReconciliation as getBillReconciliationApi,
  getBudgetHealth as getBudgetHealthApi,
  getCashFlowSummary as getCashFlowSummaryApi,
  getCategoryTrend as getCategoryTrendApi,
  getEmergencyFund as getEmergencyFundApi,
  getGoalHealth as getGoalHealthApi,
  getMonthlyDigest as getMonthlyDigestApi,
  getRecurringCharges as getRecurringChargesApi,
  getSavingsRate as getSavingsRateApi,
  getSpendingSummary as getSpendingSummaryApi,
} from '../api/analytics.api'
import { financeKeys } from '../api/finance.keys'

const staleTime = 1000 * 60

export function useSpendingSummary(from?: string, to?: string) {
  return useQuery({
    queryKey: financeKeys.analyticsQuery('spending-summary', { from, to }),
    queryFn: () => getSpendingSummaryApi(from, to),
    staleTime,
  })
}

export function useCashFlowSummary(from?: string, to?: string) {
  return useQuery({
    queryKey: financeKeys.analyticsQuery('cash-flow-summary', { from, to }),
    queryFn: () => getCashFlowSummaryApi(from, to),
    staleTime,
  })
}

export function useCategoryTrend(category: string, currency: string, months = 6) {
  return useQuery({
    queryKey: financeKeys.analyticsQuery('category-trend', { category, currency, months }),
    queryFn: () => getCategoryTrendApi(category, currency, months),
    staleTime,
  })
}

export function useBudgetHealth(month?: string) {
  return useQuery({
    queryKey: financeKeys.analyticsQuery('budget-health', { month }),
    queryFn: () => getBudgetHealthApi(month),
    staleTime,
  })
}

export function useGoalHealth() {
  return useQuery({
    queryKey: financeKeys.analyticsQuery('goal-health'),
    queryFn: getGoalHealthApi,
    staleTime,
  })
}

export function useSavingsRate(from?: string, to?: string) {
  return useQuery({
    queryKey: financeKeys.analyticsQuery('savings-rate', { from, to }),
    queryFn: () => getSavingsRateApi(from, to),
    staleTime,
  })
}

export function useAnomalies(currency: string, months = 6) {
  return useQuery({
    queryKey: financeKeys.analyticsQuery('anomalies', { currency, months }),
    queryFn: () => getAnomaliesApi(currency, months),
    staleTime,
  })
}

export function useRecurringCharges(currency: string, months = 6) {
  return useQuery({
    queryKey: financeKeys.analyticsQuery('recurring-charges', { currency, months }),
    queryFn: () => getRecurringChargesApi(currency, months),
    staleTime,
  })
}

export function useBillReconciliation(month?: string) {
  return useQuery({
    queryKey: financeKeys.analyticsQuery('bill-reconciliation', { month }),
    queryFn: () => getBillReconciliationApi(month),
    staleTime,
  })
}

export function useEmergencyFund(currency: string, targetMonths = 0) {
  return useQuery({
    queryKey: financeKeys.analyticsQuery('emergency-fund', { currency, targetMonths }),
    queryFn: () => getEmergencyFundApi(currency, targetMonths),
    staleTime,
  })
}

export function useAffordability(currency: string, amountCents: number) {
  return useQuery({
    queryKey: financeKeys.analyticsQuery('affordability', { currency, amountCents }),
    queryFn: () => getAffordabilityApi(currency, amountCents),
    staleTime,
  })
}

export function useMonthlyDigest(month?: string) {
  return useQuery({
    queryKey: financeKeys.analyticsQuery('monthly-digest', { month }),
    queryFn: () => getMonthlyDigestApi(month),
    staleTime,
  })
}