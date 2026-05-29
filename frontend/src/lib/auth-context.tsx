"use client";

import React, { createContext, useContext, useState, useEffect, useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { buildAuthorizationUrl } from "./eve-sso";

interface CharacterInfo {
  character_id: number;
  character_name: string;
  scopes: string[];
  owner_hash: string;
  portrait_url: string;
}

interface AuthContextType {
  isAuthenticated: boolean;
  character: CharacterInfo | null;
  isLoading: boolean;
  login: () => Promise<void>;
  logout: () => void;
  refreshSession: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:9001";
const EVE_CALLBACK_URL = process.env.NEXT_PUBLIC_EVE_CALLBACK_URL || "http://localhost:9000/callback";
const EVE_SCOPES: string[] = [
  "esi-location.read_location.v1",
  "esi-location.read_ship_type.v1",
  "esi-clones.read_clones.v1",
  "esi-assets.read_assets.v1",
  "esi-ui.write_waypoint.v1",
  "esi-skills.read_skills.v1",
];

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [character, setCharacter] = useState<CharacterInfo | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const queryClient = useQueryClient();

  const logout = useCallback(async () => {
    await fetch(`${API_BASE_URL}/auth/logout`, {
      method: "POST",
      credentials: "include",
    });
    setIsAuthenticated(false);
    setCharacter(null);
    // Drop all cached per-character data (location, ship, …) so the next login
    // in the same browser can't be served the previous character's data.
    queryClient.clear();
  }, [queryClient]);

  const checkSession = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/auth/session`, {
        credentials: "include",
      });
      const data = await response.json();

      if (!data.authenticated || !data.character) {
        setIsAuthenticated(false);
        setCharacter(null);
        setIsLoading(false);
        return;
      }

      setCharacter({
        character_id: data.character.character_id,
        character_name: data.character.character_name,
        scopes: data.character.scopes,
        owner_hash: data.character.owner_hash ?? "",
        portrait_url: data.character.portrait_url,
      });
      setIsAuthenticated(true);
      setIsLoading(false);
    } catch (error) {
      console.error("[AuthContext] Session check failed:", error);
      setIsAuthenticated(false);
      setCharacter(null);
      setIsLoading(false);
    }
  }, []);

  // Check for existing session on mount
  useEffect(() => {
    const init = async () => {
      await checkSession();
    };
    void init();

    // Listen for custom event from callback page
    const handleLoginSuccess = () => {
      void checkSession();
    };

    window.addEventListener("eve-login-success", handleLoginSuccess);

    return () => {
      window.removeEventListener("eve-login-success", handleLoginSuccess);
    };
  }, [checkSession]);

  // Background token refresh — check every 60 seconds
  const performTokenRefresh = useCallback(async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/auth/refresh`, {
        method: "POST",
        credentials: "include",
      });
      if (!response.ok) {
        logout();
      }
    } catch (error) {
      console.error("[AuthContext] Token refresh failed:", error);
      logout();
    }
  }, [logout]);

  useEffect(() => {
    if (!isAuthenticated) return;

    const refreshInterval = setInterval(async () => {
      await performTokenRefresh();
    }, 60 * 1000);

    return () => clearInterval(refreshInterval);
  }, [isAuthenticated, performTokenRefresh]);

  const login = async () => {
    const clientId = process.env.NEXT_PUBLIC_EVE_CLIENT_ID;
    if (!clientId) {
      throw new Error("NEXT_PUBLIC_EVE_CLIENT_ID is not set");
    }
    try {
      const authUrl = await buildAuthorizationUrl(
        clientId,
        EVE_CALLBACK_URL,
        EVE_SCOPES
      );
      window.location.href = authUrl;
    } catch (error) {
      console.error("Failed to initiate login:", error);
    }
  };

  const refreshSession = useCallback(async () => {
    await checkSession();
  }, [checkSession]);

  return (
    <AuthContext.Provider
      value={{
        isAuthenticated,
        character,
        isLoading,
        login,
        logout,
        refreshSession,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
}
