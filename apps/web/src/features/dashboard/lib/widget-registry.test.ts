import { describe, it, expect, beforeEach } from 'vitest'
import { registerWidget, getWidgets, getWidgetComponent, getWidgetsByModule, getWidgetManifest, type WidgetManifest } from './widget-registry'

// Use counter to generate unique IDs per test to avoid registry pollution
let testCounter = 0
function uniqueId(prefix: string): string {
  testCounter++
  return `${prefix}-${testCounter}`
}

describe('widget registry', () => {
  beforeEach(() => {
    testCounter = 0
  })

  it('registerWidget stores entry', () => {
    const id = uniqueId('test-widget')
    registerWidget({ id, title: 'Test Widget', module: 'dashboard', defaultSize: 'sm' }, () => null)
    const widgets = getWidgets()
    expect(widgets.some(w => w.manifest.id === id)).toBe(true)
  })

  it('registerWidget same id twice is idempotent', () => {
    const id = uniqueId('idempotent')
    const comp = () => null
    const manifest: WidgetManifest = { id, title: 'Test', module: 'dashboard', defaultSize: 'sm' }
    registerWidget(manifest, comp)
    registerWidget(manifest, comp)
    const widgets = getWidgets()
    const matches = widgets.filter(w => w.manifest.id === id)
    expect(matches).toHaveLength(1)
  })

  it('getWidgets returns all registered entries', () => {
    const id1 = uniqueId('w1')
    const id2 = uniqueId('w2')
    registerWidget({ id: id1, title: 'W1', module: 'dashboard', defaultSize: 'sm' }, () => null)
    registerWidget({ id: id2, title: 'W2', module: 'finance', defaultSize: 'md' }, () => null)

    const widgets = getWidgets()
    const ids = widgets.map(w => w.manifest.id)
    expect(ids).toContain(id1)
    expect(ids).toContain(id2)
  })

  it('getWidgetComponent returns component for known id', () => {
    const id = uniqueId('comp')
    const TestComponent = () => null
    registerWidget({ id, title: 'Test', module: 'dashboard', defaultSize: 'sm' }, TestComponent)
    const comp = getWidgetComponent(id)
    expect(comp).toBe(TestComponent)
  })

  it('getWidgetComponent returns undefined for unknown id', () => {
    const comp = getWidgetComponent('nonexistent')
    expect(comp).toBeUndefined()
  })

  it('getWidgetsByModule filters by module', () => {
    const dashId = uniqueId('dash')
    const finId = uniqueId('fin')
    registerWidget({ id: dashId, title: 'Dashboard', module: 'dashboard', defaultSize: 'sm' }, () => null)
    registerWidget({ id: finId, title: 'Finance', module: 'finance', defaultSize: 'md' }, () => null)

    const dashWidgets = getWidgetsByModule('dashboard')
    const dashIds = dashWidgets.map(w => w.manifest.id)
    expect(dashIds).toContain(dashId)
    expect(dashIds).not.toContain(finId)

    const finWidgets = getWidgetsByModule('finance')
    const finIds = finWidgets.map(w => w.manifest.id)
    expect(finIds).toContain(finId)
    expect(finIds).not.toContain(dashId)
  })

  it('getWidgetManifest returns manifest for known id', () => {
    const id = uniqueId('manifest')
    const manifest: WidgetManifest = { id, title: 'Test', module: 'dashboard', defaultSize: 'sm' }
    registerWidget(manifest, () => null)
    const result = getWidgetManifest(id)
    expect(result).toEqual(manifest)
  })

  it('getWidgetManifest returns undefined for unknown id', () => {
    const manifest = getWidgetManifest('nonexistent')
    expect(manifest).toBeUndefined()
  })

  it('WidgetManifest requires id, title, module, defaultSize', () => {
    const id = uniqueId('req')
    registerWidget({ id, title: 'Test', module: 'dashboard', defaultSize: 'sm' }, () => null)
    const widgets = getWidgets()
    const entry = widgets.find(w => w.manifest.id === id)
    expect(entry?.manifest).toHaveProperty('id')
    expect(entry?.manifest).toHaveProperty('title')
    expect(entry?.manifest).toHaveProperty('module')
    expect(entry?.manifest).toHaveProperty('defaultSize')
  })

  it('getWidgetsByModule returns empty array for module with no widgets', () => {
    const result = getWidgetsByModule('habits')
    expect(result).toEqual([])
  })
})