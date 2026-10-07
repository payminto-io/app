export function DemoVideo() {
  return (
    <section id="demo" className="border-t border-line py-20 md:py-28">
      <div className="mx-auto max-w-6xl px-4 sm:px-6">
        <p className="t-kicker mb-3">Demo</p>
        <h2 className="t-section mb-4 max-w-2xl">See it running</h2>
        <p className="t-lead mb-8 max-w-2xl">
          A walkthrough of Payminto: a payment link, the hosted checkout, the ledger and the
          Chainlink CRE solvency attestation.
        </p>
        <div className="overflow-hidden rounded-[14px] border border-line bg-surface shadow-[var(--elev-3)]">
          <video
            className="block aspect-video w-full"
            src="/video/payminto-demo.mp4"
            poster="/video/payminto-demo-poster.jpg"
            controls
            playsInline
            preload="metadata"
          >
            <a href="/video/payminto-demo.mp4">Download the demo video</a>
          </video>
        </div>
      </div>
    </section>
  );
}
