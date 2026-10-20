import type { NextConfig } from "next";

const apiTarget =
  process.env.API_INTERNAL_URL?.replace(/\/$/, "") || "http://localhost:8080";

const nextConfig: NextConfig = {
  images: {
    remotePatterns: [
      { protocol: "http", hostname: "localhost", port: "8080", pathname: "/media/**" },
      { protocol: "https", hostname: "**" },
    ],
  },
  async rewrites() {
    return [
      {
        source: "/api/:path*",
        destination: `${apiTarget}/api/:path*`,
      },
      {
        source: "/media/:path*",
        destination: `${apiTarget}/media/:path*`,
      },
    ];
  },
};

export default nextConfig;
