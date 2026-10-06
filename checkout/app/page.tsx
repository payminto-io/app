import { ShieldCheck } from "lucide-react";

export default function Home() {
  return (
    <main className="empty-shell">
      <div className="brand-mark"><ShieldCheck aria-hidden="true" /></div>
      <p className="eyebrow">Payminto checkout</p>
      <h1>Open a payment link to continue</h1>
      <p>This checkout only accepts a valid payment reference issued by a merchant.</p>
    </main>
  );
}
