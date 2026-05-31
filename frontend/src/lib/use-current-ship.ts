"use client";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "@/lib/auth-context";
import { fetchCharacterShip } from "@/lib/api-client";

export function useCurrentShip() {
  const { isAuthenticated } = useAuth();
  const q = useQuery({
    queryKey: ["currentShip"],
    queryFn: fetchCharacterShip,
    enabled: isAuthenticated,
    staleTime: 5 * 1000, // ESI caches the current ship for 5s
  });
  return {
    ship: q.data ?? null,
    isLoading: q.isLoading,
    error: q.isError,
    refresh: () => q.refetch(),
  };
}
