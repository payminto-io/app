"use client";

import { useEffect, type RefObject } from "react";
import { gsap, MOTION_OK, WIDE } from "./gsap";

type Setup = (env: { wide: boolean }) => void | (() => void);

/**
 * Runs `setup` inside a gsap.matchMedia scoped to `scope`, only when motion is allowed.
 * The markup is the final state; setup tweens *from* a before-state, so reverting restores it.
 */
export function useMotion(scope: RefObject<HTMLElement | null>, setup: Setup) {
  useEffect(() => {
    const el = scope.current;
    if (!el) return;
    const mm = gsap.matchMedia(el);
    mm.add({ motion: MOTION_OK, wide: WIDE }, (ctx) => {
      const { motion, wide } = ctx.conditions as { motion: boolean; wide: boolean };
      if (!motion) return;
      return setup({ wide });
    });
    return () => mm.revert();
    // setup is defined inline by each section and is stable for the component's life.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);
}
