package circuits

import (
	"github.com/consensys/gnark-crypto/ecc/twistededwards"
	"github.com/consensys/gnark/frontend"
	gnark_twistededwards "github.com/consensys/gnark/std/algebra/native/twistededwards"
	"github.com/consensys/gnark/std/hash/mimc"
	"github.com/consensys/gnark/std/signature/eddsa"
)

// Operation type tags.
const (
	OpTrade      = 0
	OpTransfer   = 1
	OpWithdrawal = 2
	OpDeposit    = 3
)

const MerkleDepth = 28
const BatchSize = 25 // real fills per proof — must match sequencer/internal/batcher.BatchSize.
// A batch only seals once exactly this many real transitions have accumulated (no
// dummy padding), so this is also the minimum trade volume before anything settles
// to L1 — see sequencer/internal/batcher/batcher.go's AddTransition.

// StateUpdate represents the data needed to update a single leaf in the SMT.
//
// IsGenesis marks this leg's leaf as never having been written before — the very
// first time this index is ever touched by any transition. A Sparse Merkle Tree's
// true default for an untouched leaf is the literal field value 0 (see
// sequencer/internal/state/merkle_db.go's zeroHashes[0].SetZero()), NOT
// accountLeaf(index, pubX, pubY, 0, 0) — those are different field elements for any
// real pubkey. Every op type's "old leaf" check unconditionally hashed via
// accountLeaf() before this flag existed, which silently assumed every leaf had
// already been populated with a real accountLeaf(...) value by some earlier,
// untracked mechanism. That assumption held in every test fixture (which always
// pre-seeds accounts directly into the test tree) but not in production: the very
// first deposit or lab-credit into a genuinely fresh Sequencer database failed this
// exact check — confirmed live, batch #1 on Sepolia, TakerBase inclusion check.
// IsGenesis must be boolean; when 1, Balance and Nonce are constrained to 0 (a
// genesis leg cannot claim a fabricated prior balance) and the old-leaf hash used
// for the inclusion check is 0 instead of accountLeaf(...). A prover cannot lie
// about IsGenesis for an already-funded leaf — the inclusion check would fail to
// reconcile against the true on-chain root either way, so this doesn't weaken the
// existing soundness, it just makes the one legitimate case (genuinely first touch)
// provable at all.
type StateUpdate struct {
	Index     frontend.Variable
	Balance   frontend.Variable
	Nonce     frontend.Variable
	IsGenesis frontend.Variable
	Path      [MerkleDepth]frontend.Variable
	PathBits  [MerkleDepth]frontend.Variable
}

// Operation represents a state-mutating action inside a batch.
type Operation struct {
	OpType frontend.Variable

	Amount      frontend.Variable 
	QuoteAmount frontend.Variable // Used only in Trade

	// Maker (Sender in Transfer/Withdraw)
	MakerPubKey   eddsa.PublicKey
	MakerSig      eddsa.Signature
	MakerBase     StateUpdate // Token being sold (or transferred)
	MakerQuote    StateUpdate // Token being bought (or zeroed for transfer)

	// Taker (Receiver in Transfer/Deposit)
	TakerPubKey   eddsa.PublicKey
	TakerSig      eddsa.Signature // Used only in Trade
	TakerBase     StateUpdate // Token being bought (or received)
	TakerQuote    StateUpdate // Token being sold
}

type StateTransitionCircuit struct {
	OldRoot        frontend.Variable `gnark:",public"`
	NewRoot        frontend.Variable `gnark:",public"`
	WithdrawalHash frontend.Variable `gnark:",public"`
	DepositHash    frontend.Variable `gnark:",public"`

	Ops [BatchSize]Operation
}

// accountLeaf computes the MiMC hash of an account's state.
func accountLeaf(h *mimc.MiMC, index, pubX, pubY, balance, nonce frontend.Variable) frontend.Variable {
	h.Reset()
	h.Write(index, pubX, pubY, balance, nonce)
	return h.Sum()
}

// oldLeafHash computes the "old" leaf value to use in a Merkle inclusion check for
// one leg of an operation, accounting for genesis (see StateUpdate.IsGenesis's doc
// comment for why this can't just always be accountLeaf(...)). Also constrains
// Balance and Nonce to 0 whenever IsGenesis is claimed — a genesis leg cannot
// fabricate a prior balance, since the inclusion check no longer depends on those
// fields at all once genesis is selected.
func oldLeafHash(api frontend.API, h *mimc.MiMC, u StateUpdate, pubX, pubY frontend.Variable) frontend.Variable {
	api.AssertIsBoolean(u.IsGenesis)
	api.AssertIsEqual(api.Mul(u.IsGenesis, u.Balance), 0)
	api.AssertIsEqual(api.Mul(u.IsGenesis, u.Nonce), 0)
	real := accountLeaf(h, u.Index, pubX, pubY, u.Balance, u.Nonce)
	return api.Select(u.IsGenesis, 0, real)
}

func merkleRoot(h *mimc.MiMC, api frontend.API, leaf frontend.Variable, path, bits [MerkleDepth]frontend.Variable) frontend.Variable {
	cur := leaf
	for i := 0; i < MerkleDepth; i++ {
		left := api.Select(bits[i], path[i], cur)
		right := api.Select(bits[i], cur, path[i])
		h.Reset()
		h.Write(left, right)
		cur = h.Sum()
	}
	return cur
}

func rangeCheck(api frontend.API, val frontend.Variable) {
	api.ToBinary(val, 252)
}

// Dummy EdDSA fixture used to make signature checks conditional per OpType.
//
// gnark's eddsa.Verify performs hard, unconditional constraints — including an
// on-curve assertion on an internally-derived point — so it cannot be wrapped with
// api.Select after the fact the way the arithmetic/root checks below are (that would
// still force the real, possibly-invalid inputs through the on-curve assertion). The
// fix instead muxes the *inputs* to Verify between the real op data and this fixed,
// publicly known, always-valid signature: when a check is inactive for a given
// OpType, Verify checks this harmless fixture instead of failing on whatever
// real-but-irrelevant data happens to sit in that leg of the op.
//
// This is a manually constructed Schnorr-style signature (R = r*G, S = r + hRAM*sk,
// hRAM = MiMC(R, A, msg)) satisfying the exact verification equation eddsa.Verify
// checks, using the same MiMC construction used everywhere else in this circuit. It
// is not derived from any secret worth protecting — sk/r are arbitrary fixed
// scalars chosen only to produce this fixture. Independently verified against the
// native (non-circuit) equivalent before being hardcoded here.
const (
	dummySigAX  = "10298319502295528021229850954227631805200385207603455133709476095367371825263"
	dummySigAY  = "2504327762688954486123660800211611946128639234034449816647673413157039235394"
	dummySigRX  = "19615509352750535901637443802064826507464158183993156839337785904648430343214"
	dummySigRY  = "14649913852925992635033550000810266724942332423222322031248971935568823466398"
	dummySigS   = "1567003771490820873806856620396274719304459608209466000889389818739844298687"
	dummySigMsg = "1"
)

// conditionalVerify checks (pubKey, sig) against msg only when active == 1. When
// active == 0, it verifies the fixed dummy fixture above instead (which always
// succeeds), so an inactive leg's real-but-meaningless data can never trip the
// signature constraint. active must be a boolean (0 or 1) value, e.g. from
// api.IsZero — the caller is responsible for that, this does not re-check it.
func conditionalVerify(api frontend.API, curve gnark_twistededwards.Curve, h *mimc.MiMC, active frontend.Variable, pubKey eddsa.PublicKey, sig eddsa.Signature, msg frontend.Variable) error {
	effPubKey := eddsa.PublicKey{A: gnark_twistededwards.Point{
		X: api.Select(active, pubKey.A.X, dummySigAX),
		Y: api.Select(active, pubKey.A.Y, dummySigAY),
	}}
	effSig := eddsa.Signature{
		R: gnark_twistededwards.Point{
			X: api.Select(active, sig.R.X, dummySigRX),
			Y: api.Select(active, sig.R.Y, dummySigRY),
		},
		S: api.Select(active, sig.S, dummySigS),
	}
	effMsg := api.Select(active, msg, dummySigMsg)
	h.Reset()
	return eddsa.Verify(curve, effSig, effMsg, effPubKey, h)
}

func processOperation(api frontend.API, op *Operation, root frontend.Variable, currentWithdrawalHash frontend.Variable, currentDepositHash frontend.Variable, curve gnark_twistededwards.Curve, h *mimc.MiMC) (frontend.Variable, frontend.Variable, frontend.Variable, error) {
	rangeCheck(api, op.Amount)
	rangeCheck(api, op.QuoteAmount)

	isTrade := api.IsZero(api.Sub(op.OpType, OpTrade))
	isTransfer := api.IsZero(api.Sub(op.OpType, OpTransfer))
	isWithdrawal := api.IsZero(api.Sub(op.OpType, OpWithdrawal))
	isDeposit := api.IsZero(api.Sub(op.OpType, OpDeposit))

	// Ensure valid OpType
	validOp := api.Or(isTrade, isTransfer)
	validOp = api.Or(validOp, isWithdrawal)
	validOp = api.Or(validOp, isDeposit)
	api.AssertIsEqual(validOp, 1)

	// 1. Verify Maker Signature (Trade, Transfer, Withdrawal — NOT Deposit).
	// Message omits exact fill amounts so one auth covers partial fills of a resting order.
	// Deposits have no Maker (the L1 Deposit event is the authenticity source, not a
	// signature), so the check is skipped via conditionalVerify rather than forced
	// through on real-but-meaningless Maker data — see its doc comment for why a plain
	// api.Select wrap after the fact doesn't work for eddsa.Verify.
	h.Reset()
	h.Write(op.OpType, op.MakerPubKey.A.X, op.MakerPubKey.A.Y, op.MakerBase.Index, op.MakerQuote.Index)
	makerMsgHash := h.Sum()

	makerSigActive := api.Sub(1, isDeposit)
	if err := conditionalVerify(api, curve, h, makerSigActive, op.MakerPubKey, op.MakerSig, makerMsgHash); err != nil {
		return nil, nil, nil, err
	}

	// 2. Verify Taker Signature (Trade only — a trade needs both counterparties'
	// consent; Transfer/Withdrawal/Deposit only ever need the Maker's authorization,
	// there's no "receiver must sign to receive" requirement).
	h.Reset()
	h.Write(op.OpType, op.TakerPubKey.A.X, op.TakerPubKey.A.Y, op.TakerBase.Index, op.TakerQuote.Index)
	takerMsgHash := h.Sum()

	if err := conditionalVerify(api, curve, h, isTrade, op.TakerPubKey, op.TakerSig, takerMsgHash); err != nil {
		return nil, nil, nil, err
	}

	// --- 3. Withdrawal Accumulator ---
	h.Reset()
	h.Write(currentWithdrawalHash, op.MakerBase.Index, op.Amount)
	nextWithdrawalHash := h.Sum()
	currentWithdrawalHash = api.Select(isWithdrawal, nextWithdrawalHash, currentWithdrawalHash)

	// --- 4. Deposit Accumulator ---
	h.Reset()
	h.Write(currentDepositHash, op.TakerBase.Index, op.Amount)
	nextDepositHash := h.Sum()
	currentDepositHash = api.Select(isDeposit, nextDepositHash, currentDepositHash)

	// --- 5. State Updates ---

	// A. Maker Base (Debit Amount)
	// Active for: Trade, Transfer, Withdrawal
	makerBaseActive := api.Sub(1, isDeposit)
	
	oldMakerBaseLeaf := oldLeafHash(api, h, op.MakerBase, op.MakerPubKey.A.X, op.MakerPubKey.A.Y)
	root1 := merkleRoot(h, api, oldMakerBaseLeaf, op.MakerBase.Path, op.MakerBase.PathBits)
	// Assert root only if active
	api.AssertIsEqual(api.Select(makerBaseActive, root1, root), root)

	api.AssertIsLessOrEqual(api.Select(makerBaseActive, op.Amount, 0), op.MakerBase.Balance)
	newMakerBaseBal := api.Sub(op.MakerBase.Balance, api.Select(makerBaseActive, op.Amount, 0))
	// Trades do not bump nonce (allows partial fills under one circuit auth). Transfers/withdrawals do.
	newMakerBaseNonce := api.Add(op.MakerBase.Nonce, api.Sub(makerBaseActive, api.Mul(makerBaseActive, isTrade)))
	
	newMakerBaseLeaf := accountLeaf(h, op.MakerBase.Index, op.MakerPubKey.A.X, op.MakerPubKey.A.Y, newMakerBaseBal, newMakerBaseNonce)
	root = api.Select(makerBaseActive, merkleRoot(h, api, newMakerBaseLeaf, op.MakerBase.Path, op.MakerBase.PathBits), root)

	// B. Maker Quote (Credit QuoteAmount)
	// Active for: Trade
	makerQuoteActive := isTrade
	oldMakerQuoteLeaf := oldLeafHash(api, h, op.MakerQuote, op.MakerPubKey.A.X, op.MakerPubKey.A.Y)
	root2 := merkleRoot(h, api, oldMakerQuoteLeaf, op.MakerQuote.Path, op.MakerQuote.PathBits)
	api.AssertIsEqual(api.Select(makerQuoteActive, root2, root), root)

	newMakerQuoteBal := api.Add(op.MakerQuote.Balance, api.Select(makerQuoteActive, op.QuoteAmount, 0))
	newMakerQuoteLeaf := accountLeaf(h, op.MakerQuote.Index, op.MakerPubKey.A.X, op.MakerPubKey.A.Y, newMakerQuoteBal, op.MakerQuote.Nonce) // nonce unchanged
	root = api.Select(makerQuoteActive, merkleRoot(h, api, newMakerQuoteLeaf, op.MakerQuote.Path, op.MakerQuote.PathBits), root)

	// C. Taker Base (Credit Amount)
	// Active for: Trade, Transfer, Deposit
	takerBaseActive := api.Sub(1, isWithdrawal)
	oldTakerBaseLeaf := oldLeafHash(api, h, op.TakerBase, op.TakerPubKey.A.X, op.TakerPubKey.A.Y)
	root3 := merkleRoot(h, api, oldTakerBaseLeaf, op.TakerBase.Path, op.TakerBase.PathBits)
	api.AssertIsEqual(api.Select(takerBaseActive, root3, root), root)

	newTakerBaseBal := api.Add(op.TakerBase.Balance, api.Select(takerBaseActive, op.Amount, 0))
	newTakerBaseLeaf := accountLeaf(h, op.TakerBase.Index, op.TakerPubKey.A.X, op.TakerPubKey.A.Y, newTakerBaseBal, op.TakerBase.Nonce)
	root = api.Select(takerBaseActive, merkleRoot(h, api, newTakerBaseLeaf, op.TakerBase.Path, op.TakerBase.PathBits), root)

	// D. Taker Quote (Debit QuoteAmount)
	// Active for: Trade
	takerQuoteActive := isTrade
	oldTakerQuoteLeaf := oldLeafHash(api, h, op.TakerQuote, op.TakerPubKey.A.X, op.TakerPubKey.A.Y)
	root4 := merkleRoot(h, api, oldTakerQuoteLeaf, op.TakerQuote.Path, op.TakerQuote.PathBits)
	api.AssertIsEqual(api.Select(takerQuoteActive, root4, root), root)

	api.AssertIsLessOrEqual(api.Select(takerQuoteActive, op.QuoteAmount, 0), op.TakerQuote.Balance)
	newTakerQuoteBal := api.Sub(op.TakerQuote.Balance, api.Select(takerQuoteActive, op.QuoteAmount, 0))
	// Trades: nonce unchanged so resting leftovers keep a valid circuit auth across partial fills.
	newTakerQuoteNonce := op.TakerQuote.Nonce

	newTakerQuoteLeaf := accountLeaf(h, op.TakerQuote.Index, op.TakerPubKey.A.X, op.TakerPubKey.A.Y, newTakerQuoteBal, newTakerQuoteNonce)
	root = api.Select(takerQuoteActive, merkleRoot(h, api, newTakerQuoteLeaf, op.TakerQuote.Path, op.TakerQuote.PathBits), root)

	return root, currentWithdrawalHash, currentDepositHash, nil
}

func (c *StateTransitionCircuit) Define(api frontend.API) error {
	curve, err := gnark_twistededwards.NewEdCurve(api, twistededwards.BN254)
	if err != nil { return err }

	h, err := mimc.NewMiMC(api)
	if err != nil { return err }

	root := c.OldRoot
	currentWithdrawalHash := frontend.Variable(0)
	currentDepositHash := frontend.Variable(0)

	for i := 0; i < BatchSize; i++ {
		op := &c.Ops[i]
		root, currentWithdrawalHash, currentDepositHash, err = processOperation(api, op, root, currentWithdrawalHash, currentDepositHash, curve, &h)
		if err != nil { return err }
	}

	api.AssertIsEqual(root, c.NewRoot)
	api.AssertIsEqual(currentWithdrawalHash, c.WithdrawalHash)
	api.AssertIsEqual(currentDepositHash, c.DepositHash)

	return nil
}
