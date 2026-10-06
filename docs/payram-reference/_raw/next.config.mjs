/** @type {import('next').NextConfig} */

const nextConfig = {
  env: {
    NEXT_PUBLIC_INFURA_SEPOLIA_NODE: process.env.NEXT_PUBLIC_INFURA_SEPOLIA_NODE,
    NEXT_PUBLIC_INFURA_PROJECT_ID: process.env.NEXT_PUBLIC_INFURA_PROJECT_ID,
    NEXT_PUBLIC_BACKEND_URL: process.env.NEXT_PUBLIC_BACKEND_URL,
    NEXT_PUBLIC_API_KEY: process.env.NEXT_PUBLIC_API_KEY,
    NEXT_PUBLIC_TRON_API_KEY: process.env.NEXT_PUBLIC_TRON_API_KEY,
    NEXT_PUBLIC_SWEEP_BATCH_SIZE: process.env.NEXT_PUBLIC_SWEEP_BATCH_SIZE,
    NEXT_PUBLIC_APPROVAL_BATCH_SIZE: process.env.NEXT_PUBLIC_APPROVAL_BATCH_SIZE,
    NEXT_PUBLIC_ENVIRONMENT: process.env.NEXT_PUBLIC_ENVIRONMENT,
    NEXT_PUBLIC_PAYMENTS_API_KEY: process.env.NEXT_PUBLIC_PAYMENTS_API_KEY,
    NEXT_PUBLIC_TRON_HTTP_PROVIDER: process.env.NEXT_PUBLIC_TRON_HTTP_PROVIDER,
  },
  images: {
    remotePatterns: [],
    unoptimized: true,
  },
  webpack: (config, { isServer }) => {
    // Enable WebAssembly and layers support
    config.experiments = {
      asyncWebAssembly: true,
      syncWebAssembly: true,
      layers: true,
    };

    // Fix for vanilla-extract and RainbowKit ESM compatibility
    config.externals = config.externals || [];
    if (isServer) {
      config.externals.push({
        '@vanilla-extract/sprinkles/createUtils': '@vanilla-extract/sprinkles/createUtils',
      });
    }

    // Required for tiny-secp256k1 library to work properly
    if (!isServer) {
      config.resolve.fallback = {
        fs: false,
        path: false,
        crypto: false,
      };
    }

    return config;
  },
  transpilePackages: [
    '@rainbow-me/rainbowkit',
    '@vanilla-extract/sprinkles',
    'react-responsive-modal',
  ],
  // Empty turbopack config to allow both turbopack (dev) and webpack (build) modes
  turbopack: {},
};

export default nextConfig;
