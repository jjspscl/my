import type { ReactNode } from 'react'
import { AlertCircle } from 'lucide-react'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { ApiError } from '@/shared/api/client'
import { UnclassifiedNotice } from './unclassified-notice'

interface SectionCardProps {
  title: string
  loading?: boolean
  error?: Error | null
  isEmpty?: boolean
  emptyMessage?: string
  children: ReactNode
}

export function SectionCard({
  title,
  loading,
  error,
  isEmpty,
  emptyMessage = 'Not enough history yet',
  children,
}: SectionCardProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm font-medium">{title}</CardTitle>
      </CardHeader>
      <CardContent>
        {loading ? (
          <div className="space-y-2">
            <Skeleton className="h-4 w-3/4" />
            <Skeleton className="h-4 w-1/2" />
            <Skeleton className="h-4 w-2/3" />
          </div>
        ) : error ? (
          error instanceof ApiError && error.status === 422 ? (
            <UnclassifiedNotice message={error.message} />
          ) : (
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <AlertCircle className="h-4 w-4" />
              Failed to load
            </div>
          )
        ) : isEmpty ? (
          <p className="text-sm text-muted-foreground">{emptyMessage}</p>
        ) : (
          children
        )}
      </CardContent>
    </Card>
  )
}