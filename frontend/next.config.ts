import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Phase 33: production image runs the standalone output
  // (.next/standalone/server.js) so the runtime image needs no node_modules.
  // `next dev` (compose) is unaffected.
  output: "standalone",
};

export default nextConfig;
