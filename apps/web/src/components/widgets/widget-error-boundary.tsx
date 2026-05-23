import { Component } from 'react'
import { Button } from '@/components/ui/button'
import { AlertTriangle } from 'lucide-react'

interface Props {
  children: React.ReactNode
}

interface State {
  hasError: boolean
  error: Error | null
}

export class WidgetErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props)
    this.state = { hasError: false, error: null }
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error }
  }

  componentDidCatch(error: Error) {
    console.warn('Widget error:', error.message)
  }

  handleRetry = () => {
    this.setState({ hasError: false, error: null })
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="flex flex-col items-center gap-2 py-6 text-center">
          <AlertTriangle className="h-5 w-5 text-muted-foreground" />
          <p className="text-xs text-muted-foreground">Something went wrong</p>
          <Button variant="ghost" size="sm" onClick={this.handleRetry}>
            Retry
          </Button>
        </div>
      )
    }

    return this.props.children
  }
}