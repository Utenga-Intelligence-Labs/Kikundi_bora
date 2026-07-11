interface FieldProps {
  icon?: React.ComponentType<{ className?: string }>;
  label: string;
  type?: string;
  value: string;
  onChange: (v: string) => void;
  placeholder?: string;
  autoComplete?: string;
}

export function Field({
  icon: Icon,
  label,
  type = "text",
  value,
  onChange,
  placeholder,
  autoComplete,
}: FieldProps) {
  return (
    <label className="block">
      <span className="mb-1 block text-xs font-medium text-muted-foreground">
        {label}
      </span>
      <div className="flex items-center gap-2 rounded-xl border border-input bg-background px-3 py-2.5 focus-within:border-primary focus-within:ring-2 focus-within:ring-ring/20">
        {Icon && <Icon className="h-4 w-4 text-muted-foreground" />}
        <input
          type={type}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder={placeholder}
          autoComplete={autoComplete}
          className="w-full bg-transparent text-sm outline-none"
        />
      </div>
    </label>
  );
}
