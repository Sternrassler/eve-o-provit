"use client";

import React, { createContext, useContext, useState, useEffect } from "react";

interface CharacterInfo {
  character_id: number;
  character_name: string;
  scopes: string[];
  portrait_url: string;
}

interface AuthContextType {
  isAuthenticated: boolean;
  character: CharacterInfo | null;
  isLoading: boolean;
  login: () => void;
  logout: () => Promise<void>;
  refresh: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [character, setCharacter] = useState<CharacterInfo | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  // Verify session on mount and periodically
  useEffect(() => {
    verifySession();
    
    // Refresh session every 15 minutes
    const interval = setInterval(() => {
      refresh();
    }, 15 * 60 * 1000);
    
    return () => clearInterval(interval);
  }, []);

  const verifySession = async () => {
    try {
      const response = await fetch(`${API_URL}/api/v1/auth/verify`, {
        credentials: "include",
      });

      if (response.ok) {
        const data = await response.json();
        setIsAuthenticated(true);
        // Fetch character info for complete data
        await fetchCharacterInfo();
      } else {
        setIsAuthenticated(false);
        setCharacter(null);
      }
    } catch (error) {
      console.error("Failed to verify session:", error);
      setIsAuthenticated(false);
      setCharacter(null);
    } finally {
      setIsLoading(false);
    }
  };

  const fetchCharacterInfo = async () => {
    try {
      const response = await fetch(`${API_URL}/api/v1/auth/character`, {
        credentials: "include",
      });

      if (response.ok) {
        const data = await response.json();
        setCharacter(data);
      }
    } catch (error) {
      console.error("Failed to fetch character info:", error);
    }
  };

  const login = () => {
    // Redirect to backend login endpoint
    window.location.href = `${API_URL}/api/v1/auth/login`;
  };

  const logout = async () => {
    try {
      await fetch(`${API_URL}/api/v1/auth/logout`, {
        method: "POST",
        credentials: "include",
      });
      
      setIsAuthenticated(false);
      setCharacter(null);
    } catch (error) {
      console.error("Failed to logout:", error);
    }
  };

  const refresh = async () => {
    try {
      await fetch(`${API_URL}/api/v1/auth/refresh`, {
        method: "POST",
        credentials: "include",
      });
    } catch (error) {
      console.error("Failed to refresh session:", error);
    }
  };

  return (
    <AuthContext.Provider
      value={{
        isAuthenticated,
        character,
        isLoading,
        login,
        logout,
        refresh,
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
