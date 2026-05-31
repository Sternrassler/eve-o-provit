"use client";

import React, { createContext, useContext, useState, useEffect, useCallback } from "react";
import { useAuth } from "./auth-context";
import type { TradingSkills } from "@/types/character";

interface TradingSkillsContextType {
  skills: TradingSkills | null;
  loading: boolean;
  error: string | null;
  refreshSkills: () => Promise<void>;
}

const TradingSkillsContext = createContext<TradingSkillsContextType | undefined>(undefined);

/**
 * Default skills (all = 0) used as fallback when skills cannot be fetched
 * Ensures worst-case calculations (highest fees, lowest cargo)
 */
const getDefaultSkills = (): TradingSkills => ({
  Accounting: 0,
  BrokerRelations: 0,
  AdvancedBrokerRelations: 0,
  FactionStanding: 0.0,
  CorpStanding: 0.0,
  SpaceshipCommand: 0,
  CargoOptimization: 0,
  Navigation: 0,
  EvasiveManeuvering: 0,
  GallenteIndustrial: 0,
  CaldariIndustrial: 0,
  AmarrIndustrial: 0,
  MinmatarIndustrial: 0,
});

export function TradingSkillsProvider({ children }: { children: React.ReactNode }) {
  const { character, isAuthenticated } = useAuth();
  const [skills, setSkills] = useState<TradingSkills | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchSkills = useCallback(async () => {
    if (!character?.character_id) {
      setSkills(getDefaultSkills());
      setLoading(false);
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const response = await fetch(
        `${process.env.NEXT_PUBLIC_API_URL || "http://localhost:9001"}/api/v1/characters/${character.character_id}/skills`,
        {
          credentials: "include",
        }
      );

      if (!response.ok) {
        throw new Error(`Failed to fetch skills: ${response.statusText}`);
      }

      const data = await response.json();
      setSkills(data.skills);
      setError(null);
    } catch (err) {
      console.error("[TradingSkillsContext] Failed to fetch skills:", err);
      setError(err instanceof Error ? err.message : "Failed to fetch skills");

      // Do NOT substitute zeroed default skills here — that would silently feed
      // wrong (worst-case) fee/cargo numbers into the UI as if they were real.
      // Leave skills null so consumers surface the error instead.
      setSkills(null);
    } finally {
      setLoading(false);
    }
  }, [character?.character_id]);

  // Auto-fetch on login
  useEffect(() => {
    if (isAuthenticated && character) {
      fetchSkills();
    } else {
      setSkills(getDefaultSkills());
      setLoading(false);
    }
  }, [isAuthenticated, character, fetchSkills]);

  const refreshSkills = useCallback(async () => {
    await fetchSkills();
  }, [fetchSkills]);

  return (
    <TradingSkillsContext.Provider
      value={{
        skills,
        loading,
        error,
        refreshSkills,
      }}
    >
      {children}
    </TradingSkillsContext.Provider>
  );
}

export function useTradingSkills() {
  const context = useContext(TradingSkillsContext);
  if (context === undefined) {
    throw new Error("useTradingSkills must be used within a TradingSkillsProvider");
  }
  return context;
}
