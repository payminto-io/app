import { Checkout } from "@/components/checkout";

export default async function CheckoutPage({ params }: { params: Promise<{ referenceId: string }> }) {
  const { referenceId } = await params;
  return <Checkout referenceId={referenceId} />;
}
