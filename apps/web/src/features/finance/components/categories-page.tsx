import { useState } from 'react'
import { motion } from 'motion/react'
import { Loader2 } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { useCategories, useUpdateCategory } from '../hooks/use-categories'
import { useMotionPreset } from '@/shared/lib/motion'
import type { Category, Classification } from '../schemas/category.schemas'

const CLASSIFICATION_LABELS: Record<Classification, string> = {
  needs: 'Needs',
  wants: 'Wants',
  savings: 'Savings',
  income: 'Income',
  debt: 'Debt',
  other: 'Other',
  unclassified: 'Unclassified',
}

const CLASSIFICATION_ORDER: Classification[] = [
  'needs',
  'wants',
  'savings',
  'income',
  'debt',
  'other',
  'unclassified',
]

function CategoryRow({ category }: { category: Category }) {
  const update = useUpdateCategory()
  const [classification, setClassification] = useState<Classification>(category.classification)
  const [essential, setEssential] = useState(category.essential)
  const [active, setActive] = useState(category.active)
  const preset = useMotionPreset()

  const dirty =
    classification !== category.classification ||
    essential !== category.essential ||
    active !== category.active

  const handleSave = () => {
    update.mutate({
      name: category.name,
      data: { classification, essential, active },
    })
  }

  return (
    <motion.tr
      initial={preset.item.initial}
      animate={preset.item.animate}
      className="border-b transition-colors hover:bg-muted/50 has-aria-expanded:bg-muted/50 data-[state=selected]:bg-muted"
    >
      <TableCell className="font-medium">{category.name}</TableCell>
      <TableCell>
        <Select value={classification} onValueChange={(v) => setClassification(v as Classification)}>
          <SelectTrigger className="h-8 w-36 text-xs">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {CLASSIFICATION_ORDER.map((c) => (
              <SelectItem key={c} value={c}>
                {CLASSIFICATION_LABELS[c]}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </TableCell>
      <TableCell>
        <div className="flex items-center gap-2">
          <Switch checked={essential} onCheckedChange={setEssential} aria-label={`${category.name} essential`} />
          <span className="text-xs text-muted-foreground">Essential</span>
        </div>
      </TableCell>
      <TableCell>
        <div className="flex items-center gap-2">
          <Switch checked={active} onCheckedChange={setActive} aria-label={`${category.name} active`} />
          <span className="text-xs text-muted-foreground">Active</span>
        </div>
      </TableCell>
      <TableCell className="text-right">
        <Button
          size="sm"
          variant="outline"
          disabled={!dirty || update.isPending}
          onClick={handleSave}
        >
          {update.isPending ? 'Saving…' : 'Save'}
        </Button>
      </TableCell>
    </motion.tr>
  )
}

export function CategoriesPage() {
  const { data: categories, isLoading, isError } = useCategories()

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (isError) {
    return (
      <div className="py-12 text-center text-sm text-muted-foreground">
        Failed to load categories.
      </div>
    )
  }

  const unclassifiedCount = (categories ?? []).filter(
    (c) => c.classification === 'unclassified',
  ).length

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">Categories</CardTitle>
        <p className="text-xs text-muted-foreground">
          Classify each category as needs, wants, savings, income, debt, or other. Essential
          marks a category as required spending. Unclassified categories are treated as
          non-essential by analytics.
        </p>
      </CardHeader>
      <CardContent>
        <div className="mb-4 text-xs text-muted-foreground">
          {unclassifiedCount} of {(categories ?? []).length} categories unclassified
        </div>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Classification</TableHead>
              <TableHead>Essential</TableHead>
              <TableHead>Active</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {(categories ?? []).map((category) => (
              <CategoryRow key={category.name} category={category} />
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  )
}