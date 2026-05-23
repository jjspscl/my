import { z } from 'zod'
import { WidgetSizeSchema } from '../lib/widget-registry'

export const WidgetLayoutSchema = z.object({
  widgetId: z.string(),
  position: z.number().int().min(0),
  visible: z.boolean(),
  size: WidgetSizeSchema,
})
export type WidgetLayout = z.infer<typeof WidgetLayoutSchema>

export const DashboardConfigSchema = z.object({
  widgets: z.array(WidgetLayoutSchema),
})
export type DashboardConfig = z.infer<typeof DashboardConfigSchema>