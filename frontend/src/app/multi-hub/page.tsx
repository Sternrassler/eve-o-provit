"use client";

import { useState } from "react";
import { useQuery, useMutation } from "@tanstack/react-query";
import { useAuth } from "@/lib/auth-context";
import { ErrorBoundary } from "@/components/ErrorBoundary";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { HubComparisonTable } from "@/components/trading/HubComparisonTable";
import { SkillsAppliedPanel } from "@/components/trading/SkillsAppliedPanel";
import { searchItems, compareHubs } from "@/lib/api-client";
import { ItemSearchResult } from "@/types/trading";
import { Loader2, Search } from "lucide-react";

const MIN_QUERY_LENGTH = 3;

function MultiHubPageContent() {
  const { isAuthenticated } = useAuth();
  const [query, setQuery] = useState("");
  const [selectedItem, setSelectedItem] = useState<ItemSearchResult | null>(
    null
  );

  // Item search — runs once the query reaches the backend's 3-char minimum.
  const { data: searchResults = [], isFetching: isSearching } = useQuery({
    queryKey: ["itemSearch", query],
    queryFn: () => searchItems(query),
    enabled: query.trim().length >= MIN_QUERY_LENGTH,
    staleTime: 60 * 1000,
  });

  // Multi-hub compare (authed POST), triggered when an item is picked.
  const compareMutation = useMutation({
    mutationFn: (typeId: number) => compareHubs(typeId),
  });

  const handleSelectItem = (item: ItemSearchResult) => {
    setSelectedItem(item);
    setQuery(item.name);
    compareMutation.mutate(item.type_id);
  };

  const showSuggestions =
    query.trim().length >= MIN_QUERY_LENGTH &&
    selectedItem?.name !== query &&
    searchResults.length > 0;

  const apiError = compareMutation.isError
    ? compareMutation.error instanceof Error
      ? compareMutation.error.message
      : "Unbekannter Fehler"
    : undefined;

  return (
    <div className="container mx-auto px-4 py-8">
      <div className="mb-8">
        <h1 className="mb-2 text-3xl font-bold">Multi-Hub Comparison</h1>
        <p className="text-muted-foreground">
          Vergleiche die Station-Trading-Margen eines Items über die großen
          Handels-Hubs hinweg
        </p>
      </div>

      {!isAuthenticated && (
        <Alert className="mb-6">
          <AlertTitle>Login erforderlich</AlertTitle>
          <AlertDescription>
            Melde dich per EVE SSO an, um skill-bereinigte Hub-Vergleiche zu
            laden.
          </AlertDescription>
        </Alert>
      )}

      {/* Item search */}
      <div className="mb-8 max-w-xl">
        <label
          htmlFor="item-search"
          className="mb-2 block text-sm font-medium"
        >
          Item suchen
        </label>
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            id="item-search"
            type="text"
            placeholder="z.B. Tritanium (min. 3 Zeichen)"
            className="pl-9"
            value={query}
            disabled={!isAuthenticated}
            onChange={(e) => {
              setQuery(e.target.value);
              setSelectedItem(null);
            }}
            autoComplete="off"
          />
          {isSearching && (
            <Loader2 className="absolute right-3 top-1/2 size-4 -translate-y-1/2 animate-spin text-muted-foreground" />
          )}

          {showSuggestions && (
            <ul
              className="absolute z-10 mt-1 max-h-72 w-full overflow-auto rounded-md border bg-popover shadow-md"
              role="listbox"
              aria-label="Suchergebnisse"
            >
              {searchResults.map((item) => (
                <li key={item.type_id} role="option" aria-selected="false">
                  <button
                    type="button"
                    className="flex w-full items-center justify-between px-3 py-2 text-left text-sm hover:bg-muted"
                    onClick={() => handleSelectItem(item)}
                  >
                    <span className="font-medium">{item.name}</span>
                    <span className="text-xs text-muted-foreground">
                      {item.group_name}
                    </span>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>

      {/* Compare state */}
      {compareMutation.isPending && (
        <div className="flex items-center gap-2 text-muted-foreground">
          <Loader2 className="size-4 animate-spin" />
          Lade Hub-Vergleich für {selectedItem?.name}...
        </div>
      )}

      {apiError && (
        <Alert variant="destructive" className="mb-6">
          <AlertTitle>Fehler</AlertTitle>
          <AlertDescription className="flex items-center justify-between gap-4">
            <span>{apiError}</span>
            {selectedItem && (
              <Button
                variant="outline"
                size="sm"
                onClick={() => compareMutation.mutate(selectedItem.type_id)}
              >
                Erneut versuchen
              </Button>
            )}
          </AlertDescription>
        </Alert>
      )}

      {compareMutation.isSuccess && compareMutation.data && (
        <div className="grid gap-6 lg:grid-cols-[1fr_320px]">
          <HubComparisonTable result={compareMutation.data} />
          <SkillsAppliedPanel skills={compareMutation.data.skills_applied} />
        </div>
      )}
    </div>
  );
}

export default function MultiHubPage() {
  return (
    <ErrorBoundary>
      <MultiHubPageContent />
    </ErrorBoundary>
  );
}
