import { describe, it, expect } from 'vitest'
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
  SavingsRateSchema,
  SpendingSummarySchema,
} from './analytics.schemas'

describe('SpendingSummarySchema', () => {
  it('parses a valid spending summary', () => {
    const summary = SpendingSummarySchema.parse({
      dateRange: { from: '2026-07-01', to: '2026-07-31' },
      currencies: [
        {
          currency: 'PHP',
          totalExpenseCents: 150000,
          byClassification: { needs: 100000, wants: 50000 },
          unclassifiedCents: 0,
        },
      ],
      unclassifiedSharePct: 0,
      assumptions: ['Classified by category metadata'],
    })
    expect(summary.currencies[0]!.totalExpenseCents).toBe(150000)
    expect(summary.currencies[0]!.byClassification.needs).toBe(100000)
  })

  it('rejects missing assumptions', () => {
    const result = SpendingSummarySchema.safeParse({
      dateRange: { from: '2026-07-01', to: '2026-07-31' },
      currencies: [],
      unclassifiedSharePct: 0,
    })
    expect(result.success).toBe(false)
  })
})

describe('CashFlowSummarySchema', () => {
  it('parses a valid cash flow summary with monthly series', () => {
    const flow = CashFlowSummarySchema.parse({
      dateRange: { from: '2026-07-01', to: '2026-07-31' },
      currencies: [
        {
          currency: 'PHP',
          incomeCents: 500000,
          expenseCents: 300000,
          netCents: 200000,
          monthly: [
            { month: '2026-07', currency: 'PHP', incomeCents: 500000, expenseCents: 300000, netCents: 200000 },
          ],
        },
      ],
      assumptions: [],
    })
    expect(flow.currencies[0]!.monthly[0]!.netCents).toBe(200000)
  })
})

describe('CategoryTrendSchema', () => {
  it('parses a valid category trend', () => {
    const trend = CategoryTrendSchema.parse({
      category: 'Food',
      currency: 'PHP',
      months: [{ month: '2026-07', amountCents: 12000 }],
      sampleSize: 6,
      sufficient: true,
      assumptions: [],
    })
    expect(trend.sufficient).toBe(true)
    expect(trend.months[0]!.amountCents).toBe(12000)
  })
})

describe('BudgetHealthSchema', () => {
  it('parses a valid budget health', () => {
    const health = BudgetHealthSchema.parse({
      month: '2026-07',
      currency: 'PHP',
      hasBudget: true,
      totalAllocatedCents: 300000,
      totalSpentCents: 250000,
      totalRemainingCents: 50000,
      unbudgetedSpentCents: 10000,
      categories: [{ category: 'Food', allocatedCents: 100000, spentCents: 80000, remainingCents: 20000 }],
      assumptions: [],
    })
    expect(health.hasBudget).toBe(true)
  })

  it('accepts missing currency (omitempty)', () => {
    const result = BudgetHealthSchema.safeParse({
      month: '2026-07',
      hasBudget: false,
      totalAllocatedCents: 0,
      totalSpentCents: 0,
      totalRemainingCents: 0,
      unbudgetedSpentCents: 0,
      categories: [],
      assumptions: [],
    })
    expect(result.success).toBe(true)
  })
})

describe('GoalHealthSchema', () => {
  it('parses a valid goal health', () => {
    const health = GoalHealthSchema.parse({
      goals: [
        {
          id: 'g1',
          name: 'Emergency fund',
          currency: 'PHP',
          targetAmountCents: 1000000,
          currentAmountCents: 400000,
          remainingAmountCents: 600000,
          progressPercent: 40,
          status: 'on_track',
        },
      ],
      assumptions: [],
    })
    expect(health.goals[0]!.progressPercent).toBe(40)
  })

  it('accepts optional requiredMonthlyCents', () => {
    const result = GoalHealthSchema.safeParse({
      goals: [
        {
          id: 'g1',
          name: 'Emergency fund',
          currency: 'PHP',
          targetAmountCents: 1000000,
          currentAmountCents: 400000,
          remainingAmountCents: 600000,
          progressPercent: 40,
          status: 'on_track',
          requiredMonthlyCents: 50000,
        },
      ],
      assumptions: [],
    })
    expect(result.success).toBe(true)
  })
})

describe('SavingsRateSchema', () => {
  it('parses a valid savings rate', () => {
    const rate = SavingsRateSchema.parse({
      currency: 'PHP',
      incomeCents: 500000,
      expenseCents: 300000,
      netCents: 200000,
      ratePercent: 40,
      zeroIncome: false,
      assumptions: [],
    })
    expect(rate.ratePercent).toBe(40)
  })
})

describe('AnomalyReportSchema', () => {
  it('parses a valid anomaly report', () => {
    const report = AnomalyReportSchema.parse({
      currency: 'PHP',
      months: 6,
      sufficient: true,
      anomalies: [
        {
          category: 'Food',
          currency: 'PHP',
          month: '2026-07',
          amountCents: 30000,
          medianCents: 10000,
          ratio: 3,
          explanation: '3x median',
        },
      ],
      assumptions: [],
    })
    expect(report.anomalies[0]!.ratio).toBe(3)
  })
})

describe('BillReconciliationSchema', () => {
  it('parses a valid bill reconciliation', () => {
    const recon = BillReconciliationSchema.parse({
      month: '2026-07',
      items: [
        {
          billId: 'b1',
          name: 'Netflix',
          category: 'Entertainment',
          currency: 'PHP',
          expectedCents: 1000,
          paidCents: 1000,
          varianceCents: 0,
          paidCount: 1,
          paidWithoutTransactionCount: 0,
          explanation: 'Paid in full',
        },
      ],
      assumptions: [],
    })
    expect(recon.items[0]!.varianceCents).toBe(0)
  })
})

describe('EmergencyFundSchema', () => {
  it('parses targetRangeMonths as a tuple', () => {
    const fund = EmergencyFundSchema.parse({
      currency: 'PHP',
      liquidBalanceCents: 500000,
      monthlyEssentialCents: 100000,
      monthsOfRunway: 5,
      targetRangeMonths: [3, 6],
      shortfallToMinCents: 0,
      shortfallToMaxCents: 100000,
      assumptions: [],
    })
    expect(fund.targetRangeMonths).toEqual([3, 6])
  })
})

describe('AffordabilitySchema', () => {
  it('parses a valid affordability model', () => {
    const model = AffordabilitySchema.parse({
      currency: 'PHP',
      amountCents: 200000,
      liquidBalanceCents: 500000,
      monthlyEssentialCents: 100000,
      upcomingBillsCents: 20000,
      monthlyObligationCents: 120000,
      runwayMonthsBefore: 4.2,
      runwayMonthsAfter: 2.5,
      assumptions: [],
    })
    expect(model.runwayMonthsAfter).toBe(2.5)
  })
})

describe('MonthlyDigestSchema', () => {
  const baseDigest = {
    month: '2026-07',
    cashFlow: { present: true, summary: 'Net +200,000', currencies: [] },
    spending: { present: true, summary: 'Spent 300,000', currencies: [], unclassifiedSharePct: 0 },
    savingsRate: { present: true, summary: '40%', rates: [] },
    recurring: { present: false, summary: '', charges: [] },
    anomalies: { present: false, summary: '', anomalies: [] },
    emergency: { present: false, summary: '' },
    omitted: ['recurring', 'anomalies', 'emergency'],
    assumptions: [],
  }

  it('parses a digest with present sections', () => {
    const digest = MonthlyDigestSchema.parse(baseDigest)
    expect(digest.cashFlow.present).toBe(true)
    expect(digest.recurring.present).toBe(false)
  })

  it('accepts emergency status when present', () => {
    const digest = MonthlyDigestSchema.parse({
      ...baseDigest,
      emergency: {
        present: true,
        summary: '5 months of runway',
        status: {
          currency: 'PHP',
          liquidBalanceCents: 500000,
          monthlyEssentialCents: 100000,
          monthsOfRunway: 5,
          targetRangeMonths: [3, 6],
          shortfallToMinCents: 0,
          shortfallToMaxCents: 100000,
          assumptions: [],
        },
      },
    })
    expect(digest.emergency.status!.monthsOfRunway).toBe(5)
  })

  it('rejects a digest missing a section (sections are never absent)', () => {
    const withoutCashFlow = { ...baseDigest }
    delete withoutCashFlow.cashFlow
    const result = MonthlyDigestSchema.safeParse(withoutCashFlow)
    expect(result.success).toBe(false)
  })
})