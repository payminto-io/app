import { renderToStaticMarkup } from "react-dom/server";
import { QRCodeSVG } from "qrcode.react";
import { describe, expect, it } from "vitest";
import { fromApi } from "../lib/model";
import { ChainPay } from "./chain-pay";

const owner = "5V9fNFkbp388PpANzYLMx3NPbNqEJPcqYMnKF1tw8gkK";
const tokenAccount = "6YCE2sL1gZZfXQxeJ1eSwqneEwVDztyNsmmtB2oR2Asx";

describe("Solana payer recipient", () => {
  it("uses the wallet owner for both the copy field and QR after mapping the public payment", () => {
    const payment = fromApi({
      referenceID: "solana-recipient-test",
      amountInUSD: "25",
      state: "OPEN",
      depositAddress: tokenAccount,
      depositOwnerAddress: owner,
      blockchainCode: "SOLANA",
      currencyCode: "USDC",
    }, []);
    const html = renderToStaticMarkup(<ChainPay chain={payment.chain!} />);
    expect(html).toContain('aria-label="Copy usdc address on solana"');
    expect(html).toContain(owner);
    expect(html).not.toContain(tokenAccount);
    const walletQR = renderToStaticMarkup(<QRCodeSVG value={owner} size={168} level="M" marginSize={0} title="USDC address on Solana" bgColor="#ffffff" fgColor="#15181d" />);
    expect(html).toContain(walletQR);
  });
});
