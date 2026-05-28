import type { NextConfig } from "next"

const nextConfig: NextConfig = {
  reactStrictMode: true,
  async rewrites() {
    return [
      {
        source: "/f/:slug",
        destination: "/forms/public/:slug",
      },
    ]
  },
}

export default nextConfig
