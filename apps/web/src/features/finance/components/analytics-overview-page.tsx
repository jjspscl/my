import { AnomaliesSection } from './analytics/anomalies-section'
import { BudgetHealthSection } from './analytics/budget-health-section'
import { CashFlowSection } from './analytics/cash-flow-section'
import { EmergencyFundSection } from './analytics/emergency-fund-section'
import { SavingsRateSection } from './analytics/savings-rate-section'
import { SpendingSection } from './analytics/spending-section'

export function AnalyticsOverviewPage() {
  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
      <CashFlowSection />
      <SpendingSection />
      <SavingsRateSection />
      <BudgetHealthSection />
      <EmergencyFundSection />
      <AnomaliesSection />
    </div>
  )
}