# Stage 1: Dependencies
FROM node:26-alpine AS deps
WORKDIR /app

# Copy package files
COPY package.json package-lock.json* ./
RUN npm ci

# Stage 2: Builder
FROM node:26-alpine AS builder
WORKDIR /app

# NEXT_PUBLIC_* vars must be available at build time (baked into JS bundle)
ARG NEXT_PUBLIC_API_URL
ARG NEXT_PUBLIC_EVE_CLIENT_ID
ARG NEXT_PUBLIC_EVE_CALLBACK_URL
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL
ENV NEXT_PUBLIC_EVE_CLIENT_ID=$NEXT_PUBLIC_EVE_CLIENT_ID
ENV NEXT_PUBLIC_EVE_CALLBACK_URL=$NEXT_PUBLIC_EVE_CALLBACK_URL

# Copy dependencies from deps stage
COPY --from=deps /app/node_modules ./node_modules
COPY . .

# Build Next.js app
RUN npm run build

# Stage 3: Runner
FROM node:26-alpine AS runner
WORKDIR /app

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

# Create non-root user
RUN addgroup --system --gid 1001 nodejs
RUN adduser --system --uid 1001 nextjs

# Copy built app
COPY --from=builder /app/public ./public
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

USER nextjs

EXPOSE 3000

ENV PORT=3000
ENV HOSTNAME="0.0.0.0"

CMD ["node", "server.js"]
