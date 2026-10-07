"use client";

import Image from "next/image";
import { useState } from "react";

// Click-to-play facade: the poster is served from this origin and nothing is
// requested from YouTube until the viewer actually asks for the video.
export function DemoVideo({
  videoId,
  poster,
  title,
}: {
  videoId: string;
  poster: string;
  title: string;
}) {
  const [playing, setPlaying] = useState(false);

  if (playing) {
    return (
      <div className="relative aspect-video w-full bg-black">
        <iframe
          src={`https://www.youtube-nocookie.com/embed/${videoId}?autoplay=1&rel=0&modestbranding=1`}
          title={title}
          allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share"
          referrerPolicy="strict-origin-when-cross-origin"
          allowFullScreen
          className="absolute inset-0 h-full w-full border-0"
        />
      </div>
    );
  }

  return (
    <button
      type="button"
      onClick={() => setPlaying(true)}
      aria-label={`Play ${title}`}
      className="group relative block aspect-video w-full overflow-hidden bg-black focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-brand-ink"
    >
      <Image
        src={poster}
        alt=""
        fill
        sizes="(min-width: 1152px) 1100px, 100vw"
        className="object-cover transition-transform duration-500 group-hover:scale-[1.02]"
      />
      <span className="absolute inset-0 bg-black/20 transition-colors group-hover:bg-black/10" />
      <span className="absolute left-1/2 top-1/2 flex h-[76px] w-[76px] -translate-x-1/2 -translate-y-1/2 items-center justify-center rounded-full bg-white/95 shadow-float transition-transform duration-300 group-hover:scale-110">
        <svg viewBox="0 0 24 24" aria-hidden className="ml-[3px] h-7 w-7 fill-[#0e0f0c]">
          <path d="M8 5v14l11-7z" />
        </svg>
      </span>
    </button>
  );
}
