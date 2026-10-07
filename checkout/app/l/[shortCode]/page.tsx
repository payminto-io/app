import { notFound } from "next/navigation";
import { loadPublicLink, methodLabel, validShortCode } from "@/lib/links";
import { startPayment } from "./actions";

export const dynamic = "force-dynamic";

export default async function LinkPage({
  params,
  searchParams,
}: {
  params: Promise<{ shortCode: string }>;
  searchParams: Promise<{ error?: string }>;
}) {
  const { shortCode } = await params;
  if (!validShortCode(shortCode)) notFound();

  const link = await loadPublicLink(shortCode);
  if (!link) notFound();

  const { error } = await searchParams;
  const amount =
    link.amount_mode === "fixed" && link.amount
      ? `${link.amount} ${link.currency}`
      : `Amount at checkout (${link.currency})`;

  return (
    <main className="mx-auto flex min-h-screen max-w-md flex-col justify-center px-5 py-12">
      <div className="rounded-2xl border border-black/10 bg-white p-7 shadow-sm">
        {link.merchant_name ? (
          <p className="mb-5 text-[12px] font-semibold uppercase tracking-[0.14em] text-black/45">
            {link.merchant_name}
          </p>
        ) : null}

        <h1 className="text-[22px] font-bold tracking-tight text-black">{link.title}</h1>
        {link.description ? (
          <p className="mt-2 text-[14px] leading-[1.5] text-black/60">{link.description}</p>
        ) : null}

        <p className="mt-6 font-mono text-[28px] font-bold text-black">{amount}</p>

        {!link.available ? (
          <p className="mt-6 rounded-xl bg-black/[0.04] px-4 py-3 text-[13px] font-semibold text-black/60">
            This link is not accepting payments{link.unavailable_reason ? `: ${link.unavailable_reason}` : "."}
          </p>
        ) : (
          <>
            {error ? (
              <p className="mt-6 rounded-xl bg-red-50 px-4 py-3 text-[13px] font-semibold text-red-700">
                That payment could not be started. Please try again.
              </p>
            ) : null}

            <div className="mt-7 space-y-2.5">
              {link.methods.map((m) => (
                <form key={`${m.method}-${m.chain}-${m.asset}`} action={startPayment}>
                  <input type="hidden" name="short_code" value={link.short_code} />
                  <input type="hidden" name="method" value={m.method} />
                  <input type="hidden" name="chain" value={m.chain ?? ""} />
                  <input type="hidden" name="asset" value={m.asset ?? ""} />
                  <button
                    type="submit"
                    className="flex w-full items-center justify-between rounded-xl bg-black px-5 py-3.5 text-[14px] font-semibold text-white transition-opacity hover:opacity-90"
                  >
                    <span>Pay with {methodLabel(m)}</span>
                    <span aria-hidden>&rarr;</span>
                  </button>
                </form>
              ))}
            </div>

            {link.methods.length === 0 ? (
              <p className="mt-6 text-[13px] font-semibold text-black/50">
                No payment method is enabled on this link.
              </p>
            ) : null}
          </>
        )}
      </div>

      <p className="mt-5 text-center text-[12px] font-semibold text-black/40">
        Secured by Payminto · test environment
      </p>
    </main>
  );
}
