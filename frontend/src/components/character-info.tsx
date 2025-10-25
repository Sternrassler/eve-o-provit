"use client";

import Image from "next/image";
import { Button } from "@/components/ui/button";
import { useAuth } from "@/lib/auth-context";
import { LogOut } from "lucide-react";

export function CharacterInfo() {
  const { character, logout } = useAuth();

  if (!character) {
    return null;
  }

  return (
    <div className="flex items-center gap-3">
      <Image
        src={character.portrait_url}
        alt={character.character_name}
        width={40}
        height={40}
        className="rounded-full"
      />
      <div className="hidden sm:block">
        <p className="text-sm font-medium">{character.character_name}</p>
        <p className="text-xs text-muted-foreground">
          ID: {character.character_id}
        </p>
      </div>
      <Button
        variant="ghost"
        size="icon"
        onClick={logout}
        title="Logout"
      >
        <LogOut className="h-4 w-4" />
      </Button>
    </div>
  );
}
