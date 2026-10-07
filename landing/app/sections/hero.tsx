"use client";

import { useRef } from "react";
import { gsap, EASE_IN } from "../motion/gsap";
import { useMotion } from "../motion/use-motion";
import { LINKS } from "./links";
import { Mark3D } from "./mark-3d";
import { ThemeImage } from "./theme-image";

const FACTS = [
  ["Self-hosted", "Your servers, your keys, your database."],
  ["One ledger", "Every rail posts the same double-entry lines."],
  ["MIT licensed", "Read it, run it, change it."],
] as const;

export function Hero() {
  const ref = useRef<HTMLElement>(null);

  useMotion(ref, () => {
    const q = gsap.utils.selector(ref);
    const tl = gsap.timeline({ defaults: { ease: EASE_IN } });
    tl.fromTo(q("[data-hero-line]"), { yPercent: 105 }, { yPercent: 0, duration: 0.9, stagger: 0.08 })
      .fromTo(q("[data-hero-in='copy']"), { autoAlpha: 0, y: 16 }, { autoAlpha: 1, y: 0, duration: 0.6, stagger: 0.06 }, 0.3)
      .fromTo(q("[data-hero-in='mark']"), { autoAlpha: 0, y: 24, scale: 0.94 }, { autoAlpha: 1, y: 0, scale: 1, duration: 1 }, 0.2)
      .fromTo(q("[data-hero-in='plane']"), { autoAlpha: 0, y: 40 }, { autoAlpha: 1, y: 0, duration: 0.9 }, 0.55)
      .fromTo(q("[data-hero-in='card']"), { autoAlpha: 0, y: 28 }, { autoAlpha: 1, y: 0, duration: 0.8, stagger: 0.12 }, 0.8);
    return () => tl.kill();
  });

  return (
    <section ref={ref} data-parallax-scope className="relative overflow-hidden pt-28 md:pt-36">
      <div className="mx-auto max-w-6xl px-4 sm:px-6">
        <div className="grid items-center gap-10 lg:grid-cols-[1.35fr_1fr] lg:gap-6">
          <div>
            <p data-hero-in="copy" className="t-kicker mb-5">
              Open-source, self-hosted payment gateway
            </p>
            <h1 className="t-hero mb-6">
              <span className="block overflow-hidden pb-[0.04em]">
                <span data-hero-line className="block">The payment gateway</span>
              </span>
              <span className="block overflow-hidden pb-[0.08em]">
                <span data-hero-line className="block text-ink-soft">you deploy and own.</span>
              </span>
            </h1>
            <p data-hero-in="copy" className="t-lead mb-8 max-w-xl">
              Run it on your own servers. Connect payment providers and stablecoins as peer
              rails on one double-entry ledger, onboard your merchants, and turn on a Chainlink
              CRE workflow when you want anyone to check that the gateway holds what it owes.
            </p>
            <div data-hero-in="copy" className="flex flex-wrap gap-3">
              <a href={LINKS.source} className="btn btn-primary">
                Deploy from GitHub
              </a>
              <a href={LINKS.app} className="btn btn-outline">
                Open the app
              </a>
            </div>
          </div>

          {/* Layer 1, the centrepiece. A rendered 3D mark can replace this at public/brand/3d/. */}
          <div data-parallax="60" className="relative mx-auto w-[150px] sm:w-[190px] lg:w-[250px]">
            <div data-hero-in="mark">
              <Mark3D />
            </div>
          </div>
        </div>

        <div className="relative mt-16 pb-20 md:mt-20 md:pb-28">
          {/* Layer 2, the product plane. */}
          <div data-parallax="30">
            <figure data-hero-in="plane" className="overflow-hidden rounded-[14px] border border-line bg-surface shadow-[var(--elev-3)]">
              <ThemeImage
                srcLight="/screens/dashboard-home.png"
                srcDark="/screens/dashboard-home-dark.png"
                alt="Payminto merchant dashboard home with paid payments, paid by asset, payments received per day and recent payments (sample data)"
                width={1440}
                height={1024}
                fetchPriority="high"
                sizes="(min-width: 1152px) 1104px, 100vw"
                className="h-auto w-full"
              />
            </figure>
          </div>

          {/* Layer 3, foreground pieces that run ahead of the page. */}
          <div data-parallax="-110" className="absolute -bottom-2 right-2 w-[36%] max-w-[250px] sm:right-6 md:-bottom-4 md:right-10">
            <figure data-hero-in="card" className="overflow-hidden rounded-[14px] border border-line bg-surface shadow-[var(--elev-3)]">
              <ThemeImage
                srcLight="/screens/checkout-confirming.png"
                srcDark="/screens/checkout-confirming-dark.png"
                alt="Hosted checkout confirming a USDC payment on Solana, with the Received, Final, Settled rail (sample data)"
                width={780}
                height={890}
                sizes="250px"
                className="h-auto w-full"
              />
            </figure>
          </div>

          <div data-parallax="-180" className="absolute -bottom-6 left-2 hidden w-[300px] sm:left-6 sm:block md:-bottom-10 md:left-10">
            <div data-hero-in="card" className="panel-raised p-4">
              <div className="mb-3 flex items-center justify-between gap-3">
                <span className="text-[13px] font-semibold text-ink">Journal for USDC.SOLANA</span>
                <span className="badge badge-mute">Illustrative</span>
              </div>
              <dl className="num space-y-1.5 font-mono text-[13px]">
                <div className="flex justify-between gap-4">
                  <dt className="text-ink">Dr Clearing</dt>
                  <dd className="text-ink">128.00</dd>
                </div>
                <div className="flex justify-between gap-4">
                  <dt className="pl-4 text-ink-soft">Cr Merchant balance</dt>
                  <dd className="text-ink">128.00</dd>
                </div>
              </dl>
              <div className="mt-3 border-t border-line pt-3">
                <div className="rail mb-1.5">
                  <span className="rail-seg"><span className="rail-fill" /></span>
                  <span className="rail-seg"><span className="rail-fill" /></span>
                  <span className="rail-seg" />
                </div>
                <div className="grid grid-cols-3 text-[12px] font-medium">
                  <span className="text-ink">Received</span>
                  <span className="text-ink">Final</span>
                  <span className="text-ink-faint">Settled</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <ul className="grid gap-6 border-t border-line py-10 sm:grid-cols-3">
          {FACTS.map(([title, line]) => (
            <li key={title}>
              <div className="text-[18px] font-semibold text-ink">{title}</div>
              <div className="mt-1 text-[15px] text-ink-soft">{line}</div>
            </li>
          ))}
        </ul>
      </div>
    </section>
  );
}
