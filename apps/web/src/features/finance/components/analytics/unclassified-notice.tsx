import { Link } from '@tanstack/react-router'
import { AlertTriangle } from 'lucide-react'

interface UnclassifiedNoticeProps {
  message: string
}

/** Renders the server's 422 message verbatim with a link to Categories. */
export function UnclassifiedNotice({ message }: UnclassifiedNoticeProps) {
  return (
    <div className="space-y-2">
      <div className="flex items-start gap-2 text-sm text-muted-foreground">
        <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
        <p>{message}</p>
      </div>
      <Link
        to="/finance/categories"
        className="text-xs font-medium text-foreground underline underline-offset-4"
      >
        Classify categories
      </Link>
    </div>
  )
}