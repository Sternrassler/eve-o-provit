"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";

// Sub-tabs shared across the five trading tools, so the Trading section reads as
// one hub (switch tools without going back to the top-nav). Mirrors the Flutter
// client's Trading section. Each tab is a real route — deep links keep working.
const tabs = [
  { href: "/trading", label: "Routes" },
  { href: "/hauling", label: "Hauling" },
  { href: "/roi-calculator", label: "ROI" },
  { href: "/multi-hub", label: "Multi-Hub" },
  { href: "/sell-assets", label: "Sell Assets" },
] as const;

export function TradingTabs() {
  const pathname = usePathname();

  return (
    <nav
      aria-label="Trading-Werkzeuge"
      className="mb-6 flex gap-1 overflow-x-auto border-b"
    >
      {tabs.map((tab) => {
        const active = pathname === tab.href;
        return (
          <Link
            key={tab.href}
            href={tab.href}
            aria-current={active ? "page" : undefined}
            className={cn(
              "-mb-px whitespace-nowrap border-b-2 px-3 py-2 text-sm font-medium transition-colors",
              active
                ? "border-primary text-primary"
                : "border-transparent text-muted-foreground hover:border-muted-foreground/40 hover:text-foreground",
            )}
          >
            {tab.label}
          </Link>
        );
      })}
    </nav>
  );
}
