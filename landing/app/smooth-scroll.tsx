"use client";

import { useEffect } from "react";
import Lenis from "lenis";
import { gsap, ScrollTrigger, EASE_IN, MOTION_OK } from "./motion/gsap";

/**
 * Lenis smooth scroll plus the page-wide motion attributes:
 * - data-reveal: element rises 20px and fades in once.
 * - data-reveal-stagger > data-stagger-child: children do the same in sequence.
 * - data-parallax="<px>": element travels that many px across its data-parallax-scope
 *   (positive lags behind the page, negative runs ahead). Halved below 768px.
 * Everything is skipped under prefers-reduced-motion, including Lenis itself.
 */
export function SmoothScrollProvider({ children }: { children: React.ReactNode }) {
  useEffect(() => {
    const root = document.documentElement;
    const mm = gsap.matchMedia();
    let lenis: Lenis | null = null;
    const raf = (time: number) => lenis?.raf(time * 1000);

    mm.add(MOTION_OK, () => {
      lenis = new Lenis({
        duration: 1.1,
        easing: (t) => Math.min(1, 1.001 - Math.pow(2, -10 * t)),
        smoothWheel: true,
        touchMultiplier: 1.4,
      });
      lenis.on("scroll", ScrollTrigger.update);
      gsap.ticker.add(raf);
      gsap.ticker.lagSmoothing(0);

      gsap.utils.toArray<HTMLElement>("[data-reveal]").forEach((el) => {
        gsap.from(el, {
          y: 20,
          autoAlpha: 0,
          duration: 0.7,
          ease: EASE_IN,
          scrollTrigger: { trigger: el, start: "top 88%", once: true },
        });
      });

      gsap.utils.toArray<HTMLElement>("[data-reveal-stagger]").forEach((parent) => {
        const children = parent.querySelectorAll<HTMLElement>("[data-stagger-child]");
        if (!children.length) return;
        gsap.from(children, {
          y: 16,
          autoAlpha: 0,
          duration: 0.6,
          ease: EASE_IN,
          stagger: 0.06,
          scrollTrigger: { trigger: parent, start: "top 85%", once: true },
        });
      });

      const small = window.matchMedia("(max-width: 767px)").matches;
      gsap.utils.toArray<HTMLElement>("[data-parallax]").forEach((el) => {
        const travel = Number(el.dataset.parallax) || 0;
        if (!travel) return;
        const scope = el.closest<HTMLElement>("[data-parallax-scope]") ?? el;
        const atTop = scope.getBoundingClientRect().top + window.scrollY < window.innerHeight;
        gsap.to(el, {
          y: small ? travel / 2 : travel,
          ease: "none",
          scrollTrigger: {
            trigger: scope,
            start: atTop ? "top top" : "top bottom",
            end: "bottom top",
            scrub: true,
          },
        });
      });

      root.classList.add("motion-ready");
      return () => {
        gsap.ticker.remove(raf);
        lenis?.destroy();
        lenis = null;
      };
    });

    root.classList.add("motion-ready");
    ScrollTrigger.refresh();
    return () => mm.revert();
  }, []);

  return <>{children}</>;
}
