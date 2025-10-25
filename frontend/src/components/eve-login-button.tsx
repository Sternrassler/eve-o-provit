"use client";

import { Button } from "@/components/ui/button";
import { useAuth } from "@/lib/auth-context";

export function EveLoginButton() {
  const { login, isLoading } = useAuth();

  return (
    <Button
      onClick={login}
      disabled={isLoading}
      className="bg-[#0098da] hover:bg-[#0087c2] text-white"
    >
      {isLoading ? "Loading..." : "Login with EVE"}
    </Button>
  );
}
