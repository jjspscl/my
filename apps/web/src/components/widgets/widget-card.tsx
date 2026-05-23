import { createElement, ComponentType } from 'react'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import { cn } from '@/shared/lib/utils'
import { WidgetSize, sizeColSpan } from '@/features/dashboard/lib/widget-registry'
import { WidgetErrorBoundary } from './widget-error-boundary'

interface WidgetCardProps {
  title: string
  size: WidgetSize
  children: React.ReactNode
  action?: React.ReactNode
}

export function WidgetCard({ title, size, children, action }: WidgetCardProps) {
  return (
    <div className={cn(sizeColSpan[size])}>
      <Card className="shadow-none">
        <CardHeader className="flex flex-row items-center justify-between p-4 pb-0">
          <CardTitle className="text-sm font-medium">{title}</CardTitle>
          {action && <div className="flex items-center">{action}</div>}
        </CardHeader>
        <CardContent className="p-4">
          {children}
        </CardContent>
      </Card>
    </div>
  )
}

interface WidgetRendererProps {
  id: string
  title: string
  size: WidgetSize
  component: ComponentType<Record<string, never>>
}

export function WidgetRenderer({ id, title, size, component }: WidgetRendererProps) {
  return (
    <WidgetCard key={id} title={title} size={size}>
      <WidgetErrorBoundary>
        {createElement(component)}
      </WidgetErrorBoundary>
    </WidgetCard>
  )
}