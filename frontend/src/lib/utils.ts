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
