"use client";

import React, { createContext, useContext, useState, useEffect, useCallback } from "react";
import { 
  buildAuthorizationUrl, 
  TokenStorage, 
  verifyToken,
  refreshAccessToken 
} from "./eve-sso";

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
  accessToken: string | null;
  login: () => Promise<void>;
  logout: () => void;
  getAuthHeader: () => string | null;
  refreshSession: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

const EVE_CLIENT_ID = process.env.NEXT_PUBLIC_EVE_CLIENT_ID || "0828b4bcd20242aeb9b8be10f5451094";
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
  const [accessToken, setAccessToken] = useState<string | null>(null);

  const logout = useCallback(() => {
    TokenStorage.clear();
    setIsAuthenticated(false);
    setCharacter(null);
    setAccessToken(null);
  }, []);

  const checkSession = useCallback(async () => {
    try {
      const token = TokenStorage.getAccessToken();
      
      console.log("[AuthContext] Checking session, token:", token ? "exists" : "none");
      
      if (!token || TokenStorage.isExpired()) {
        // No valid token
        console.log("[AuthContext] No valid token, clearing session");
        setIsAuthenticated(false);
        setCharacter(null);
        setAccessToken(null);
        TokenStorage.clear();
        setIsLoading(false);
        return;
      }

      console.log("[AuthContext] Verifying token with EVE ESI...");
      
      // Verify token with EVE ESI
      const charInfo = await verifyToken(token);
      
      console.log("[AuthContext] Token verified, character:", charInfo.CharacterName);
      
      // Convert to our format
      const character: CharacterInfo = {
        character_id: charInfo.CharacterID,
        character_name: charInfo.CharacterName,
        scopes: charInfo.Scopes ? charInfo.Scopes.split(" ") : [],
        owner_hash: charInfo.CharacterOwnerHash,
        portrait_url: `https://images.evetech.net/characters/${charInfo.CharacterID}/portrait?size=64`,
      };

      setCharacter(character);
      setAccessToken(token);
      setIsAuthenticated(true);
      
      console.log("[AuthContext] Session set, authenticated:", true);
      
      // Save character info
      TokenStorage.saveCharacterInfo(charInfo);
    } catch (error) {
      console.error("[AuthContext] Session verification failed:", error);
      
      // Don't clear tokens on verification errors - might be temporary network issue
      // Only clear UI state
      setIsAuthenticated(false);
      setCharacter(null);
      setAccessToken(null);
      
      // Only clear tokens if it's an auth error (401/403), not network errors
      if (error instanceof Error && error.message.includes("401")) {
        console.log("[AuthContext] Token invalid (401), clearing storage");
        TokenStorage.clear();
      }
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Check for existing session on mount
  useEffect(() => {
    checkSession();
    
    // Listen for custom event from callback page
    const handleLoginSuccess = () => {
      console.log("[AuthContext] Login success event received, checking session...");
      // Small delay to ensure localStorage is fully written
      setTimeout(() => {
        checkSession();
      }, 100);
    };
    
    window.addEventListener("eve-login-success", handleLoginSuccess);
    
    return () => {
      window.removeEventListener("eve-login-success", handleLoginSuccess);
    };
  }, [checkSession]);

  // Token refresh function (wrapped in useCallback to prevent re-creation)
  const performTokenRefresh = useCallback(async () => {
    try {
      const refreshToken = TokenStorage.getRefreshToken();
      
      if (!refreshToken) {
        console.warn("[AuthContext] No refresh token available");
        logout();
        return;
      }

      console.log("[AuthContext] Refreshing access token...");
      
      const newToken = await refreshAccessToken(refreshToken, EVE_CLIENT_ID);
      
      // Save new token
      TokenStorage.save(newToken);
      setAccessToken(newToken.access_token);
      
      console.log("[AuthContext] Token refreshed successfully");
    } catch (error) {
      console.error("[AuthContext] Token refresh failed:", error);
      // On refresh failure, logout user
      logout();
    }
  }, [logout]);

  // Background token refresh - check every 60 seconds
  useEffect(() => {
    if (!isAuthenticated) return;

    const refreshInterval = setInterval(async () => {
      try {
        // Check if token needs refresh (3 minutes before expiry)
        if (TokenStorage.shouldRefresh()) {
          console.log("[AuthContext] Token expiring soon, refreshing...");
          await performTokenRefresh();
        }
      } catch (error) {
        console.error("[AuthContext] Background refresh check failed:", error);
      }
    }, 60 * 1000); // Check every 60 seconds

    return () => clearInterval(refreshInterval);
  }, [isAuthenticated, performTokenRefresh]);

  const login = async () => {
    try {
      const authUrl = await buildAuthorizationUrl(
        EVE_CLIENT_ID,
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

  const getAuthHeader = useCallback((): string | null => {
    if (!accessToken) return null;
    return `Bearer ${accessToken}`;
  }, [accessToken]);

  return (
    <AuthContext.Provider
      value={{
        isAuthenticated,
        character,
        isLoading,
        accessToken,
        login,
        logout,
        getAuthHeader,
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
