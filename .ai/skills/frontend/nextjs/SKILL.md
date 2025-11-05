# Frontend Skill: Next.js App Router

**Tech Stack:** Next.js 16.0.0 + React 19.2.0

**Project:** eve-o-provit Web Application

---

## Architecture Patterns

### App Router Architecture

- **Server Components:** Default rendering mode (fetch data on server)
- **Client Components:** Interactive elements (`"use client"` directive)
- **File-Based Routing:** `app/` directory with `page.tsx`, `layout.tsx`
- **Nested Layouts:** Shared UI across route segments

### React Context for State

- **AuthProvider:** Global authentication state
- **Context API:** Share state without prop drilling
- **Client-Side Only:** Context requires `"use client"` directive

---

## Best Practices (Normative Requirements)

1. **Server Components by Default (SHOULD):** Only use `"use client"` when needed (interactivity, hooks, browser APIs)
2. **Minimize Client Bundles (SHOULD):** Keep server components for data fetching, static content
3. **Environment Variables (MUST):** `NEXT_PUBLIC_*` for client-side, no prefix for server-side
4. **Loading States (SHOULD):** Use `loading.tsx` for route-level loading UI
5. **Error Boundaries (SHOULD):** Use `error.tsx` for error handling
6. **Metadata API (SHOULD):** Define SEO metadata in `layout.tsx` or `page.tsx`
7. **Route Organization (SHOULD):** Group related routes in folders with `layout.tsx`

---

## Common Patterns

### 1. Page Component (Server Component)

```tsx
// app/intra-region/page.tsx
export default function IntraRegionPage() {
  return (
    <div className="container mx-auto p-4">
      <h1 className="text-3xl font-bold">Intra-Region Trading</h1>
      <RegionSelect /> {/* Client component */}
    </div>
  );
}
```

### 2. Client Component with State

```tsx
// components/RegionSelect.tsx
"use client";

import { useState } from "react";

export function RegionSelect() {
  const [regionId, setRegionId] = useState(10000002);
  
  return (
    <Select value={regionId.toString()} onValueChange={(val) => setRegionId(parseInt(val))}>
      {/* ... */}
    </Select>
  );
}
```

### 3. React Context (Auth State)

```tsx
// lib/auth-context.tsx
"use client";

import { createContext, useContext, useState, useEffect } from "react";

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  
  useEffect(() => {
    checkSession();
  }, []);
  
  return <AuthContext.Provider value={{ isAuthenticated }}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) throw new Error("useAuth must be used within AuthProvider");
  return context;
}
```

### 4. API Fetch Pattern

```tsx
const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:9001";

async function fetchMarketOrders(regionId: number, typeId: number) {
  const response = await fetch(`${API_URL}/api/v1/market/${regionId}/${typeId}`);
  if (!response.ok) throw new Error("Failed to fetch orders");
  return response.json();
}
```

---

## Anti-Patterns

❌ **Unnecessary Client Components:** Don't use `"use client"` for static content

❌ **Fetch in useEffect:** Prefer Server Components for data fetching (better performance)

❌ **Missing Loading States:** Always handle loading/error states in client components

❌ **Prop Drilling:** Use Context for deeply nested state, not prop drilling

❌ **Client-Side Environment Variables:** Don't expose secrets via `NEXT_PUBLIC_*`

---

## Integration with Radix UI

### Component Import Pattern

```tsx
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Button } from "@/components/ui/button";
import { toast } from "@/hooks/use-toast";
```

### Controlled Components

- Radix UI uses controlled pattern (`value`, `onValueChange`)
- State management via React `useState`
- Type-safe event handlers

---

## Performance Considerations

- **Server Components:** Zero client JavaScript for non-interactive content
- **Code Splitting:** Automatic per-route code splitting
- **Image Optimization:** Use `next/image` component (automatic optimization)
- **Font Optimization:** `next/font` for optimized font loading
- **Caching:** Server components are cached by default (use `cache: 'no-store'` to opt-out)

---

## Security Guidelines

- **Environment Variables:** Never expose API keys via `NEXT_PUBLIC_*`
- **CSRF Protection:** Use SameSite cookies, CSRF tokens for mutations
- **XSS Prevention:** React escapes JSX by default, avoid `dangerouslySetInnerHTML`
- **Auth Tokens:** Store in HttpOnly cookies, not localStorage (XSS-safe)

---

## Quick Reference

| Task | Pattern |
|------|---------|
| Page component | `export default function Page() { ... }` |
| Client component | `"use client"; export function Component() { ... }` |
| Metadata | `export const metadata: Metadata = { title: "..." }` |
| Loading UI | Create `loading.tsx` in route folder |
| Error UI | Create `error.tsx` in route folder |
| Layout | `export default function Layout({ children }) { ... }` |
| Dynamic route | `app/[id]/page.tsx` → `params.id` |
| API route | `app/api/route.ts` → `export async function GET() { ... }` |
