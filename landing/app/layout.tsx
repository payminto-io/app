import type { Metadata } from "next";
import { Inter, Geist_Mono } from "next/font/google";
import "./globals.css";
import { SmoothScrollProvider } from "./smooth-scroll";

const inter = Inter({
  variable: "--font-inter",
  subsets: ["latin"],
  weight: ["400", "500", "600", "700", "800", "900"],
  display: "swap",
});

const geistMono = Geist_Mono({
  variable: "--font-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "Payminto — The payment processor you actually own.",
  description:
    "Self-hosted crypto + card payments. Zero fees. Zero approvals. Yours by deployment, not by license.",
  openGraph: {
    title: "Payminto — The payment processor you actually own.",
    description:
      "Self-hosted crypto + card payments. Zero fees. Zero approvals.",
    images: ["/generated/og-image.png"],
    type: "website",
  },
  twitter: {
    card: "summary_large_image",
    title: "Payminto — The payment processor you actually own.",
    images: ["/generated/og-image.png"],
  },
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html
      lang="en"
      className={`${inter.variable} ${geistMono.variable} h-full antialiased`}
    >
      <body className="min-h-full flex flex-col bg-background text-foreground">
        <SmoothScrollProvider>{children}</SmoothScrollProvider>
      </body>
    </html>
  );
}
