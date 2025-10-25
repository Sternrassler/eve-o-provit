# EVE-O-Provit Frontend

Next.js 14+ Frontend mit TypeScript, Tailwind CSS und shadcn/ui.

## Tech Stack

- **Framework**: Next.js 14.x (App Router)
- **Language**: TypeScript 5.x
- **Styling**: Tailwind CSS 4.x
- **UI Components**: shadcn/ui (Radix UI + Tailwind)
- **Icons**: lucide-react
- **State Management**: React Context (Zustand geplant)

## Getting Started

```bash
# Install dependencies
npm install

# Run development server
npm run dev
```

Open [http://localhost:3000](http://localhost:3000) with your browser to see the result.

You can start editing the page by modifying `src/app/page.tsx`. The page auto-updates as you edit the file.

## Mobile-First Design

Responsive Breakpoints (Tailwind):
- `xs`: 375px (Mobile)
- `sm`: 640px (Large Mobile)
- `md`: 768px (Tablet)
- `lg`: 1024px (Desktop)
- `xl`: 1280px (Large Desktop)
- `2xl`: 1536px (XL Desktop)

## Features

- ✅ Landing Page mit Hero & Features
- ✅ Mobile Navigation (Hamburger Menu)
- ✅ Desktop Navigation
- ✅ Responsive Layout
- ✅ Dark Mode Support (via Tailwind)
- 🚧 Navigation Page (Placeholder)
- 🚧 Cargo Calculator (Placeholder)
- 🚧 Market Analysis (Placeholder)

## Commands

```bash
# Development
npm run dev          # Start dev server (http://localhost:3000)
npm run build        # Build for production
npm start            # Start production server
npm run lint         # Run ESLint
```

## Docker

```bash
# Build Docker image
docker build -t eve-o-provit-frontend .

# Run container
docker run -p 3000:3000 eve-o-provit-frontend

# Or use Docker Compose (from project root)
make docker-up
```

## Environment Variables

```env
NEXT_PUBLIC_API_URL=http://localhost:8082  # Backend API URL
NODE_ENV=development|production
```

## Project Structure

```
frontend/
├── src/
│   ├── app/                 # Next.js App Router
│   │   ├── layout.tsx       # Root Layout (Navigation)
│   │   ├── page.tsx         # Landing Page
│   │   ├── navigation/      # Navigation Feature
│   │   ├── cargo/           # Cargo Calculator
│   │   └── market/          # Market Analysis
│   ├── components/
│   │   ├── ui/              # shadcn/ui Components
│   │   └── navigation.tsx   # Navigation Component
│   └── lib/
│       └── utils.ts         # Utility Functions
├── public/                  # Static Assets
├── Dockerfile               # Production Build
├── next.config.ts           # Next.js Config (standalone output)
└── tailwind.config.ts       # Tailwind Config
```

## UI Components (shadcn/ui)

Installierte Components:
- Button
- Card
- Navigation Menu
- Sheet (Mobile Drawer)

Weitere hinzufügen:
```bash
npx shadcn@latest add [component-name]
```

## API Integration

Backend API läuft auf `http://localhost:8082` (via Docker Compose).

```typescript
// Beispiel API Call
const response = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/health`);
const data = await response.json();
```

## Learn More

- [Next.js Documentation](https://nextjs.org/docs)
- [Tailwind CSS](https://tailwindcss.com/docs)
- [shadcn/ui](https://ui.shadcn.com/)
- [Radix UI](https://www.radix-ui.com/)

## License

Siehe Root-Projekt LICENSE
