// Package xdr extracts deposit routing information directly from raw,
// base64-encoded Stellar transaction envelopes, as delivered by Stellar Core
// and Horizon (the envelope_xdr field).
//
// It decodes only the parts of the envelope that routing needs with its own
// XDR stream reader, so it adds no dependency beyond this module.
package xdr

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/REDISHFISH/BLUEWHALE/packages/core-go/address"
	"github.com/REDISHFISH/BLUEWHALE/packages/core-go/routing"
)

var (
	// ErrInvalidXDR reports an envelope that is not valid base64 or not a
	// well-formed XDR TransactionEnvelope.
	ErrInvalidXDR = errors.New("xdr: invalid transaction envelope")

	// ErrUnsupportedEnvelope reports an envelope type other than a v0, v1 or
	// fee-bump transaction.
	ErrUnsupportedEnvelope = errors.New("xdr: unsupported envelope type")

	// ErrUnsupportedOperation reports an operation this package cannot
	// decode. Operations are not length-prefixed in XDR, so an operation that
	// cannot be decoded also hides every operation after it.
	ErrUnsupportedOperation = errors.New("xdr: unsupported operation type")
)

// XDR discriminants, from Stellar-transaction.x and Stellar-types.x.
const (
	envelopeTypeTxV0     = 0
	envelopeTypeTx       = 2
	envelopeTypeFeeBump  = 5
	keyTypeEd25519       = 0
	keyTypeMuxedEd25519  = 0x100
	publicKeyTypeEd25519 = 0

	memoNone   = 0
	memoText   = 1
	memoID     = 2
	memoHash   = 3
	memoReturn = 4

	precondNone = 0
	precondTime = 1
	precondV2   = 2

	signerKeyEd25519       = 0
	signerKeyPreAuthTx     = 1
	signerKeyHashX         = 2
	signerKeySignedPayload = 3

	assetTypeNative    = 0
	assetTypeAlphanum4 = 1
	assetTypeAlpha12   = 2
	assetTypePoolShare = 3

	opCreateAccount                 = 0
	opPayment                       = 1
	opPathPaymentStrictReceive      = 2
	opChangeTrust                   = 6
	opAccountMerge                  = 8
	opInflation                     = 9
	opManageData                    = 10
	opBumpSequence                  = 11
	opPathPaymentStrictSend         = 13
	opBeginSponsoringFutureReserves = 16
	opEndSponsoringFutureReserves   = 17

	maxOperations   = 100
	maxSignatures   = 20
	maxMemoText     = 28
	maxPathLength   = 5
	maxExtraSigners = 2
)

// ExtractRoutingFromXDR decodes a base64 TransactionEnvelope (v0, v1 or fee
// bump) and returns one RoutingResult per payment-like operation, in
// operation order: create account, payment, and both path payments. Muxed
// (M) destinations are supported and take precedence over the memo exactly as
// in routing.ExtractRouting, which produces each result.
//
// The source account passed to routing is the operation's source when it has
// one and the transaction's source otherwise. For a fee bump the inner
// transaction is used; the fee source does not affect routing.
//
// Other operation types carry no deposit and are skipped. Those this package
// cannot decode return ErrUnsupportedOperation. Malformed input returns
// ErrInvalidXDR. Both are wrapped; test with errors.Is.
func ExtractRoutingFromXDR(rawEnvelope string) ([]routing.RoutingResult, error) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(rawEnvelope))
	if err != nil {
		return nil, fmt.Errorf("%w: not valid base64", ErrInvalidXDR)
	}

	r := &reader{buf: raw}
	tx, err := decodeEnvelope(r)
	if err != nil {
		return nil, err
	}
	if err := r.done(); err != nil {
		return nil, err
	}

	results := make([]routing.RoutingResult, 0, len(tx.payments))
	for _, p := range tx.payments {
		source := tx.source
		if p.source != "" {
			source = p.source
		}
		results = append(results, routing.ExtractRouting(routing.RoutingInput{
			Destination:   p.destination,
			SourceAccount: source,
			MemoType:      tx.memoType,
			MemoValue:     tx.memoValue,
		}))
	}
	return results, nil
}

type transaction struct {
	source    string
	memoType  string
	memoValue string
	payments  []payment
}

type payment struct {
	destination string
	source      string
}

func decodeEnvelope(r *reader) (transaction, error) {
	envelopeType := r.int32()
	if r.err != nil {
		return transaction{}, r.err
	}

	switch envelopeType {
	case envelopeTypeTxV0:
		// TransactionV0 stores the source as a bare Ed25519 key.
		source := encodeAccount(r.fixed(32))
		tx, err := decodeTransactionBody(r, source, false)
		r.skipSignatures()
		return tx, firstErr(err, r.err)

	case envelopeTypeTx:
		return decodeV1Envelope(r)

	case envelopeTypeFeeBump:
		r.muxedAccount() // fee source
		r.uint64()       // fee
		innerType := r.int32()
		if r.err != nil {
			return transaction{}, r.err
		}
		if innerType != envelopeTypeTx {
			return transaction{}, fmt.Errorf("%w: fee bump inner type %d", ErrUnsupportedEnvelope, innerType)
		}
		tx, err := decodeV1Envelope(r)
		if err != nil {
			return transaction{}, err
		}
		r.extensionPoint()
		r.skipSignatures()
		return tx, r.err

	default:
		return transaction{}, fmt.Errorf("%w: %d", ErrUnsupportedEnvelope, envelopeType)
	}
}

func decodeV1Envelope(r *reader) (transaction, error) {
	source := r.muxedAccount()
	tx, err := decodeTransactionBody(r, source, true)
	r.skipSignatures()
	return tx, firstErr(err, r.err)
}

// decodeTransactionBody decodes the fields after the source account. v1
// transactions carry Preconditions; v0 carries an optional TimeBounds.
func decodeTransactionBody(r *reader, source string, v1 bool) (transaction, error) {
	tx := transaction{source: source}

	r.uint32() // fee
	r.uint64() // sequence number
	if v1 {
		r.preconditions()
	} else if r.bool() {
		r.take(16) // TimeBounds
	}

	tx.memoType, tx.memoValue = r.memo()

	n := r.count(maxOperations)
	for i := 0; i < n && r.err == nil; i++ {
		p, routable, err := r.operation()
		if err != nil {
			return transaction{}, fmt.Errorf("operation %d: %w", i, err)
		}
		if routable {
			tx.payments = append(tx.payments, p)
		}
	}

	r.extensionPoint()
	return tx, r.err
}

// operation decodes one Operation. routable reports whether it moves funds to
// a destination.
func (r *reader) operation() (p payment, routable bool, err error) {
	if r.bool() {
		p.source = r.muxedAccount()
	}

	opType := r.int32()
	if r.err != nil {
		return payment{}, false, r.err
	}

	switch opType {
	case opCreateAccount:
		p.destination = r.accountID()
		r.uint64() // starting balance
		routable = true
	case opPayment:
		p.destination = r.muxedAccount()
		r.asset()
		r.uint64() // amount
		routable = true
	case opPathPaymentStrictReceive, opPathPaymentStrictSend:
		// Both are: asset, int64, destination, asset, int64, path.
		r.asset()
		r.uint64()
		p.destination = r.muxedAccount()
		r.asset()
		r.uint64()
		for i, n := 0, r.count(maxPathLength); i < n; i++ {
			r.asset()
		}
		routable = true
	case opChangeTrust:
		r.changeTrustAsset()
		r.uint64() // limit
	case opAccountMerge:
		r.muxedAccount()
	case opInflation, opEndSponsoringFutureReserves:
	case opManageData:
		r.variable(64) // data name
		if r.bool() {
			r.variable(64) // data value
		}
	case opBumpSequence:
		r.uint64()
	case opBeginSponsoringFutureReserves:
		r.accountID()
	default:
		return payment{}, false, fmt.Errorf("%w: %d", ErrUnsupportedOperation, opType)
	}

	return p, routable, r.err
}

// muxedAccount decodes a MuxedAccount into a G- or M-address.
func (r *reader) muxedAccount() string {
	switch keyType := r.int32(); keyType {
	case keyTypeEd25519:
		return encodeAccount(r.fixed(32))
	case keyTypeMuxedEd25519:
		id := r.uint64()
		key := r.fixed(32)
		if r.err != nil {
			return ""
		}
		var payload [40]byte
		copy(payload[:32], key)
		binary.BigEndian.PutUint64(payload[32:], id)
		m, _ := address.EncodeStrKey(address.VersionByteMuxedAccount, payload[:])
		return m
	default:
		r.fail("unknown muxed account type %d", keyType)
		return ""
	}
}

// accountID decodes an AccountID (PublicKey) into a G-address.
func (r *reader) accountID() string {
	if keyType := r.int32(); keyType != publicKeyTypeEd25519 {
		r.fail("unknown public key type %d", keyType)
		return ""
	}
	return encodeAccount(r.fixed(32))
}

func encodeAccount(key []byte) string {
	if len(key) != 32 {
		return ""
	}
	g, _ := address.EncodeStrKey(address.VersionByteAccountID, key)
	return g
}

// memo decodes a Memo into the memo type and value strings routing expects.
// An ID is rendered in decimal; a hash or return hash in lowercase hex.
func (r *reader) memo() (memoType, memoValue string) {
	switch t := r.int32(); t {
	case memoNone:
		return "none", ""
	case memoText:
		return "text", string(r.variable(maxMemoText))
	case memoID:
		return "id", strconv.FormatUint(r.uint64(), 10)
	case memoHash:
		return "hash", hex.EncodeToString(r.fixed(32))
	case memoReturn:
		return "return", hex.EncodeToString(r.fixed(32))
	default:
		r.fail("unknown memo type %d", t)
		return "", ""
	}
}

func (r *reader) preconditions() {
	switch t := r.int32(); t {
	case precondNone:
	case precondTime:
		r.take(16)
	case precondV2:
		if r.bool() {
			r.take(16) // time bounds
		}
		if r.bool() {
			r.take(8) // ledger bounds
		}
		if r.bool() {
			r.take(8) // min sequence number
		}
		r.uint64() // min sequence age
		r.uint32() // min sequence ledger gap
		for i, n := 0, r.count(maxExtraSigners); i < n; i++ {
			r.signerKey()
		}
	default:
		r.fail("unknown precondition type %d", t)
	}
}

func (r *reader) signerKey() {
	switch t := r.int32(); t {
	case signerKeyEd25519, signerKeyPreAuthTx, signerKeyHashX:
		r.take(32)
	case signerKeySignedPayload:
		r.take(32)
		r.variable(64)
	default:
		r.fail("unknown signer key type %d", t)
	}
}

func (r *reader) asset() {
	r.assetBody(r.int32())
}

func (r *reader) assetBody(assetType int32) {
	switch assetType {
	case assetTypeNative:
	case assetTypeAlphanum4:
		r.take(4)
		r.accountID()
	case assetTypeAlpha12:
		r.take(12)
		r.accountID()
	default:
		r.fail("unknown asset type %d", assetType)
	}
}

// changeTrustAsset is an Asset that may also be a liquidity pool share.
func (r *reader) changeTrustAsset() {
	assetType := r.int32()
	if assetType != assetTypePoolShare {
		r.assetBody(assetType)
		return
	}
	if poolType := r.int32(); poolType != 0 {
		r.fail("unknown liquidity pool type %d", poolType)
		return
	}
	r.asset()
	r.asset()
	r.int32() // fee
}

// extensionPoint reads an "ext" union, which must be the empty v0 arm.
func (r *reader) extensionPoint() {
	if v := r.int32(); v != 0 {
		r.fail("unsupported extension version %d", v)
	}
}

// skipSignatures reads the DecoratedSignature list that ends every envelope.
func (r *reader) skipSignatures() {
	for i, n := 0, r.count(maxSignatures); i < n && r.err == nil; i++ {
		r.take(4)      // signature hint
		r.variable(64) // signature
	}
}

func firstErr(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}
