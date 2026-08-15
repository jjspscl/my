import { Link } from '@tanstack/react-router'
import { Bot, ChevronRight, Settings } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'

import { useProviders } from '../hooks/use-intelligence'

/**
 * Compact status card shown on the import page. Provider management lives on
 * /settings/intelligence; this card only surfaces availability.
 */
export function AiAnalysisCard() {
  const { data: providers, isLoading } = useProviders()
  const enabled = (providers ?? []).filter((p) => p.enabled && p.hasCredential)
  const active = enabled[0]

  return (
    <div className="flex items-center justify-between gap-3 rounded-lg border p-3">
      <div className="flex items-center gap-2">
        <Bot className="h-4 w-4 text-muted-foreground" />
        <div>
          <p className="text-xs font-medium">AI import analysis</p>
          <p className="text-[11px] text-muted-foreground">
            {isLoading
              ? 'Loading…'
              : active
                ? `Active: ${active.name} (${active.model})`
                : providers && providers.length > 0
                  ? 'No enabled provider with a credential'
                  : 'Not configured'}
          </p>
        </div>
      </div>
      <div className="flex items-center gap-2">
        {active && <Badge variant="default" className="text-[10px]">ready</Badge>}
        <Button asChild size="sm" variant="outline" className="h-7 text-xs">
          <Link to="/settings">
            <Settings className="mr-1 h-3 w-3" /> Configure
            <ChevronRight className="ml-1 h-3 w-3" />
          </Link>
        </Button>
      </div>
    </div>
  )
}
