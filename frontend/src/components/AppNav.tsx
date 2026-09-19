"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

import { API_BASE_URL } from "@/lib/api";

const links = [
  { href: "/", label: "Pipeline" },
  { href: "/stats", label: "Analytics" },
];

export function AppNav() {
  const pathname = usePathname();

  return (
    <header className="border-b border-[var(--line)] bg-[color-mix(in_srgb,var(--panel)_88%,transparent)] backdrop-blur-sm">
      <div className="mx-auto flex max-w-[1400px] items-center justify-between gap-4 px-4 py-3 md:px-6">
        <div>
          <p className="mono text-[11px] uppercase tracking-[0.14em] text-[var(--muted)]">
            Job Agent
          </p>
          <h1 className="text-lg font-medium tracking-tight text-[var(--ink)]">
            Control Center
          </h1>
        </div>
        <nav className="flex items-center gap-1">
          {links.map((link) => {
            const active =
              link.href === "/"
                ? pathname === "/"
                : pathname.startsWith(link.href);
            return (
              <Link
                key={link.href}
                href={link.href}
                className={`rounded-sm px-3 py-2 text-sm transition-colors ${
                  active
                    ? "bg-[var(--accent-soft)] text-[var(--accent)]"
                    : "text-[var(--muted)] hover:text-[var(--ink)]"
                }`}
              >
                {link.label}
              </Link>
            );
          })}
          <a
            href={`${API_BASE_URL}/docs`}
            target="_blank"
            rel="noreferrer"
            className="mono ml-2 hidden rounded-sm border border-[var(--line)] px-3 py-2 text-[11px] uppercase tracking-wide text-[var(--muted)] hover:border-[var(--accent)] hover:text-[var(--accent)] sm:inline-block"
          >
            API Docs
          </a>
        </nav>
      </div>
    </header>
  );
}
