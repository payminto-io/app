//! Payminto receipts: direct-to-merchant invoice payment with on-chain idempotency.
//!
//! `pay_invoice` moves SPL tokens straight from the payer to the merchant (no
//! custody, no admin keys) and creates a `Receipt` PDA seeded by
//! ["receipt", merchant, invoice_id]. Because the PDA is created with `init`,
//! a second payment for the same merchant invoice fails atomically, so an
//! invoice can never be paid twice. Off-chain services (the Payminto gateway)
//! verify a payment by deriving and reading the Receipt PDA.

use anchor_lang::prelude::*;
use anchor_spl::token_interface::{
    transfer_checked, Mint, TokenAccount, TokenInterface, TransferChecked,
};

declare_id!("3aJrkdT9RtQGE9sFszGY9worUS7QEGKv5XXZgb74hVud");

pub const RECEIPT_SEED: &[u8] = b"receipt";

#[program]
pub mod payminto_receipts {
    use super::*;

    pub fn pay_invoice(ctx: Context<PayInvoice>, invoice_id: [u8; 32], amount: u64) -> Result<()> {
        require!(amount > 0, ReceiptError::ZeroAmount);

        transfer_checked(
            CpiContext::new(
                ctx.accounts.token_program.to_account_info(),
                TransferChecked {
                    from: ctx.accounts.payer_token.to_account_info(),
                    mint: ctx.accounts.mint.to_account_info(),
                    to: ctx.accounts.merchant_token.to_account_info(),
                    authority: ctx.accounts.payer.to_account_info(),
                },
            ),
            amount,
            ctx.accounts.mint.decimals,
        )?;

        let clock = Clock::get()?;
        let receipt = &mut ctx.accounts.receipt;
        receipt.invoice_id = invoice_id;
        receipt.payer = ctx.accounts.payer.key();
        receipt.merchant = ctx.accounts.merchant.key();
        receipt.mint = ctx.accounts.mint.key();
        receipt.amount = amount;
        receipt.slot = clock.slot;
        receipt.paid_at = clock.unix_timestamp;
        receipt.bump = ctx.bumps.receipt;

        emit!(InvoicePaid {
            invoice_id,
            payer: receipt.payer,
            merchant: receipt.merchant,
            mint: receipt.mint,
            amount,
            slot: receipt.slot,
            paid_at: receipt.paid_at,
        });
        Ok(())
    }
}

#[derive(Accounts)]
#[instruction(invoice_id: [u8; 32])]
pub struct PayInvoice<'info> {
    #[account(mut)]
    pub payer: Signer<'info>,

    /// CHECK: only used as the receipt seed and as the required owner of
    /// `merchant_token`; it never signs and holds no data we read.
    pub merchant: UncheckedAccount<'info>,

    #[account(mint::token_program = token_program)]
    pub mint: InterfaceAccount<'info, Mint>,

    #[account(
        mut,
        constraint = payer_token.mint == mint.key() @ ReceiptError::MintMismatch,
        constraint = payer_token.owner == payer.key() @ ReceiptError::PayerOwnerMismatch,
        token::token_program = token_program,
    )]
    pub payer_token: InterfaceAccount<'info, TokenAccount>,

    #[account(
        mut,
        constraint = merchant_token.mint == mint.key() @ ReceiptError::MintMismatch,
        constraint = merchant_token.owner == merchant.key() @ ReceiptError::MerchantOwnerMismatch,
        token::token_program = token_program,
    )]
    pub merchant_token: InterfaceAccount<'info, TokenAccount>,

    #[account(
        init,
        payer = payer,
        space = 8 + Receipt::INIT_SPACE,
        seeds = [RECEIPT_SEED, merchant.key().as_ref(), invoice_id.as_ref()],
        bump,
    )]
    pub receipt: Account<'info, Receipt>,

    pub token_program: Interface<'info, TokenInterface>,
    pub system_program: Program<'info, System>,
}

/// Proof that `invoice_id` of `merchant` was paid. One per (merchant, invoice).
#[account]
#[derive(InitSpace)]
pub struct Receipt {
    pub invoice_id: [u8; 32],
    pub payer: Pubkey,
    pub merchant: Pubkey,
    pub mint: Pubkey,
    pub amount: u64,
    pub slot: u64,
    pub paid_at: i64,
    pub bump: u8,
}

#[event]
pub struct InvoicePaid {
    pub invoice_id: [u8; 32],
    pub payer: Pubkey,
    pub merchant: Pubkey,
    pub mint: Pubkey,
    pub amount: u64,
    pub slot: u64,
    pub paid_at: i64,
}

#[error_code]
pub enum ReceiptError {
    #[msg("Payment amount must be greater than zero")]
    ZeroAmount,
    #[msg("Token account mint does not match the payment mint")]
    MintMismatch,
    #[msg("Payer token account is not owned by the payer")]
    PayerOwnerMismatch,
    #[msg("Merchant token account is not owned by the merchant")]
    MerchantOwnerMismatch,
}
