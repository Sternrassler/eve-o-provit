"use client";

import Link from "next/link";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Sheet, SheetContent, SheetTrigger } from "@/components/ui/sheet";
import {
  NavigationMenu,
  NavigationMenuContent,
  NavigationMenuItem,
  NavigationMenuLink,
  NavigationMenuList,
  NavigationMenuTrigger,
  navigationMenuTriggerStyle,
} from "@/components/ui/navigation-menu";
import { Menu } from "lucide-react";
import { useAuth } from "@/lib/auth-context";
import { EveLoginButton } from "@/components/eve-login-button";
import { CharacterInfo } from "@/components/character-info";

// "Trading" bundles the five buy→sell→profit tools under one menu; Mining and
// Character stay top-level. Mirrors the mobile (Flutter) client's grouping.
// Trends/Watchlist are intentionally absent: their pages are static Phase-3
// placeholders (no backend yet, see #47/#48), so the nav must not advertise them
// as working features.
const tradingLinks = [
  { href: "/trading", label: "Trading Routes" },
  { href: "/hauling", label: "Hauling" },
  { href: "/roi-calculator", label: "ROI Calculator" },
  { href: "/multi-hub", label: "Multi-Hub" },
  { href: "/sell-assets", label: "Sell Assets" },
] as const;

const topLevelLinks = [
  { href: "/mining", label: "Mining" },
  { href: "/character", label: "Character" },
] as const;

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
        <div className="hidden items-center gap-2 md:flex">
          <NavigationMenu>
            <NavigationMenuList>
              <NavigationMenuItem>
                <NavigationMenuTrigger>Trading</NavigationMenuTrigger>
                <NavigationMenuContent>
                  <ul className="grid w-[220px] gap-1 p-2">
                    {tradingLinks.map((item) => (
                      <li key={item.href}>
                        <NavigationMenuLink asChild>
                          <Link
                            href={item.href}
                            className="block rounded-md px-3 py-2 text-sm font-medium transition-colors hover:bg-accent hover:text-accent-foreground"
                          >
                            {item.label}
                          </Link>
                        </NavigationMenuLink>
                      </li>
                    ))}
                  </ul>
                </NavigationMenuContent>
              </NavigationMenuItem>
              {topLevelLinks.map((item) => (
                <NavigationMenuItem key={item.href}>
                  <NavigationMenuLink asChild className={navigationMenuTriggerStyle()}>
                    <Link href={item.href}>{item.label}</Link>
                  </NavigationMenuLink>
                </NavigationMenuItem>
              ))}
            </NavigationMenuList>
          </NavigationMenu>

          {/* Auth Section */}
          <div className="ml-2 border-l pl-4">
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
                <div className="flex flex-col gap-2">
                  <span className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                    Trading
                  </span>
                  {tradingLinks.map((item) => (
                    <Link
                      key={item.href}
                      href={item.href}
                      onClick={() => setIsOpen(false)}
                      className="pl-2 text-base font-medium transition-colors hover:text-primary"
                    >
                      {item.label}
                    </Link>
                  ))}
                </div>
                {topLevelLinks.map((item) => (
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
