"use client";

import * as React from "react";
import * as DialogPrimitive from "@radix-ui/react-dialog";
import { X, CheckCircle2, AlertCircle, AlertTriangle, Info } from "lucide-react";
import { cn } from "@/lib/utils";

type Variant = "success" | "error" | "warning" | "info";

interface ShowOpts {
  title: string;
  message?: string;
  variant?: Variant;
  primaryLabel?: string;
  onPrimary?: () => void;
  secondaryLabel?: string;
  onSecondary?: () => void;
  children?: React.ReactNode;
  closeOnBackdrop?: boolean;
}

interface ModalState extends ShowOpts {
  open: boolean;
}

const AppModalContext = React.createContext<{
  showModal: (opts: ShowOpts) => void;
  hideModal: () => void;
} | null>(null);

export function useAppModal() {
  const ctx = React.useContext(AppModalContext);
  if (!ctx) throw new Error("useAppModal must be used inside AppModalProvider");
  return ctx;
}

const variantConfig: Record<Variant, { icon: React.ElementType; iconWrap: string; iconColor: string }> = {
  success: { icon: CheckCircle2, iconWrap: "bg-success/15 text-success", iconColor: "text-success" },
  error: { icon: AlertCircle, iconWrap: "bg-destructive/15 text-destructive", iconColor: "text-destructive" },
  warning: { icon: AlertTriangle, iconWrap: "bg-warning/25 text-warning-foreground", iconColor: "text-amber-600" },
  info: { icon: Info, iconWrap: "bg-primary/10 text-primary", iconColor: "text-primary" },
};

export function AppModalProvider({ children }: { children: React.ReactNode }) {
  const [state, setState] = React.useState<ModalState>({
    open: false,
    title: "",
    variant: "info",
  });

  const showModal = React.useCallback((opts: ShowOpts) => {
    setState({ open: true, variant: "info", closeOnBackdrop: true, ...opts });
  }, []);

  const hideModal = React.useCallback(() => {
    setState((s) => ({ ...s, open: false }));
  }, []);

  const handleOpenChange = (open: boolean) => {
    if (!open) hideModal();
  };

  const variant = state.variant ?? "info";
  const cfg = variantConfig[variant];
  const Icon = cfg.icon;

  const hasSecondary = !!state.secondaryLabel;
  // For confirm dialogs, backdrop click should act as cancel (close), not confirm.
  const onInteractOutside = (e: Event) => {
    if (state.closeOnBackdrop === false) {
      e.preventDefault();
    }
  };

  return (
    <AppModalContext.Provider value={{ showModal, hideModal }}>
      {children}
      <DialogPrimitive.Root open={state.open} onOpenChange={handleOpenChange}>
        <DialogPrimitive.Portal>
          <DialogPrimitive.Overlay
            className={cn(
              "fixed inset-0 z-50 bg-black/60 backdrop-blur-sm data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0"
            )}
          />
          <DialogPrimitive.Content
            onInteractOutside={onInteractOutside}
            onEscapeKeyDown={() => hideModal()}
            className={cn(
              "fixed left-[50%] top-[50%] z-50 w-full max-w-md -translate-x-1/2 -translate-y-1/2",
              "card-surface p-0 overflow-hidden",
              "data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 data-[state=closed]:slide-out-to-top-[2%] data-[state=open]:slide-in-from-top-[2%]",
              "duration-200"
            )}
          >
            <div className="p-6">
              <div className="flex items-start gap-4">
                <span className={cn("grid h-10 w-10 shrink-0 place-items-center rounded-xl", cfg.iconWrap)}>
                  <Icon className="h-5 w-5" />
                </span>
                <div className="flex-1 min-w-0">
                  <DialogPrimitive.Title className="font-display text-lg font-bold leading-tight">
                    {state.title}
                  </DialogPrimitive.Title>
                  {state.message && (
                    <DialogPrimitive.Description className="mt-1.5 text-sm leading-relaxed text-muted-foreground">
                      {state.message}
                    </DialogPrimitive.Description>
                  )}
                  {state.children && <div className="mt-3 text-sm">{state.children}</div>}
                </div>
                <DialogPrimitive.Close
                  onClick={hideModal}
                  className="grid h-8 w-8 place-items-center rounded-lg text-muted-foreground hover:bg-muted transition-colors"
                  aria-label="Funga"
                >
                  <X className="h-4 w-4" />
                </DialogPrimitive.Close>
              </div>
            </div>
            <div className="flex gap-2 justify-end bg-muted/30 px-6 py-4 border-t">
              {hasSecondary && (
                <button
                  onClick={() => {
                    state.onSecondary?.();
                    hideModal();
                  }}
                  className="rounded-xl border bg-background px-4 py-2 text-sm font-semibold hover:bg-muted transition-colors"
                >
                  {state.secondaryLabel}
                </button>
              )}
              <button
                autoFocus
                onClick={() => {
                  state.onPrimary?.();
                  hideModal();
                }}
                className={cn(
                  "rounded-xl px-5 py-2 text-sm font-semibold text-white transition-colors",
                  variant === "error" || variant === "warning" ? "bg-destructive hover:bg-destructive/90" : variant === "success" ? "bg-success hover:bg-success/90" : "bg-primary hover:bg-primary/90"
                )}
              >
                {state.primaryLabel ?? "Sawa"}
              </button>
            </div>
          </DialogPrimitive.Content>
        </DialogPrimitive.Portal>
      </DialogPrimitive.Root>
    </AppModalContext.Provider>
  );
}

// Declarative component for local use (same design, without context)
interface AppModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  message?: string;
  variant?: Variant;
  primaryLabel?: React.ReactNode;
  onPrimary?: () => void;
  secondaryLabel?: string;
  onSecondary?: () => void;
  children?: React.ReactNode;
  closeOnBackdrop?: boolean;
}

export function AppModal({
  open,
  onOpenChange,
  title,
  message,
  variant = "info",
  primaryLabel = "Sawa",
  onPrimary,
  secondaryLabel,
  onSecondary,
  children,
  closeOnBackdrop = true,
}: AppModalProps) {
  const cfg = variantConfig[variant];
  const Icon = cfg.icon;
  const hasSecondary = !!secondaryLabel;

  return (
    <DialogPrimitive.Root open={open} onOpenChange={onOpenChange}>
      <DialogPrimitive.Portal>
        <DialogPrimitive.Overlay className="fixed inset-0 z-50 bg-black/60 backdrop-blur-sm data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0" />
        <DialogPrimitive.Content
          onInteractOutside={(e) => {
            if (closeOnBackdrop === false) e.preventDefault();
          }}
          className="fixed left-[50%] top-[50%] z-50 w-full max-w-md -translate-x-1/2 -translate-y-1/2 card-surface p-0 overflow-hidden data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0 data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95 duration-200"
        >
          <div className="p-6">
            <div className="flex items-start gap-4">
              <span className={cn("grid h-10 w-10 shrink-0 place-items-center rounded-xl", cfg.iconWrap)}>
                <Icon className="h-5 w-5" />
              </span>
              <div className="flex-1 min-w-0">
                <DialogPrimitive.Title className="font-display text-lg font-bold leading-tight">{title}</DialogPrimitive.Title>
                {message && <DialogPrimitive.Description className="mt-1.5 text-sm leading-relaxed text-muted-foreground">{message}</DialogPrimitive.Description>}
                {children && <div className="mt-3 text-sm">{children}</div>}
              </div>
              <DialogPrimitive.Close className="grid h-8 w-8 place-items-center rounded-lg text-muted-foreground hover:bg-muted transition-colors">
                <X className="h-4 w-4" />
              </DialogPrimitive.Close>
            </div>
          </div>
          <div className="flex gap-2 justify-end bg-muted/30 px-6 py-4 border-t">
            {hasSecondary && (
              <button
                onClick={() => {
                  onSecondary?.();
                  onOpenChange(false);
                }}
                className="rounded-xl border bg-background px-4 py-2 text-sm font-semibold hover:bg-muted transition-colors"
              >
                {secondaryLabel}
              </button>
            )}
            <button
              autoFocus
              onClick={() => {
                onPrimary?.();
                onOpenChange(false);
              }}
              className={cn(
                "rounded-xl px-5 py-2 text-sm font-semibold text-white transition-colors",
                variant === "error" || variant === "warning" ? "bg-destructive hover:bg-destructive/90" : variant === "success" ? "bg-success hover:bg-success/90" : "bg-primary hover:bg-primary/90"
              )}
            >
              {primaryLabel}
            </button>
          </div>
        </DialogPrimitive.Content>
      </DialogPrimitive.Portal>
    </DialogPrimitive.Root>
  );
}
