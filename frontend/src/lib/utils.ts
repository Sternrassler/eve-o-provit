import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

/**
 * Formats ISK values with thousand separators using German locale
 * @param value - The ISK amount to format
 * @returns Formatted string with German locale thousand separators (e.g., "1.234.567,89 ISK")
 */
export function formatISKWithSeparators(value: number): string {
  return new Intl.NumberFormat("de-DE", {
    style: "decimal",
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(value) + " ISK";
}

/**
 * Formats ISK values in compact notation (M/k suffixes)
 * @param value - The ISK amount to format
 * @returns Compact string e.g. "1.23M ISK", "123k ISK", "500 ISK"
 */
export function formatISK(value: number): string {
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(2)}M ISK`;
  if (value >= 1_000) return `${(value / 1_000).toFixed(0)}k ISK`;
  return `${value.toFixed(0)} ISK`;
}

export function getSpreadColor(spread: number): string {
  if (spread >= 10) return "text-green-600 dark:text-green-400";
  if (spread >= 5) return "text-yellow-600 dark:text-yellow-400";
  return "text-red-600 dark:text-red-400";
}

export function getProfitColor(netMarginPercent: number): string {
  if (netMarginPercent >= 10) return "text-green-600 dark:text-green-400";
  if (netMarginPercent >= 5) return "text-yellow-600 dark:text-yellow-400";
  return "text-red-600 dark:text-red-400";
}
