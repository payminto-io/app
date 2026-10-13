import * as anchor from "@coral-xyz/anchor";
import { Program } from "@coral-xyz/anchor";
import {
  createMint,
  createAccount,
  mintTo,
  getAccount,
  TOKEN_PROGRAM_ID,
  TOKEN_2022_PROGRAM_ID,
} from "@solana/spl-token";
import { Keypair, LAMPORTS_PER_SOL, PublicKey } from "@solana/web3.js";
import { createHash } from "crypto";
import { expect } from "chai";
import { PaymintoReceipts } from "../target/types/payminto_receipts";

describe("payminto-receipts", () => {
  const provider = anchor.AnchorProvider.env();
  anchor.setProvider(provider);
  const program = anchor.workspace.PaymintoReceipts as Program<PaymintoReceipts>;
  const conn = provider.connection;

  const payer = Keypair.generate();
  const merchant = Keypair.generate();
  let mint: PublicKey;
  let otherMint: PublicKey;
  let payerToken: PublicKey;
  let merchantToken: PublicKey;
  let payerOtherToken: PublicKey;

  const invoiceId = (label: string) =>
    Array.from(createHash("sha256").update(label).digest());

  const receiptPda = (id: number[]) =>
    PublicKey.findProgramAddressSync(
      [Buffer.from("receipt"), merchant.publicKey.toBuffer(), Buffer.from(id)],
      program.programId
    )[0];

  const pay = (id: number[], amount: number, accts: Partial<Record<string, PublicKey>> = {}) =>
    program.methods
      .payInvoice(id, new anchor.BN(amount))
      .accountsPartial({
        payer: payer.publicKey,
        merchant: merchant.publicKey,
        mint,
        payerToken,
        merchantToken,
        receipt: receiptPda(id),
        tokenProgram: TOKEN_PROGRAM_ID,
        ...accts,
      })
      .signers([payer])
      .rpc();

  const balance = async (acct: PublicKey) => (await getAccount(conn, acct)).amount;

  before(async () => {
    const sig = await conn.requestAirdrop(payer.publicKey, 10 * LAMPORTS_PER_SOL);
    await conn.confirmTransaction(sig, "confirmed");
    mint = await createMint(conn, payer, payer.publicKey, null, 6);
    otherMint = await createMint(conn, payer, payer.publicKey, null, 6);
    payerToken = await createAccount(conn, payer, mint, payer.publicKey);
    merchantToken = await createAccount(conn, payer, mint, merchant.publicKey);
    payerOtherToken = await createAccount(conn, payer, otherMint, payer.publicKey);
    await mintTo(conn, payer, mint, payerToken, payer, 1_000_000_000);
    await mintTo(conn, payer, otherMint, payerOtherToken, payer, 1_000_000_000);
  });

  it("pays an invoice: moves tokens and writes the receipt", async () => {
    const id = invoiceId("inv_happy");
    const amount = 25_000_000;
    const [pBefore, mBefore] = [await balance(payerToken), await balance(merchantToken)];

    let event: any = null;
    const listener = program.addEventListener("invoicePaid", (e) => (event = e));
    await pay(id, amount);

    expect(await balance(payerToken)).to.equal(pBefore - BigInt(amount));
    expect(await balance(merchantToken)).to.equal(mBefore + BigInt(amount));

    const r = await program.account.receipt.fetch(receiptPda(id));
    expect(Buffer.from(r.invoiceId).equals(Buffer.from(id))).to.be.true;
    expect(r.payer.toBase58()).to.equal(payer.publicKey.toBase58());
    expect(r.merchant.toBase58()).to.equal(merchant.publicKey.toBase58());
    expect(r.mint.toBase58()).to.equal(mint.toBase58());
    expect(r.amount.toNumber()).to.equal(amount);
    expect(r.slot.toNumber()).to.be.greaterThan(0);
    expect(r.paidAt.toNumber()).to.be.greaterThan(0);

    for (let i = 0; i < 20 && !event; i++) await new Promise((res) => setTimeout(res, 100));
    await program.removeEventListener(listener);
    expect(event, "InvoicePaid event").to.not.be.null;
    expect(event.amount.toNumber()).to.equal(amount);
  });

  it("rejects paying the same invoice twice", async () => {
    const id = invoiceId("inv_double");
    await pay(id, 1_000_000);
    const before = await balance(merchantToken);
    try {
      await pay(id, 1_000_000);
      expect.fail("second payment should fail");
    } catch (e: any) {
      expect(String(e.logs ?? e)).to.match(/already in use/);
    }
    expect(await balance(merchantToken)).to.equal(before);
  });

  it("rejects a token account with the wrong mint", async () => {
    const id = invoiceId("inv_wrong_mint");
    try {
      await pay(id, 1_000_000, { payerToken: payerOtherToken });
      expect.fail("wrong mint should fail");
    } catch (e: any) {
      expect(e.error?.errorCode?.code).to.equal("MintMismatch");
    }
    expect(await conn.getAccountInfo(receiptPda(id))).to.be.null;
  });

  it("rejects a merchant token account not owned by the merchant", async () => {
    const id = invoiceId("inv_wrong_owner");
    try {
      await pay(id, 1_000_000, { merchantToken: payerToken });
      expect.fail("wrong merchant owner should fail");
    } catch (e: any) {
      expect(e.error?.errorCode?.code).to.equal("MerchantOwnerMismatch");
    }
  });

  it("rejects a zero amount", async () => {
    const id = invoiceId("inv_zero");
    try {
      await pay(id, 0);
      expect.fail("zero amount should fail");
    } catch (e: any) {
      expect(e.error?.errorCode?.code).to.equal("ZeroAmount");
    }
    expect(await conn.getAccountInfo(receiptPda(id))).to.be.null;
  });

  it("pays with a Token-2022 mint", async () => {
    const t22 = TOKEN_2022_PROGRAM_ID;
    const mint22 = await createMint(conn, payer, payer.publicKey, null, 6, undefined, undefined, t22);
    const from = await createAccount(conn, payer, mint22, payer.publicKey, undefined, undefined, t22);
    const to = await createAccount(conn, payer, mint22, merchant.publicKey, undefined, undefined, t22);
    await mintTo(conn, payer, mint22, from, payer, 5_000_000, [], undefined, t22);
    const id = invoiceId("inv_token_2022");
    await pay(id, 2_000_000, { mint: mint22, payerToken: from, merchantToken: to, tokenProgram: t22 });
    expect((await getAccount(conn, to, undefined, t22)).amount).to.equal(2_000_000n);
    const r = await program.account.receipt.fetch(receiptPda(id));
    expect(r.mint.toBase58()).to.equal(mint22.toBase58());
  });
});
