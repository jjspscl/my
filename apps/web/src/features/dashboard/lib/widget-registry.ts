import { z } from 'zod'
import type { ComponentType } from 'react'

export const WidgetSizeSchema = z.enum(['sm', 'md', 'lg', 'full'])
export type WidgetSize = z.infer<typeof WidgetSizeSchema>

export const WidgetManifestSchema = z.object({
  id: z.string(),
  title: z.string(),
  module: z.enum(['finance', 'habits', 'dashboard']),
  defaultSize: WidgetSizeSchema,
})
export type WidgetManifest = z.infer<typeof WidgetManifestSchema>

export const sizeColSpan: Record<WidgetSize, string> = {
  sm: 'col-span-1',
  md: 'col-span-1 md:col-span-2',
  lg: 'col-span-1 md:col-span-2 lg:col-span-3',
  full: 'col-span-full',
}

interface WidgetEntry {
  manifest: WidgetManifest
  component: ComponentType<Record<string, never>>
}

const registry = new Map<string, WidgetEntry>()

export function registerWidget(
  manifest: WidgetManifest,
  component: ComponentType<Record<string, never>>,
): void {
  if (registry.has(manifest.id)) {
    return
  }
  registry.set(manifest.id, { manifest, component })
}

export function getWidgets(): WidgetEntry[] {
  return Array.from(registry.values())
}

export function getWidgetComponent(id: string): ComponentType<Record<string, never>> | undefined {
  return registry.get(id)?.component
}

export function getWidgetsByModule(module: string): WidgetEntry[] {
  return getWidgets().filter((e) => e.manifest.module === module)
}

export function getWidgetManifest(id: string): WidgetManifest | undefined {
  return registry.get(id)?.manifest
}