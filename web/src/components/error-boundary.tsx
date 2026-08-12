import { Component, type ErrorInfo, type ReactNode } from "react";

interface Props {
  children: ReactNode;
}

interface State {
  error: Error | null;
}

export class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null };

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error("Uncaught render error:", error, info.componentStack);
  }

  render() {
    if (this.state.error) {
      return (
        <div className="flex min-h-screen flex-col items-center justify-center gap-4 bg-background p-8 text-center">
          <p className="text-lg font-semibold text-foreground">Something went wrong</p>
          <p className="max-w-md text-sm text-muted-foreground">
            {this.state.error.message || "An unexpected error occurred while rendering the page."}
          </p>
          <button
            onClick={() => this.setState({ error: null })}
            className="rounded-lg border border-border bg-popover px-4 py-2 text-sm font-medium transition-colors hover:bg-accent"
          >
            Try again
          </button>
        </div>
      );
    }
    return this.props.children;
  }
}
