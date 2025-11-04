"use client";

import Link from "next/link";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Sheet, SheetContent, SheetTrigger } from "@/components/ui/sheet";
import { Menu } from "lucide-react";
import { useAuth } from "@/lib/auth-context";
import { EveLoginButton } from "@/components/eve-login-button";
import { CharacterInfo } from "@/components/character-info";

const navItems = [
  { href: "/", label: "Home" },
  { href: "/navigation", label: "Navigation" },
  { href: "/cargo", label: "Cargo" },
  { href: "/market", label: "Market" },
  { href: "/intra-region", label: "Intra-Region" },
  { href: "/inventory-sell", label: "Inventory Sell" },
];

export function Navigation() {
  const [isOpen, setIsOpen] = useState(false);
  const { isAuthenticated, isLoading } = useAuth();

  return (
    <nav className="border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="container mx-auto flex h-16 items-center justify-between px-4">
        {/* Logo */}
        <Link href="/" className="text-xl font-bold">
          EVE-O-Provit
        </Link>

        {/* Desktop Navigation */}
        <div className="hidden items-center gap-6 md:flex">
          {navItems.map((item) => (
            <Link
              key={item.href}
              href={item.href}
              className="text-sm font-medium transition-colors hover:text-primary"
            >
              {item.label}
            </Link>
          ))}
          
          {/* Auth Section */}
          <div className="ml-4 border-l pl-4">
            {!isLoading && (isAuthenticated ? <CharacterInfo /> : <EveLoginButton />)}
          </div>
        </div>

        {/* Mobile Navigation */}
        <div className="flex items-center gap-2 md:hidden">
          {!isLoading && (isAuthenticated ? <CharacterInfo /> : <EveLoginButton />)}
          
          <Sheet open={isOpen} onOpenChange={setIsOpen}>
            <SheetTrigger asChild>
              <Button variant="ghost" size="icon">
                <Menu className="h-6 w-6" />
                <span className="sr-only">Toggle menu</span>
              </Button>
            </SheetTrigger>
            <SheetContent side="right" className="w-[240px] sm:w-[300px]">
              <nav className="flex flex-col gap-4">
                {navItems.map((item) => (
                  <Link
                    key={item.href}
                    href={item.href}
                    onClick={() => setIsOpen(false)}
                    className="text-lg font-medium transition-colors hover:text-primary"
                  >
                    {item.label}
                  </Link>
                ))}
              </nav>
            </SheetContent>
          </Sheet>
        </div>
      </div>
    </nav>
  );
}
