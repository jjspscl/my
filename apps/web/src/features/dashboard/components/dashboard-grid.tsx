import { getWidgets } from '../lib/widget-registry'
import { WidgetRenderer } from '@/components/widgets/widget-card'

export function DashboardGrid() {
  const widgets = getWidgets()

  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-4 p-4">
      {widgets.map((entry) => (
        <WidgetRenderer
          key={entry.manifest.id}
          id={entry.manifest.id}
          title={entry.manifest.title}
          size={entry.manifest.defaultSize}
          component={entry.component}
        />
      ))}
    </div>
  )
}