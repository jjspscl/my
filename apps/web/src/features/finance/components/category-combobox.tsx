import { forwardRef, useState } from 'react'
import { Check, ChevronsUpDown } from 'lucide-react'

import { Button } from '@/components/ui/button'
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { cn } from '@/shared/lib/utils'
import { useCategories } from '../hooks/use-categories'

interface CategoryComboboxProps {
  value: string
  onChange: (value: string) => void
  placeholder?: string
  id?: string
  className?: string
}

export const CategoryCombobox = forwardRef<HTMLButtonElement, CategoryComboboxProps>(
  function CategoryCombobox({ value, onChange, placeholder, id, className }, ref) {
  const [open, setOpen] = useState(false)
  // Categories come from the server (seeded from the presets plus every
  // distinct category already used); the hardcoded preset list is no longer
  // the source of truth.
  const { data: categories } = useCategories()
  const categoryNames = (categories ?? []).filter((c) => c.active).map((c) => c.name)

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild id={id} className={className}>
        <Button
          ref={ref}
          variant="outline"
          role="combobox"
          aria-expanded={open}
          className="w-full justify-between"
          size="sm"
        >
          {value || placeholder || 'Select category'}
          <ChevronsUpDown className="ml-2 h-3 w-3 shrink-0 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-[200px] p-0">
        <Command>
          <CommandInput placeholder="Search or add..." />
          <CommandList>
            <CommandEmpty
              onClick={() => {
                // get the current input value from the command input
                const input = document.querySelector<HTMLInputElement>('[cmdk-input]')
                if (input?.value) {
                  onChange(input.value)
                  setOpen(false)
                }
              }}
              className="cursor-pointer text-xs text-muted-foreground py-2 text-center hover:bg-muted"
            >
              Add &ldquo;<span data-cmdk-input-value></span>&rdquo;
            </CommandEmpty>
            <CommandGroup>
              {categoryNames.map((cat) => (
                <CommandItem
                  key={cat}
                  value={cat}
                  onSelect={(currentValue) => {
                    onChange(currentValue === value ? '' : currentValue)
                    setOpen(false)
                  }}
                >
                  <Check
                    className={cn(
                      'mr-2 h-3 w-3',
                      value === cat ? 'opacity-100' : 'opacity-0',
                    )}
                  />
                  {cat}
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  )
})