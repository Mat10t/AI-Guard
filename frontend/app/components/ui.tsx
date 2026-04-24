import React from "react";

type Tone = "info" | "success" | "warning" | "error";

function cn(...values: Array<string | undefined | false>): string {
  return values.filter(Boolean).join(" ");
}

export function Card({ className, children }: { className?: string; children: React.ReactNode }) {
  return <section className={cn("card", className)}>{children}</section>;
}

export function SectionHeader({
  title,
  subtitle,
  badge,
  actions
}: {
  title: string;
  subtitle?: string;
  badge?: string;
  actions?: React.ReactNode;
}) {
  return (
    <div className="section-header">
      <div>
        {badge ? <span className="badge">{badge}</span> : null}
        <h1>{title}</h1>
        {subtitle ? <p className="muted">{subtitle}</p> : null}
      </div>
      {actions ? <div className="section-actions">{actions}</div> : null}
    </div>
  );
}

export function Notice({ tone = "info", children }: { tone?: Tone; children: React.ReactNode }) {
  return <p className={cn("notice", `notice-${tone}`)}>{children}</p>;
}

export function EmptyState({ title, description }: { title: string; description: string }) {
  return (
    <div className="empty-state">
      <strong>{title}</strong>
      <p className="muted">{description}</p>
    </div>
  );
}

export function Button(
  props: React.ButtonHTMLAttributes<HTMLButtonElement> & { variant?: "primary" | "secondary" | "ghost" }
) {
  const { variant = "primary", className, ...rest } = props;
  return <button className={cn("btn", `btn-${variant}`, className)} {...rest} />;
}

export function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="field">
      <span>{label}</span>
      {children}
    </label>
  );
}
