"use client";

import Image from "next/image";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { useAuth } from "@/lib/auth-context";
import { ChevronDown, LogOut, Plus, UserCircle } from "lucide-react";

export function CharacterSwitcher() {
  const { character, allCharacters, switchCharacter, logoutCharacter, login } = useAuth();
  const [isOpen, setIsOpen] = useState(false);

  if (!character || allCharacters.length === 0) {
    return null;
  }

  const handleSwitchCharacter = async (characterID: number) => {
    if (characterID !== character.character_id) {
      await switchCharacter(characterID);
    }
    setIsOpen(false);
  };

  const handleRemoveCharacter = (characterID: number, event: React.MouseEvent) => {
    event.stopPropagation();
    if (confirm(`Remove ${allCharacters.find(c => c.character_id === characterID)?.character_name}?`)) {
      logoutCharacter(characterID);
    }
  };

  const handleAddCharacter = (event: React.MouseEvent) => {
    event.stopPropagation();
    setIsOpen(false);
    login();
  };

  return (
    <div className="relative">
      {/* Current Character Button */}
      <button
        onClick={() => setIsOpen(!isOpen)}
        className="flex items-center gap-2 rounded-lg p-2 hover:bg-accent transition-colors"
      >
        <Image
          src={character.portrait_url}
          alt={character.character_name}
          width={40}
          height={40}
          className="rounded-full"
        />
        <div className="hidden sm:block text-left">
          <p className="text-sm font-medium">{character.character_name}</p>
          <p className="text-xs text-muted-foreground">
            {allCharacters.length} Character{allCharacters.length !== 1 ? 's' : ''}
          </p>
        </div>
        <ChevronDown className={`h-4 w-4 transition-transform ${isOpen ? 'rotate-180' : ''}`} />
      </button>

      {/* Dropdown Menu */}
      {isOpen && (
        <>
          {/* Backdrop */}
          <div
            className="fixed inset-0 z-40"
            onClick={() => setIsOpen(false)}
          />
          
          {/* Dropdown Content */}
          <div className="absolute right-0 top-full mt-2 w-72 z-50 rounded-md border bg-popover p-2 shadow-md">
            <div className="text-xs font-medium text-muted-foreground px-2 py-1.5">
              Characters
            </div>
            
            {/* Character List */}
            <div className="space-y-1">
              {allCharacters.map((char) => {
                const isActive = char.character_id === character.character_id;
                
                return (
                  <div
                    key={char.character_id}
                    className={`
                      group flex items-center gap-3 rounded-md p-2 cursor-pointer
                      ${isActive ? 'bg-accent' : 'hover:bg-accent/50'}
                      transition-colors
                    `}
                    onClick={() => handleSwitchCharacter(char.character_id)}
                  >
                    <Image
                      src={char.portrait_url}
                      alt={char.character_name}
                      width={32}
                      height={32}
                      className="rounded-full"
                    />
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium truncate">
                        {char.character_name}
                        {isActive && (
                          <span className="ml-2 text-xs text-muted-foreground">(Active)</span>
                        )}
                      </p>
                      <p className="text-xs text-muted-foreground">
                        ID: {char.character_id}
                      </p>
                    </div>
                    
                    {/* Remove Button */}
                    {allCharacters.length > 1 && (
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8 opacity-0 group-hover:opacity-100 hover:bg-destructive hover:text-destructive-foreground"
                        onClick={(e) => handleRemoveCharacter(char.character_id, e)}
                        title="Remove character"
                      >
                        <LogOut className="h-4 w-4" />
                      </Button>
                    )}
                  </div>
                );
              })}
            </div>

            {/* Add Character Button */}
            <div className="mt-2 pt-2 border-t">
              <button
                onClick={handleAddCharacter}
                className="flex items-center gap-2 w-full rounded-md p-2 hover:bg-accent transition-colors text-sm"
              >
                <div className="flex h-8 w-8 items-center justify-center rounded-full bg-primary/10">
                  <Plus className="h-4 w-4" />
                </div>
                <span>Add Character</span>
              </button>
            </div>
          </div>
        </>
      )}
    </div>
  );
}
