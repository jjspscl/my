import { createFileRoute } from '@tanstack/react-router'

import { AiSettings } from '@/features/intelligence/components/ai-settings'

export const Route = createFileRoute('/_authenticated/settings')({
  component: SettingsPage,
})

function SettingsPage() {
  return <AiSettings />
}
