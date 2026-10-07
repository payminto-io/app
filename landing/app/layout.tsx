import type { Metadata } from "next";
import { IBM_Plex_Sans, IBM_Plex_Mono } from "next/font/google";
import "./globals.css";
import { SmoothScrollProvider } from "./smooth-scroll";

const plexSans = IBM_Plex_Sans({
  variable: "--font-plex-sans",
  subsets: ["latin"],
  weight: ["400", "500", "600"],
  display: "swap",
});

const plexMono = IBM_Plex_Mono({
  variable: "--font-plex-mono",
  subsets: ["latin"],
  weight: ["400", "500"],
  display: "swap",
});

const TITLE = "Payminto - the open-source payment gateway you actually own";
const DESCRIPTION =
  "Self-hostable payment gateway: payment providers and USDC/USDT on Solana as peer rails on one double-entry ledger, with optional Chainlink CRE solvency attestation.";

export const metadata: Metadata = {
  // Relative paths in the metadata below resolve against this; without it Next
  // falls back to http://localhost:3000 and ships localhost URLs in the tags.
  metadataBase: new URL(process.env.NEXT_PUBLIC_SITE_URL ?? "https://payminto.io"),
  title: TITLE,
  description: DESCRIPTION,
  openGraph: { title: TITLE, description: DESCRIPTION, type: "website" },
  twitter: { card: "summary", title: TITLE, description: DESCRIPTION },
};

// Marks the page for the hero entrance before paint; a 2.5s fallback reveals it if JS never runs the motion.
const MOTION_BOOT = `(function(){try{if(matchMedia('(prefers-reduced-motion: no-preference)').matches){var d=document.documentElement;d.classList.add('motion');setTimeout(function(){d.classList.add('motion-ready')},2500)}}catch(e){}})();`;

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html
      lang="en"
      className={`${plexSans.variable} ${plexMono.variable} h-full antialiased`}
      suppressHydrationWarning
    >
      <head>
        <script dangerouslySetInnerHTML={{ __html: MOTION_BOOT }} />
      </head>
      <body className="min-h-full bg-canvas text-ink">
        <SmoothScrollProvider>{children}</SmoothScrollProvider>
      </body>
    </html>
  );
}
