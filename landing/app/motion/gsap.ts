"use client";

import { gsap } from "gsap";
import { ScrollTrigger } from "gsap/ScrollTrigger";
import { MotionPathPlugin } from "gsap/MotionPathPlugin";

if (typeof window !== "undefined") {
  gsap.registerPlugin(ScrollTrigger, MotionPathPlugin);
}

/** One easing family for the whole page - see docs/MOTION.md, rule 5. */
export const EASE_IN = "power3.out";

/** Animations run only when the reader has not asked for reduced motion. */
export const MOTION_OK = "(prefers-reduced-motion: no-preference)";

/** Pinned storytelling only where a pinned block fits the viewport. */
export const WIDE = "(min-width: 1024px) and (min-height: 640px)";

export { gsap, ScrollTrigger };
