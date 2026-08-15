import { createFileRoute } from '@tanstack/react-router'

import { ImportWizard } from '@/features/finance/components/import/import-wizard'

export const Route = createFileRoute('/_authenticated/finance/import')({
  component: ImportWizard,
})
