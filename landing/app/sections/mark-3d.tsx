"use client";

import { useRef } from "react";
import { gsap } from "../motion/gsap";
import { useMotion } from "../motion/use-motion";

const DEPTH_LAYERS = 12;
const LAYER_GAP = 1.6;

/**
 * The brand mark (public/brand/mark.svg) extruded with CSS 3D: stacked tile layers for depth,
 * bars lifted off the face, a contact shadow. Spin is scrubbed by scroll, tilt follows a fine pointer.
 * Under reduced motion the mark renders flat and still.
 */
export function Mark3D({ className = "" }: { className?: string }) {
  const ref = useRef<HTMLDivElement>(null);

  useMotion(ref, () => {
    const q = gsap.utils.selector(ref);
    const spin = q(".mark3d-spin")[0] as HTMLElement;
    const tilt = q(".mark3d-tilt")[0] as HTMLElement;

    const scroll = gsap.fromTo(
      spin,
      { rotationY: 0, rotationX: 0 },
      {
        rotationY: 28,
        rotationX: -10,
        ease: "none",
        scrollTrigger: { trigger: ref.current, start: "top 20%", end: "bottom top", scrub: 0.6 },
      },
    );

    const fine = window.matchMedia("(pointer: fine)").matches;
    if (!fine) return () => scroll.scrollTrigger?.kill();

    const toX = gsap.quickTo(tilt, "rotationX", { duration: 0.6, ease: "power3.out" });
    const toY = gsap.quickTo(tilt, "rotationY", { duration: 0.6, ease: "power3.out" });
    const onMove = (e: PointerEvent) => {
      const nx = e.clientX / window.innerWidth - 0.5;
      const ny = e.clientY / window.innerHeight - 0.5;
      toY(nx * 22);
      toX(-ny * 16);
    };
    window.addEventListener("pointermove", onMove, { passive: true });
    return () => {
      window.removeEventListener("pointermove", onMove);
      scroll.scrollTrigger?.kill();
    };
  });

  return (
    <div ref={ref} className={`mark3d-stage relative ${className}`} role="img" aria-label="Payminto mark">
      <div className="mark3d-rest" style={{ transformStyle: "preserve-3d" }}>
        <div className="mark3d-spin">
          <div className="mark3d-tilt">
            <div className="mark3d">
              {Array.from({ length: DEPTH_LAYERS }, (_, i) => (
                <span key={i} className="mark3d-layer" style={{ "--z": (i + 1) * LAYER_GAP } as React.CSSProperties} aria-hidden />
              ))}
              <span className="mark3d-face" aria-hidden />
              {[1, 2, 3].map((n) => (
                <span key={`s${n}`} className={`mark3d-bar-shade is-${n}`} aria-hidden />
              ))}
              {[1, 2, 3].map((n) => (
                <span key={`b${n}`} className={`mark3d-bar is-${n}`} aria-hidden />
              ))}
            </div>
          </div>
        </div>
      </div>
      <span className="mark3d-shadow" aria-hidden />
    </div>
  );
}
