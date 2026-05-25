"use client";

import { useState, useEffect, useRef } from "react";
import { Input } from "@/components/ui/input";
import { ItemSearchResult } from "@/types/trading";

interface ItemAutocompleteProps {
  value?: ItemSearchResult | null;
  onChange: (item: ItemSearchResult | null) => void;
  apiUrl: string;
}

export function ItemAutocomplete({ value, onChange, apiUrl }: ItemAutocompleteProps) {
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<ItemSearchResult[]>([]);
  const [isOpen, setIsOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const wrapperRef = useRef<HTMLDivElement>(null);

  // Debounce search
  useEffect(() => {
    if (query.length < 3) {
      setResults([]);
      setIsOpen(false);
      return;
    }

    const timer = setTimeout(async () => {
      setIsLoading(true);
      try {
        const response = await fetch(`${apiUrl}/api/v1/items/search?q=${encodeURIComponent(query)}&limit=20`);
        if (response.ok) {
          const data = await response.json();
          setResults(data.items || []);
          setIsOpen(true);
        }
      } catch (error) {
        console.error("Failed to search items:", error);
      } finally {
        setIsLoading(false);
      }
    }, 300);

    return () => clearTimeout(timer);
  }, [query, apiUrl]);

  // Close dropdown when clicking outside
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (wrapperRef.current && !wrapperRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    }

    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  const handleSelect = (item: ItemSearchResult) => {
    onChange(item);
    setQuery(item.name);
    setIsOpen(false);
  };

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newQuery = e.target.value;
    setQuery(newQuery);
    if (newQuery !== value?.name) {
      onChange(null); // Clear selection if user types
    }
  };

  return (
    <div ref={wrapperRef} className="relative">
      <Input
        type="text"
        value={query}
        onChange={handleInputChange}
        placeholder="Suche Item (min. 3 Zeichen)..."
        className="w-full"
      />
      {isLoading && (
        <div className="absolute right-3 top-1/2 -translate-y-1/2">
          <div className="h-4 w-4 animate-spin rounded-full border-2 border-primary border-t-transparent"></div>
        </div>
      )}
      {isOpen && results.length > 0 && (
        <div className="absolute z-50 mt-1 max-h-60 w-full overflow-auto rounded-md border bg-popover text-popover-foreground shadow-md">
          {results.map((item) => (
            <button
              key={item.type_id}
              onClick={() => handleSelect(item)}
              className="w-full px-3 py-2 text-left hover:bg-accent hover:text-accent-foreground focus:bg-accent focus:text-accent-foreground focus:outline-none"
            >
              <div className="font-medium">{item.name}</div>
              <div className="text-xs text-muted-foreground">{item.group_name}</div>
            </button>
          ))}
        </div>
      )}
      {isOpen && results.length === 0 && query.length >= 3 && !isLoading && (
        <div className="absolute z-50 mt-1 w-full rounded-md border bg-popover p-3 text-sm text-muted-foreground shadow-md">
          Keine Items gefunden
        </div>
      )}
    </div>
  );
}
