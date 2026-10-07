// What the platform is, in one picture: every rail lands on one ledger inside
// the merchant's own deployment, and payouts leave from there. Inline SVG so it
// stays crisp at any size and follows the theme via currentColor.
export function PlatformDiagram() {
  const rails = ["Card", "UPI", "USDC / USDT"];
  return (
    <svg
      viewBox="0 0 360 200"
      role="img"
      aria-label="Card, UPI and stablecoin rails all post to one double-entry ledger inside your own deployment, and payouts settle to your wallets"
      className="h-auto w-full max-w-[360px] text-foreground"
    >
      {/* rails in */}
      {rails.map((label, i) => {
        const y = 34 + i * 48;
        return (
          <g key={label}>
            <rect x="0" y={y - 13} width="92" height="26" rx="13" className="fill-surface-soft" />
            <text x="46" y={y + 4} textAnchor="middle" className="fill-current text-[11px] font-semibold">
              {label}
            </text>
            <path
              d={`M96 ${y} H126 Q134 ${y} 134 ${y > 82 ? y - 8 : y + 8} V${y > 82 ? 90 : 74} Q134 82 142 82 H150`}
              fill="none"
              stroke="currentColor"
              strokeWidth="1.25"
              strokeOpacity="0.45"
            />
          </g>
        );
      })}

      {/* the deployment the merchant owns */}
      <rect
        x="150" y="30" width="112" height="104" rx="14"
        fill="none" stroke="currentColor" strokeWidth="1.5" strokeDasharray="4 4" strokeOpacity="0.5"
      />
      <text x="206" y="24" textAnchor="middle" className="fill-current text-[9.5px] font-bold uppercase tracking-[0.14em]" opacity="0.55">
        your server
      </text>
      <rect x="164" y="46" width="84" height="30" rx="9" className="fill-surface-mint" />
      <text x="206" y="65" textAnchor="middle" className="fill-brand-ink text-[11px] font-bold">
        Payminto
      </text>
      <rect x="164" y="88" width="84" height="32" rx="9" fill="none" stroke="currentColor" strokeWidth="1.25" strokeOpacity="0.5" />
      <text x="206" y="101" textAnchor="middle" className="fill-current text-[10px] font-semibold">
        one ledger
      </text>
      <text x="206" y="113" textAnchor="middle" className="fill-current text-[9px] font-semibold" opacity="0.6">
        double entry
      </text>
      <path d="M206 76 V88" stroke="currentColor" strokeWidth="1.25" strokeOpacity="0.45" />

      {/* payout */}
      <path d="M262 82 H296" fill="none" stroke="currentColor" strokeWidth="1.25" strokeOpacity="0.45" />
      <path d="M292 78 l5 4 -5 4" fill="none" stroke="currentColor" strokeWidth="1.25" strokeOpacity="0.6" />
      <rect x="300" y="69" width="60" height="26" rx="13" className="fill-surface-soft" />
      <text x="330" y="86" textAnchor="middle" className="fill-current text-[11px] font-semibold">
        your wallet
      </text>

      {/* optional attestation */}
      <path d="M206 134 V158" stroke="currentColor" strokeWidth="1.25" strokeOpacity="0.3" strokeDasharray="3 3" />
      <text x="206" y="174" textAnchor="middle" className="fill-current text-[9.5px] font-semibold" opacity="0.55">
        Chainlink CRE attestation
      </text>
      <text x="206" y="187" textAnchor="middle" className="fill-current text-[9px] font-semibold" opacity="0.4">
        optional
      </text>
    </svg>
  );
}
