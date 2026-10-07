package solana

import "encoding/binary"

// SPL token instruction tags (program/src/instruction.rs).
const (
	tokenTransfer        byte = 3
	tokenMintTo          byte = 7
	tokenCloseAccount    byte = 9
	tokenTransferChecked byte = 12
	tokenInitializeMint2 byte = 20
)

// MintAccountSize is the byte size of an SPL mint account under the classic token program.
const MintAccountSize = 82

// SystemTransfer moves lamports.
func SystemTransfer(from, to PublicKey, lamports uint64) Instruction {
	data := binary.LittleEndian.AppendUint32(nil, 2)
	data = binary.LittleEndian.AppendUint64(data, lamports)
	return Instruction{ProgramID: SystemProgram, Data: data, Accounts: []AccountMeta{
		{Pubkey: from, Signer: true, Writable: true},
		{Pubkey: to, Writable: true},
	}}
}

// SystemCreateAccount allocates a new account owned by program; both from and the new account sign.
func SystemCreateAccount(from, newAccount PublicKey, lamports, space uint64, owner PublicKey) Instruction {
	data := binary.LittleEndian.AppendUint32(nil, 0)
	data = binary.LittleEndian.AppendUint64(data, lamports)
	data = binary.LittleEndian.AppendUint64(data, space)
	data = append(data, owner[:]...)
	return Instruction{ProgramID: SystemProgram, Data: data, Accounts: []AccountMeta{
		{Pubkey: from, Signer: true, Writable: true},
		{Pubkey: newAccount, Signer: true, Writable: true},
	}}
}

// TokenInitializeMint2 initializes a mint with no freeze authority.
func TokenInitializeMint2(tokenProgram, mint PublicKey, decimals uint8, mintAuthority PublicKey) Instruction {
	data := []byte{tokenInitializeMint2, decimals}
	data = append(data, mintAuthority[:]...)
	data = append(data, 0) // COption::None freeze authority
	return Instruction{ProgramID: tokenProgram, Data: data, Accounts: []AccountMeta{
		{Pubkey: mint, Writable: true},
	}}
}

// TokenMintTo mints amount base units to a token account.
func TokenMintTo(tokenProgram, mint, dest, authority PublicKey, amount uint64) Instruction {
	data := binary.LittleEndian.AppendUint64([]byte{tokenMintTo}, amount)
	return Instruction{ProgramID: tokenProgram, Data: data, Accounts: []AccountMeta{
		{Pubkey: mint, Writable: true},
		{Pubkey: dest, Writable: true},
		{Pubkey: authority, Signer: true},
	}}
}

// TokenTransfer is the unchecked transfer (no mint account, no decimals assertion).
func TokenTransfer(tokenProgram, source, dest, authority PublicKey, amount uint64) Instruction {
	data := binary.LittleEndian.AppendUint64([]byte{tokenTransfer}, amount)
	return Instruction{ProgramID: tokenProgram, Data: data, Accounts: []AccountMeta{
		{Pubkey: source, Writable: true},
		{Pubkey: dest, Writable: true},
		{Pubkey: authority, Signer: true},
	}}
}

// TokenTransferChecked transfers amount base units and lets the program assert mint and decimals.
func TokenTransferChecked(tokenProgram, source, mint, dest, authority PublicKey, amount uint64, decimals uint8) Instruction {
	data := binary.LittleEndian.AppendUint64([]byte{tokenTransferChecked}, amount)
	data = append(data, decimals)
	return Instruction{ProgramID: tokenProgram, Data: data, Accounts: []AccountMeta{
		{Pubkey: source, Writable: true},
		{Pubkey: mint},
		{Pubkey: dest, Writable: true},
		{Pubkey: authority, Signer: true},
	}}
}

// TokenCloseAccount closes an empty token account and sends its rent lamports to dest.
func TokenCloseAccount(tokenProgram, account, dest, owner PublicKey) Instruction {
	return Instruction{ProgramID: tokenProgram, Data: []byte{tokenCloseAccount}, Accounts: []AccountMeta{
		{Pubkey: account, Writable: true},
		{Pubkey: dest, Writable: true},
		{Pubkey: owner, Signer: true},
	}}
}

// CreateAssociatedTokenAccountIdempotent creates owner's ATA for mint if missing; payer funds rent.
func CreateAssociatedTokenAccountIdempotent(payer, owner, mint, tokenProgram PublicKey) (Instruction, PublicKey, error) {
	ata, err := AssociatedTokenAddress(owner, mint, tokenProgram)
	if err != nil {
		return Instruction{}, PublicKey{}, err
	}
	return Instruction{ProgramID: AssociatedTokenProgram, Data: []byte{1}, Accounts: []AccountMeta{
		{Pubkey: payer, Signer: true, Writable: true},
		{Pubkey: ata, Writable: true},
		{Pubkey: owner},
		{Pubkey: mint},
		{Pubkey: SystemProgram},
		{Pubkey: tokenProgram},
	}}, ata, nil
}

// ComputeBudgetSetUnitLimit caps compute units so the priority fee is bounded.
func ComputeBudgetSetUnitLimit(units uint32) Instruction {
	return Instruction{ProgramID: ComputeBudgetProgram, Data: binary.LittleEndian.AppendUint32([]byte{2}, units)}
}

// ComputeBudgetSetUnitPrice sets the priority fee in micro-lamports per compute unit.
func ComputeBudgetSetUnitPrice(microLamports uint64) Instruction {
	return Instruction{ProgramID: ComputeBudgetProgram, Data: binary.LittleEndian.AppendUint64([]byte{3}, microLamports)}
}
