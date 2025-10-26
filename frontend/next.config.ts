import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: 'standalone',
  images: {
    remotePatterns: [
      {
        protocol: 'https',
        hostname: 'images.evetech.net',
        pathname: '/characters/**',
      },
    ],
  },
  experimental: {
    // Enable experimental features if needed
  },
};

export default nextConfig;
