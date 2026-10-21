package xdr

import (
	"encoding/base64"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/REDISHFISH/BLUEWHALE/packages/core-go/address"
	"github.com/REDISHFISH/BLUEWHALE/packages/core-go/routing"
)

const (
	testG = "GAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQADRSI"
	// testM is testG muxed with ID 2^53+1.
	testM  = "MAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQACABAAAAAAAAAAEVIG"
	testID = uint64(9007199254740993)
)

// xdrWriter builds envelopes for the tests, mirroring reader.
type xdrWriter struct{ b []byte }

func (w *xdrWriter) u32(v uint32) *xdrWriter {
	w.b = binary.BigEndian.AppendUint32(w.b, v)
	return w
}

func (w *xdrWriter) u64(v uint64) *xdrWriter {
	w.b = binary.BigEndian.AppendUint64(w.b, v)
	return w
}

func (w *xdrWriter) raw(p []byte) *xdrWriter {
	w.b = append(w.b, p...)
	for len(w.b)%4 != 0 {
		w.b = append(w.b, 0)
	}
	return w
}

func (w *xdrWriter) opaque(p []byte) *xdrWriter {
	return w.u32(uint32(len(p))).raw(p)
}

func (w *xdrWriter) account(key []byte) *xdrWriter {
	return w.u32(keyTypeEd25519).raw(key)
}

func (w *xdrWriter) muxed(key []byte, id uint64) *xdrWriter {
	return w.u32(keyTypeMuxedEd25519).u64(id).raw(key)
}

func (w *xdrWriter) native() *xdrWriter { return w.u32(assetTypeNative) }

func (w *xdrWriter) base64() string { return base64.StdEncoding.EncodeToString(w.b) }

func testKey(t *testing.T) []byte {
	t.Helper()
	_, key, err := address.DecodeStrKey(testG)
	if err != nil {
		t.Fatalf("DecodeStrKey: %v", err)
	}
	return key
}

// v1 writes an ENVELOPE_TYPE_TX header (source, fee, sequence, no
// preconditions), then memo and ops, then the ext and one signature.
func v1(key []byte, memo func(*xdrWriter), ops ...func(*xdrWriter)) *xdrWriter {
	w := &xdrWriter{}
	w.u32(envelopeTypeTx)
	writeTxV1(w, key, memo, ops...)
	writeSignatures(w)
	return w
}

func writeTxV1(w *xdrWriter, key []byte, memo func(*xdrWriter), ops ...func(*xdrWriter)) {
	w.account(key).u32(100).u64(1).u32(precondNone)
	memo(w)
	w.u32(uint32(len(ops)))
	for _, op := range ops {
		op(w)
	}
	w.u32(0) // ext
}

func writeSignatures(w *xdrWriter) {
	w.u32(1).raw([]byte{1, 2, 3, 4}).opaque(make([]byte, 64))
}

func noMemo(w *xdrWriter) { w.u32(memoNone) }

func idMemo(id uint64) func(*xdrWriter) {
	return func(w *xdrWriter) { w.u32(memoID).u64(id) }
}

func paymentTo(dest func(*xdrWriter)) func(*xdrWriter) {
	return func(w *xdrWriter) {
		w.u32(0) // no op source
		w.u32(opPayment)
		dest(w)
		w.native().u64(1_000_000)
	}
}

func extract(t *testing.T, env string) []routing.RoutingResult {
	t.Helper()
	results, err := ExtractRoutingFromXDR(env)
	if err != nil {
		t.Fatalf("ExtractRoutingFromXDR: %v", err)
	}
	return results
}

func TestExtractPaymentWithMemoID(t *testing.T) {
	key := testKey(t)
	env := v1(key, idMemo(42), paymentTo(func(w *xdrWriter) { w.account(key) })).base64()

	results := extract(t, env)
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	r := results[0]
	if r.DestinationBaseAccount != testG || r.RoutingSource != "memo" || r.RoutingID.String() != "42" {
		t.Fatalf("unexpected result: %+v (id %s)", r, r.RoutingID.String())
	}
}

func TestExtractMuxedPayment(t *testing.T) {
	key := testKey(t)
	env := v1(key, noMemo, paymentTo(func(w *xdrWriter) { w.muxed(key, testID) })).base64()

	r := extract(t, env)[0]
	if r.DestinationBaseAccount != testG || r.RoutingSource != "muxed" || r.RoutingID.String() != "9007199254740993" {
		t.Fatalf("unexpected result: %+v (id %s)", r, r.RoutingID.String())
	}

	// The decoded destination must be the canonical M-address.
	want := routing.ExtractRouting(routing.RoutingInput{Destination: testM, SourceAccount: testG, MemoType: "none"})
	if r.RoutingID.String() != want.RoutingID.String() || r.DestinationBaseAccount != want.DestinationBaseAccount {
		t.Fatalf("XDR result %+v differs from ExtractRouting(%s) %+v", r, testM, want)
	}
}

func TestExtractCreateAccountAndPathPayments(t *testing.T) {
	key := testKey(t)
	createAccount := func(w *xdrWriter) {
		w.u32(0).u32(opCreateAccount).u32(publicKeyTypeEd25519).raw(key).u64(10_000_000)
	}
	pathPayment := func(opType uint32) func(*xdrWriter) {
		return func(w *xdrWriter) {
			w.u32(0).u32(opType)
			w.native().u64(5)
			w.muxed(key, 7)
			w.u32(assetTypeAlphanum4).raw([]byte("USDC")).account(key)
			w.u64(5)
			w.u32(1).native() // path of one asset
		}
	}
	manageData := func(w *xdrWriter) {
		w.u32(0).u32(opManageData).opaque([]byte("name")).u32(1).opaque([]byte("value"))
	}

	env := v1(key, idMemo(9), createAccount, manageData, pathPayment(opPathPaymentStrictReceive), pathPayment(opPathPaymentStrictSend)).base64()
	results := extract(t, env)
	if len(results) != 3 {
		t.Fatalf("got %d results, want 3 (manage data is skipped)", len(results))
	}
	if results[0].RoutingSource != "memo" || results[0].RoutingID.String() != "9" {
		t.Fatalf("create account: %+v", results[0])
	}
	for _, r := range results[1:] {
		if r.RoutingSource != "muxed" || r.RoutingID.String() != "7" {
			t.Fatalf("path payment: %+v", r)
		}
	}
}

func TestExtractV0AndFeeBumpEnvelopes(t *testing.T) {
	key := testKey(t)
	pay := paymentTo(func(w *xdrWriter) { w.account(key) })

	v0 := &xdrWriter{}
	v0.u32(envelopeTypeTxV0).raw(key).u32(100).u64(1)
	v0.u32(1).u64(0).u64(0) // time bounds present
	idMemo(5)(v0)
	v0.u32(1)
	pay(v0)
	v0.u32(0)
	writeSignatures(v0)

	feeBump := &xdrWriter{}
	feeBump.u32(envelopeTypeFeeBump).muxed(key, 1).u64(200).u32(envelopeTypeTx)
	writeTxV1(feeBump, key, idMemo(5), pay)
	writeSignatures(feeBump) // inner
	feeBump.u32(0)           // fee bump ext
	writeSignatures(feeBump) // outer

	for name, env := range map[string]string{"v0": v0.base64(), "fee bump": feeBump.base64()} {
		results := extract(t, env)
		if len(results) != 1 || results[0].RoutingID.String() != "5" || results[0].DestinationBaseAccount != testG {
			t.Fatalf("%s: unexpected results %+v", name, results)
		}
	}
}

func TestExtractOperationWithSourceAccount(t *testing.T) {
	key := testKey(t)
	op := func(w *xdrWriter) {
		w.u32(1).muxed(key, 11) // op source present
		w.u32(opPayment).account(key).native().u64(1)
	}
	r := extract(t, v1(key, idMemo(3), op).base64())[0]
	if r.RoutingSource != "memo" || r.RoutingID.String() != "3" || len(r.Warnings) != 0 {
		t.Fatalf("unexpected result %+v", r)
	}
}

func TestExtractNoPayments(t *testing.T) {
	key := testKey(t)
	bump := func(w *xdrWriter) { w.u32(0).u32(opBumpSequence).u64(99) }
	results := extract(t, v1(key, noMemo, bump).base64())
	if results == nil || len(results) != 0 {
		t.Fatalf("got %v, want an empty, non-nil slice", results)
	}
}

func TestExtractErrors(t *testing.T) {
	key := testKey(t)
	valid := v1(key, idMemo(1), paymentTo(func(w *xdrWriter) { w.account(key) }))

	unsupportedOp := v1(key, noMemo, func(w *xdrWriter) { w.u32(0).u32(24) }) // invoke host function

	badMemo := v1(key, func(w *xdrWriter) { w.u32(memoText).opaque(make([]byte, 29)) })

	badPadding := append([]byte(nil), v1(key, func(w *xdrWriter) { w.u32(memoText).opaque([]byte("a")) }).b...)
	// The memo text's three padding bytes follow: type(4) key(36) fee(4)
	// seq(8) cond(4) memoType(4) len(4) "a".
	badPadding[4+36+4+8+4+4+4+1] = 1

	tests := []struct {
		name string
		env  string
		want error
	}{
		{"not base64", "%%%", ErrInvalidXDR},
		{"empty", "", ErrInvalidXDR},
		{"truncated", base64.StdEncoding.EncodeToString(valid.b[:len(valid.b)-3]), ErrInvalidXDR},
		{"trailing bytes", base64.StdEncoding.EncodeToString(append(append([]byte(nil), valid.b...), 0, 0, 0, 0)), ErrInvalidXDR},
		{"memo text too long", badMemo.base64(), ErrInvalidXDR},
		{"non-zero padding", base64.StdEncoding.EncodeToString(badPadding), ErrInvalidXDR},
		{"unknown envelope", (&xdrWriter{}).u32(7).base64(), ErrUnsupportedEnvelope},
		{"unsupported operation", unsupportedOp.base64(), ErrUnsupportedOperation},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := ExtractRoutingFromXDR(tt.env)
			if !errors.Is(err, tt.want) {
				t.Fatalf("err = %v, want %v", err, tt.want)
			}
			if results != nil {
				t.Fatalf("results = %v, want nil on error", results)
			}
		})
	}
}

func TestExtractAcceptsSurroundingWhitespace(t *testing.T) {
	key := testKey(t)
	env := "\n  " + v1(key, idMemo(1), paymentTo(func(w *xdrWriter) { w.account(key) })).base64() + "\n"
	if len(extract(t, env)) != 1 {
		t.Fatal("expected one result")
	}
}
