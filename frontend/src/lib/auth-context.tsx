"use client";

import React, { createContext, useContext, useState, useEffect } from "react";
import { 
  buildAuthorizationUrl, 
  TokenStorage,
  MultiCharacterTokenStorage,
  verifyToken,
  refreshAccessToken,
  revokeToken
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
  allCharacters: CharacterInfo[];
  isLoading: boolean;
  accessToken: string | null;
  login: () => Promise<void>;
  logout: () => Promise<void>;
  logoutCharacter: (characterID: number) => Promise<void>;
  switchCharacter: (characterID: number) => Promise<void>;
  getAuthHeader: () => string | null;
  refreshSession: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

const EVE_CLIENT_ID = process.env.NEXT_PUBLIC_EVE_CLIENT_ID || "0828b4bcd20242aeb9b8be10f5451094";
const EVE_CALLBACK_URL = process.env.NEXT_PUBLIC_EVE_CALLBACK_URL || "http://localhost:9000/callback";
const EVE_SCOPES: string[] = []; // Add required scopes here if needed

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [character, setCharacter] = useState<CharacterInfo | null>(null);
  const [allCharacters, setAllCharacters] = useState<CharacterInfo[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [accessToken, setAccessToken] = useState<string | null>(null);

  // Check for existing session on mount
  useEffect(() => {
    checkSession();
    
    // Listen for storage changes (e.g., after login in callback)
    const handleStorageChange = (e: StorageEvent) => {
      if (e.key === "eve_multi_characters" || e.key === "eve_access_token" || e.key === "eve_character_info") {
        checkSession();
      }
    };
    
    // Listen for custom event from callback page
    const handleLoginSuccess = () => {
      checkSession();
    };
    
    window.addEventListener("storage", handleStorageChange);
    window.addEventListener("eve-login-success", handleLoginSuccess);
    
    return () => {
      window.removeEventListener("storage", handleStorageChange);
      window.removeEventListener("eve-login-success", handleLoginSuccess);
    };
  }, []);

  // Background token refresh - check every 60 seconds
  useEffect(() => {
    if (!isAuthenticated) return;

    const refreshInterval = setInterval(async () => {
      try {
        // Check if token needs refresh (3 minutes before expiry)
        if (MultiCharacterTokenStorage.shouldRefresh()) {
          console.log("[AuthContext] Token expiring soon, refreshing...");
          await performTokenRefresh();
        }
      } catch (error) {
        console.error("[AuthContext] Background refresh check failed:", error);
      }
    }, 60 * 1000); // Check every 60 seconds

    return () => clearInterval(refreshInterval);
  }, [isAuthenticated]);

  const performTokenRefresh = async () => {
    try {
      const activeChar = MultiCharacterTokenStorage.getActiveCharacter();
      
      if (!activeChar || !activeChar.refreshToken) {
        console.warn("[AuthContext] No refresh token available");
        logout();
        return;
      }

      console.log("[AuthContext] Refreshing access token for", activeChar.characterName);
      
      const newToken = await refreshAccessToken(activeChar.refreshToken, EVE_CLIENT_ID);
      
      // Update token in multi-character storage
      MultiCharacterTokenStorage.updateCharacterToken(activeChar.characterID, newToken);
      setAccessToken(newToken.access_token);
      
      console.log("[AuthContext] Token refreshed successfully");
    } catch (error) {
      console.error("[AuthContext] Token refresh failed:", error);
      // On refresh failure, logout user
      logout();
    }
  };

  const toCharacterInfo = (stored: any): CharacterInfo => ({
    character_id: stored.characterID,
    character_name: stored.characterName,
    scopes: stored.scopes ? stored.scopes.split(" ") : [],
    owner_hash: stored.ownerHash,
    portrait_url: `https://images.evetech.net/characters/${stored.characterID}/portrait?size=64`,
  });

  const checkSession = async () => {
    try {
      const activeChar = MultiCharacterTokenStorage.getActiveCharacter();
      const allChars = MultiCharacterTokenStorage.getAllCharacters();
      
      console.log("[AuthContext] Checking session");
      console.log("[AuthContext] Active character from storage:", activeChar ? `${activeChar.characterName} (ID: ${activeChar.characterID})` : "none");
      console.log("[AuthContext] All characters in storage:", allChars.map(c => `${c.characterName} (ID: ${c.characterID})`));
      
      if (!activeChar) {
        // No active character selected
        console.log("[AuthContext] No active character selected");
        setIsAuthenticated(false);
        setCharacter(null);
        setAllCharacters([]);
        setAccessToken(null);
        setIsLoading(false);
        return;
      }

      // Check if token is expired - if yes, try to refresh
      if (MultiCharacterTokenStorage.isExpired()) {
        console.log("[AuthContext] Token expired, attempting refresh...");
        try {
          await performTokenRefresh();
          // After refresh, re-check session
          const refreshedChar = MultiCharacterTokenStorage.getActiveCharacter();
          if (!refreshedChar) {
            console.log("[AuthContext] Refresh failed, clearing session");
            setIsAuthenticated(false);
            setCharacter(null);
            setAllCharacters([]);
            setAccessToken(null);
            setIsLoading(false);
            return;
          }
          // Continue with refreshed token
        } catch (error) {
          console.error("[AuthContext] Token refresh failed:", error);
          setIsAuthenticated(false);
          setCharacter(null);
          setAllCharacters([]);
          setAccessToken(null);
          setIsLoading(false);
          return;
        }
      }

      const currentChar = MultiCharacterTokenStorage.getActiveCharacter();
      if (!currentChar) {
        setIsAuthenticated(false);
        setCharacter(null);
        setAllCharacters([]);
        setAccessToken(null);
        setIsLoading(false);
        return;
      }

      console.log("[AuthContext] Verifying token with EVE ESI...");
      
      // Verify token with EVE ESI to check if it's still valid
      const charInfo = await verifyToken(currentChar.accessToken);
      
      console.log("[AuthContext] Token verified, character:", charInfo.CharacterName);
      
      // Use character info from storage (not from token verification)
      // This ensures we display the correct character that was selected
      const activeCharacterInfo: CharacterInfo = {
        character_id: currentChar.characterID,
        character_name: currentChar.characterName,
        scopes: currentChar.scopes ? currentChar.scopes.split(" ") : [],
        owner_hash: currentChar.ownerHash,
        portrait_url: `https://images.evetech.net/characters/${currentChar.characterID}/portrait?size=64`,
      };

      setCharacter(activeCharacterInfo);
      setAccessToken(currentChar.accessToken);
      setIsAuthenticated(true);
      
      // Load all characters
      const allStored = MultiCharacterTokenStorage.getAllCharacters();
      setAllCharacters(allStored.map(toCharacterInfo));
      
      console.log("[AuthContext] Session set, authenticated:", true, "Total characters:", allStored.length);
    } catch (error) {
      console.error("[AuthContext] Session verification failed:", error);
      setIsAuthenticated(false);
      setCharacter(null);
      setAllCharacters([]);
      setAccessToken(null);
    } finally {
      setIsLoading(false);
    }
  };

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

  const logout = async () => {
    console.log("[AuthContext] Logout initiated");
    
    // Revoke all tokens at EVE SSO before clearing locally
    const allChars = MultiCharacterTokenStorage.getAllCharacters();
    
    if (allChars.length > 0) {
      console.log(`[AuthContext] Revoking tokens for ${allChars.length} character(s)...`);
      
      // Revoke all access tokens (parallel)
      const revocationPromises = allChars.map(async (char) => {
        try {
          // Revoke access token
          await revokeToken(char.accessToken, EVE_CLIENT_ID);
          
          // Revoke refresh token if exists
          if (char.refreshToken) {
            await revokeToken(char.refreshToken, EVE_CLIENT_ID);
          }
          
          console.log(`[AuthContext] Revoked tokens for ${char.characterName}`);
        } catch (error) {
          console.error(`[AuthContext] Failed to revoke tokens for ${char.characterName}:`, error);
          // Continue with logout even if revocation fails
        }
      });
      
      // Wait for all revocations (max 5s timeout)
      await Promise.race([
        Promise.all(revocationPromises),
        new Promise((resolve) => setTimeout(resolve, 5000)),
      ]);
    }
    
    // Clear local storage
    MultiCharacterTokenStorage.clear();
    TokenStorage.clear(); // Legacy cleanup
    
    // Clear state
    setIsAuthenticated(false);
    setCharacter(null);
    setAllCharacters([]);
    setAccessToken(null);
    
    console.log("[AuthContext] Logout complete");
  };

  const logoutCharacter = async (characterID: number) => {
    console.log("[AuthContext] Logging out character:", characterID);
    
    // Get character data before removal for revocation
    const allChars = MultiCharacterTokenStorage.getAllCharacters();
    const charToRevoke = allChars.find(c => c.characterID === characterID);
    
    if (charToRevoke) {
      try {
        // Revoke tokens at EVE SSO
        console.log(`[AuthContext] Revoking tokens for ${charToRevoke.characterName}...`);
        await revokeToken(charToRevoke.accessToken, EVE_CLIENT_ID);
        
        if (charToRevoke.refreshToken) {
          await revokeToken(charToRevoke.refreshToken, EVE_CLIENT_ID);
        }
        
        console.log(`[AuthContext] Tokens revoked for ${charToRevoke.characterName}`);
      } catch (error) {
        console.error(`[AuthContext] Failed to revoke tokens for ${charToRevoke.characterName}:`, error);
        // Continue with local logout even if revocation fails
      }
    }
    
    // Remove character from local storage
    MultiCharacterTokenStorage.removeCharacter(characterID);
    
    // Reload all characters
    const remaining = MultiCharacterTokenStorage.getAllCharacters();
    setAllCharacters(remaining.map(toCharacterInfo));
    
    // If removed character was active, check session to load new active
    if (character?.character_id === characterID) {
      checkSession();
    }
  };

  const switchCharacter = async (characterID: number) => {
    console.log("[AuthContext] Switching to character:", characterID);
    const success = MultiCharacterTokenStorage.setActiveCharacter(characterID);
    console.log("[AuthContext] setActiveCharacter success:", success);
    
    if (success) {
      // Force reload session with new active character
      await checkSession();
    } else {
      console.error("[AuthContext] Failed to switch character - character not found");
    }
  };

  const getAuthHeader = (): string | null => {
    if (!accessToken) return null;
    return `Bearer ${accessToken}`;
  };

  const refreshSession = async () => {
    await checkSession();
  };

  return (
    <AuthContext.Provider
      value={{
        isAuthenticated,
        character,
        allCharacters,
        isLoading,
        accessToken,
        login,
        logout,
        logoutCharacter,
        switchCharacter,
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
