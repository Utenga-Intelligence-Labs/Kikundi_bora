import { Link, useRouterState, useNavigate } from "@tanstack/react-router";
import { Settings, User as UserIcon, LogIn, LogOut } from "lucide-react";
import type { ReactNode } from "react";
import { useAuth, initials } from "@/lib/auth-provider";
import { roleMap, type Jukumu } from "@/api/types";
import { getSidebarNav, mobileNav } from "@/lib/roles";
import { useIsCommitteeMember } from "@/hooks/use-loan-committee";

const navSecondary = [
  { to: "/wasifu", label: "Wasifu wangu", icon: UserIcon },
  { to: "/mipangilio", label: "Mipangilio", icon: Settings },
] as const;

function isActive(pathname: string, to: string) {
  return to === "/" ? pathname === "/" : pathname.startsWith(to);
}


export function AppShell({
  children,
  title,
  subtitle,
  action,
}: {
  children: ReactNode;
  title?: string;
  subtitle?: string;
  action?: ReactNode;
}) {
  const path = useRouterState({ select: (s) => s.location.pathname });
  const navigate = useNavigate();
  const { user, logout } = useAuth();
  const jukumu: Jukumu = user ? roleMap[user.role] ?? "Mwanachama" : "Mwanachama";
  const { data: committeeCheck } = useIsCommitteeMember();
  const isCommitteeMember = committeeCheck?.is_committee_member ?? false;
  const navSidebar = getSidebarNav(jukumu, isCommitteeMember);
  const navMobile = mobileNav(jukumu, isCommitteeMember);


  return (
    <div className="min-h-dvh bg-background text-foreground lg:flex">
      {/* Desktop sidebar */}
      <aside className="sticky top-0 hidden h-dvh w-64 shrink-0 border-r border-border bg-card lg:flex lg:flex-col">
        <Link to="/" className="flex items-center gap-2.5 px-5 py-5">
          <span className="grid h-10 w-10 place-items-center rounded-xl bg-primary text-primary-foreground font-display text-lg font-bold">K</span>
          <div>
            <p className="font-display text-base font-bold leading-tight">Kikundi</p>
            <p className="text-[11px] text-muted-foreground">Kikundi & Mikopo</p>
          </div>
        </Link>
        <nav className="flex-1 overflow-y-auto px-3 pb-4">
          <p className="px-3 pb-1.5 pt-3 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">Menyu</p>
          <ul className="space-y-0.5">
            {navSidebar.map(({ to, label, icon: Icon }) => {
              const active = isActive(path, to);
              return (
                <li key={to}>
                  <Link
                    to={to}
                    className={`flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium transition-colors ${
                      active ? "bg-primary text-primary-foreground" : "text-foreground/80 hover:bg-muted"
                    }`}
                  >
                    <Icon className="h-4 w-4" strokeWidth={active ? 2.5 : 2} />
                    {label}
                  </Link>
                </li>
              );
            })}
          </ul>
          <p className="px-3 pb-1.5 pt-5 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">Akaunti</p>
          <ul className="space-y-0.5">
            {navSecondary.map(({ to, label, icon: Icon }) => {
              const active = isActive(path, to);
              return (
                <li key={to}>
                  <Link
                    to={to}
                    className={`flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium ${
                      active ? "bg-primary text-primary-foreground" : "text-foreground/80 hover:bg-muted"
                    }`}
                  >
                    <Icon className="h-4 w-4" />
                    {label}
                  </Link>
                </li>
              );
            })}
            {!user && (
              <li>
                <Link to="/ingia" className="flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium text-foreground/80 hover:bg-muted">
                  <LogIn className="h-4 w-4" /> Ingia
                </Link>
              </li>
            )}
          </ul>
        </nav>
        {user && (
          <button
            onClick={async () => { await logout(); navigate({ to: "/" }); }}
            className="mx-3 mb-2 flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium text-foreground/80 hover:bg-muted transition-colors"
          >
            <LogOut className="h-4 w-4" />
            Toka kwenye akaunti
          </button>
        )}
        {user && (
          <Link to="/wasifu" className="flex items-center gap-3 border-t border-border px-5 py-4 hover:bg-muted">
            <span className="grid h-9 w-9 place-items-center rounded-full text-xs font-bold text-white bg-primary">
              {initials(user.name)}
            </span>
            <div className="min-w-0">
              <p className="truncate text-sm font-semibold">{user.name}</p>
              <p className="truncate text-[11px] text-muted-foreground">{roleMap[user.role] ?? user.role}</p>
            </div>
          </Link>

        )}
      </aside>

      {/* Main column */}
      <div className="flex min-w-0 flex-1 flex-col">
        {/* Mobile top bar */}
        <header className="sticky top-0 z-30 border-b border-border bg-card lg:hidden">
          <div className="mx-auto flex max-w-3xl items-center justify-between px-4 pt-[max(0.75rem,env(safe-area-inset-top))] pb-3">
            <Link to="/" className="flex items-center gap-2">
              <span className="grid h-9 w-9 place-items-center rounded-xl bg-primary text-primary-foreground font-display font-bold">K</span>
              <span className="font-display text-lg font-semibold tracking-tight">Kikundi</span>
            </Link>
            {user ? (
              <div className="flex items-center gap-2">
                <button
                  onClick={async () => { await logout(); navigate({ to: "/" }); }}
                  className="grid h-9 w-9 place-items-center rounded-full text-foreground/80 hover:bg-muted transition-colors"
                  aria-label="Toka kwenye akaunti"
                >
                  <LogOut className="h-4 w-4" />
                </button>
                <Link to="/wasifu" className="flex items-center gap-2">
                  <span className="grid h-9 w-9 place-items-center rounded-full text-xs font-bold text-white bg-primary">
                    {initials(user.name)}
                  </span>
                </Link>
              </div>

            ) : (
              <Link to="/ingia" className="rounded-full bg-primary px-4 py-2 text-xs font-semibold text-primary-foreground">
                Ingia
              </Link>
            )}
          </div>
          {(title || action) && (
            <div className="mx-auto flex max-w-3xl items-end justify-between gap-3 px-4 pb-4">
              <div className="min-w-0">
                {title && <h1 className="truncate font-display text-2xl font-bold">{title}</h1>}
                {subtitle && <p className="mt-0.5 text-sm text-muted-foreground">{subtitle}</p>}
              </div>
              {action && <div className="shrink-0">{action}</div>}
            </div>
          )}
        </header>

        {/* Desktop page header */}
        <header className="hidden border-b border-border bg-card lg:block">
          <div className="mx-auto flex max-w-6xl items-end justify-between gap-4 px-8 py-6">
            <div className="min-w-0">
              {title && <h1 className="font-display text-3xl font-bold tracking-tight">{title}</h1>}
              {subtitle && <p className="mt-1 text-sm text-muted-foreground">{subtitle}</p>}
            </div>
            {action && <div className="shrink-0">{action}</div>}
          </div>
        </header>

        <main className="mx-auto w-full max-w-3xl flex-1 px-4 pt-5 pb-32 lg:max-w-6xl lg:px-8 lg:pt-8 lg:pb-12">
          {children}
        </main>
      </div>

      {/* Mobile bottom nav — per user */}
      <nav className="fixed inset-x-0 bottom-0 z-40 border-t border-border bg-card/95 backdrop-blur lg:hidden">
        <ul className="mx-auto grid max-w-3xl grid-cols-5 px-1 pt-1.5 pb-[max(0.375rem,env(safe-area-inset-bottom))]">
          {navMobile.map(({ to, label, icon: Icon }) => {
            const active = isActive(path, to);
            return (
              <li key={to}>
                <Link
                  to={to}
                  className={`flex flex-col items-center gap-0.5 rounded-lg px-1 py-1.5 text-[10px] font-medium transition-colors ${
                    active ? "text-primary" : "text-muted-foreground hover:text-foreground"
                  }`}
                >
                  <Icon className="h-5 w-5" strokeWidth={active ? 2.5 : 1.75} />
                  <span className="truncate">{label}</span>
                </Link>
              </li>
            );
          })}
        </ul>
      </nav>
    </div>
  );
}

export function TwoColumn({ children }: { children: ReactNode }) {
  return <div className="grid gap-5 lg:grid-cols-2">{children}</div>;
}
