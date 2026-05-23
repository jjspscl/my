# UI Style Guide

## Design Direction

- Minimal, monochrome
- White/near-white canvas
- Black/neutral text
- Thin borders and dividers
- Compact panels
- No heavy shadows, gradients, glassmorphism, or neon

## CSS Variables (via @theme)

All colors use HSL CSS variables defined in `globals.css`.

## Utility Classes

| Pattern | Usage |
|---|---|
| `border border-border` | Thin borders |
| `divide-y` | Row separators |
| `rounded-md` | Moderate corners |
| `shadow-none` | No shadows |
| `bg-background` | Page background |
| `bg-card` | Card background |
| `text-foreground` | Primary text |
| `text-muted-foreground` | Secondary text |
| `font-mono tabular-nums` | Financial numbers |
| `text-sm` / `text-xs` | Compact text |

## Component Patterns

### Widget Card
```tsx
<Card className="rounded-md border border-border bg-card shadow-none">
  <CardHeader className="border-b px-4 py-3">
    <CardTitle className="text-sm font-medium tracking-tight">Title</CardTitle>
  </CardHeader>
  <CardContent className="p-4">{children}</CardContent>
</Card>
```

### Metric Row
```tsx
<div className="flex items-center justify-between border-t py-1 text-xs">
  <span className="text-muted-foreground">Label</span>
  <span className="font-mono tabular-nums">₱12,500</span>
</div>
```

## Responsive

- **Mobile**: bottom nav, single column, sheets/drawers for forms
- **Tablet**: two-column grid, side sheets
- **Desktop**: sidebar nav, multi-column grid, command palette