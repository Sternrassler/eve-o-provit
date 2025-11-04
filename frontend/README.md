# EVE-O-Provit Frontend

Next.js 14+ Frontend für Trading & Manufacturing Optimierung in EVE Online.

## Tech Stack

- **Framework**: Next.js 14.x (App Router, Server Components)
- **Language**: TypeScript 5.x
- **Styling**: Tailwind CSS 4.x
- **UI Components**: shadcn/ui (Radix UI + Tailwind)
- **Icons**: lucide-react
- **State Management**: React Context API
- **HTTP Client**: Fetch API (native)
- **Testing**: Playwright (E2E Tests)

## Features

### Implemented ✅

- **EVE SSO Authentication** - OAuth2 Login mit Character Management
- **Intra-Region Trading** - Route Calculation mit Profit-Optimierung
- **Inventory Sell** - Best Sell Location Finder
- **Market Data Management**:
  - Region Refresh Button (parallel fetch)
  - Staleness Indicator (Datenalter-Anzeige)
  - Automatic staleness tracking (30s interval)
- **Character Integration**:
  - Auto-detect current location/region
  - Ship selection
  - Autopilot waypoint setting
- **Item Search** - Autocomplete mit ESI Integration
- **Responsive Design** - Mobile-First Layout
- **Dark Mode** - via Tailwind

### Planned 🚧

- Multi-Region Trading
- Manufacturing Calculator
- Price History Charts
- Portfolio Tracking

## Getting Started

```bash
# Install dependencies
npm install

# Run development server
npm run dev
```

Open [http://localhost:9000](http://localhost:9000) with your browser.

## Environment Variables

Create `frontend/.env.local`:

```env
# Backend API URL
NEXT_PUBLIC_API_URL=http://localhost:9001

# EVE SSO Configuration
NEXT_PUBLIC_EVE_CLIENT_ID=0828b4bcd20242aeb9b8be10f5451094
NEXT_PUBLIC_EVE_CALLBACK_URL=http://localhost:9001/api/v1/auth/callback
```

## Project Structure

```
frontend/
├── src/
│   ├── app/                      # Next.js App Router
│   │   ├── layout.tsx            # Root Layout (Navigation)
│   │   ├── page.tsx              # Landing Page
│   │   ├── intra-region/         # Intra-Region Trading
│   │   ├── inventory-sell/       # Inventory Sell Optimizer
│   │   ├── character/            # Character Management
│   │   └── callback/             # OAuth Callback
│   ├── components/
│   │   ├── ui/                   # shadcn/ui Base Components
│   │   └── trading/              # Trading Components
│   │       ├── RegionSelect.tsx
│   │       ├── RegionRefreshButton.tsx
│   │       ├── RegionStalenessIndicator.tsx
│   │       ├── TradingRouteList.tsx
│   │       ├── TradingFilters.tsx
│   │       └── ShipSelect.tsx
│   ├── lib/
│   │   ├── api-client.ts         # Backend API Client
│   │   ├── auth-context.tsx      # Auth State Management
│   │   └── utils.ts              # Utility Functions
│   └── types/
│       └── trading.ts            # TypeScript Definitions
├── tests/                        # Playwright E2E Tests
├── public/                       # Static Assets
└── Dockerfile                    # Production Build
```

## Key Components

### RegionSelect

Region-Auswahl mit integriertem Refresh-Button und Staleness-Indikator.

```tsx
<RegionSelect
  value={regionId}
  onChange={setRegionId}
  showStaleness={true}
  showRefresh={true}
  onRefreshComplete={() => console.log('Refreshed')}
/>
```

**Features:**

- Auto-load regions from SDE
- Integrated refresh button
- Real-time staleness indicator (color-coded: green < 5min, yellow < 15min, orange > 15min)
- Auto-refresh staleness every 30s

### RegionRefreshButton

Manueller Market Data Refresh für eine Region.

```tsx
<RegionRefreshButton
  regionId="10000002"
  disabled={false}
  onRefreshComplete={() => {}}
/>
```

**Behavior:**

- Triggers full region refresh (~45s for The Forge)
- Shows toast notifications (start + completion)
- Spinner animation during fetch
- Disabled state handling

### RegionStalenessIndicator

Zeigt das Alter der Market-Daten für eine Region.

```tsx
<RegionStalenessIndicator
  regionId="10000002"
  className="mt-2"
/>
```

**Display:**

- 🕐 Icon + formatted time (e.g., "16 min", "2h 30m")
- Color-coded: 🟢 Green (< 5min), 🟡 Yellow (5-15min), 🟠 Orange (> 15min)
- Tooltip with exact timestamp and total orders
- Auto-refresh every 30 seconds

## API Integration

### Authentication Flow

```typescript
const { isAuthenticated, character, login, logout } = useAuth();

// Login
await login(); // Redirects to EVE SSO

// Check session
if (isAuthenticated) {
  console.log(character?.character_name);
}

// Logout
await logout();
```

### Market Data

```typescript
// Get market orders (from cache/DB)
const response = await fetch(
  `${API_URL}/api/v1/market/${regionId}/${typeId}`
);

// Refresh market data (parallel fetch)
const response = await fetch(
  `${API_URL}/api/v1/market/${regionId}/${typeId}?refresh=true`
);

// Check staleness
const staleness = await fetch(
  `${API_URL}/api/v1/market/staleness/${regionId}`
);
```

## Commands

```bash
# Development
npm run dev          # Start dev server (http://localhost:9000)
npm run build        # Build for production
npm start            # Start production server
npm run lint         # Run ESLint

# Testing
npm run test         # Run Playwright tests
npm run test:ui      # Playwright UI mode
```

## Docker

```bash
# Build Docker image
docker build -t eve-o-provit-frontend .

# Run container
docker run -p 9000:3000 eve-o-provit-frontend

# Or use Docker Compose (from project root)
make docker-up
```

**Ports:**

- Development: `localhost:9000` (via Docker Compose)
- Production: Container port `3000` → Host port `9000`

## Mobile-First Design

Responsive Breakpoints (Tailwind):

- `xs`: 375px (Mobile)
- `sm`: 640px (Large Mobile)
- `md`: 768px (Tablet)
- `lg`: 1024px (Desktop)
- `xl`: 1280px (Large Desktop)
- `2xl`: 1536px (XL Desktop)

## UI Components (shadcn/ui)

Installierte Components:

- Button, Card, Input, Select
- Navigation Menu, Sheet (Mobile Drawer)
- Toast, Dialog, Tabs
- Table, Checkbox, Label

Weitere hinzufügen:

```bash
npx shadcn@latest add [component-name]
```

## Testing

E2E Tests mit Playwright:

```bash
# Run all tests
npm run test

# Run specific test
npx playwright test tests/auth.spec.ts

# UI mode (interactive)
npm run test:ui

# Debug mode
npx playwright test --debug
```

## Performance

**Optimizations:**

- Server Components (React 18)
- Static Generation where possible
- Code Splitting (automatic via Next.js)
- Image Optimization (next/image)
- Font Optimization (next/font)

**Metrics:**

- Initial Page Load: < 1s
- Route Rendering: < 500ms
- Market Data Refresh: ~45s (backend limitation)

## Learn More

- [Next.js Documentation](https://nextjs.org/docs)
- [Tailwind CSS](https://tailwindcss.com/docs)
- [shadcn/ui](https://ui.shadcn.com/)
- [Radix UI](https://www.radix-ui.com/)
- [Playwright](https://playwright.dev/)

## License

Siehe Root-Projekt LICENSE
