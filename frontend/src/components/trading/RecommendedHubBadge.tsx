"use client";

import { Badge } from "@/components/ui/badge";
import { Star } from "lucide-react";
import { cn } from "@/lib/utils";

interface RecommendedHubBadgeProps {
  className?: string;
}

/**
 * Small badge marking the recommended (best-margin) hub in the comparison.
 */
export function RecommendedHubBadge({ className }: RecommendedHubBadgeProps) {
  return (
    <Badge
      className={cn(
        "gap-1 border-transparent bg-green-600 text-white hover:bg-green-600/90",
        className
      )}
    >
      <Star className="size-3 fill-current" aria-hidden="true" />
      Empfohlen
    </Badge>
  );
}
