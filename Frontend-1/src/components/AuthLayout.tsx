import { Link } from "@tanstack/react-router";

export function AuthLayout({
  children,
  title,
  subtitle,
  footer,
}: {
  children: React.ReactNode;
  title: string;
  subtitle?: string;
  footer?: React.ReactNode;
}) {
  return (
    <div className="min-h-dvh bg-muted/30">
      <header className="border-b border-border bg-background">
        <div className="mx-auto flex max-w-6xl items-center justify-between px-4 py-3 lg:px-8">
          <Link to="/" className="flex items-center gap-2">
            <span className="grid h-9 w-9 place-items-center rounded-xl bg-primary text-primary-foreground font-display font-bold">
              K
            </span>
            <span className="font-display text-lg font-bold">Kikundi</span>
          </Link>
          <Link
            to="/"
            className="text-sm font-medium text-muted-foreground hover:text-foreground"
          >
            Mwanzo
          </Link>
        </div>
      </header>
      <main className="mx-auto flex max-w-md flex-col px-4 py-8 lg:py-16">
        <div className="card-surface p-6 lg:p-8">
          <h1 className="font-display text-2xl font-bold">{title}</h1>
          {subtitle && (
            <p className="mt-1 text-sm text-muted-foreground">{subtitle}</p>
          )}
          <div className="mt-6">{children}</div>
        </div>
        {footer && <div className="mt-5 text-center">{footer}</div>}
      </main>
    </div>
  );
}
