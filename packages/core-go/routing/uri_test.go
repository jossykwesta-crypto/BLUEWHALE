package routing

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestExtractRoutingFromURI_ValidPay(t *testing.T) {
	uri := "web+stellar:pay?destination=GAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQADRSI&memo=12345&memo_type=MEMO_ID"
	result, params, err := ExtractRoutingFromURI(uri)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if params.Destination != "GAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQADRSI" {
		t.Errorf("expected destination, got %s", params.Destination)
	}
	if params.Memo != "12345" {
		t.Errorf("expected memo 12345, got %s", params.Memo)
	}
	if result.DestinationBaseAccount != "GAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQADRSI" {
		t.Errorf("expected base account, got %s", result.DestinationBaseAccount)
	}
	if result.RoutingID == nil || result.RoutingID.String() != "12345" {
		t.Errorf("expected routing ID 12345, got %v", result.RoutingID)
	}
}

func TestExtractRoutingFromURI_InvalidScheme(t *testing.T) {
	uri := "https://example.com/pay?destination=GAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQADRSI"
	_, _, err := ExtractRoutingFromURI(uri)
	if err != ErrInvalidScheme {
		t.Errorf("expected ErrInvalidScheme, got %v", err)
	}
}

func TestExtractRoutingFromURI_UnsupportedOperation(t *testing.T) {
	uri := "web+stellar:tx?xdr=AAAA"
	_, _, err := ExtractRoutingFromURI(uri)
	if err != ErrUnsupportedOperation {
		t.Errorf("expected ErrUnsupportedOperation, got %v", err)
	}
}

func TestExtractRoutingFromURI_MissingDestination(t *testing.T) {
	uri := "web+stellar:pay?amount=100"
	_, _, err := ExtractRoutingFromURI(uri)
	if err != ErrMissingDestination {
		t.Errorf("expected ErrMissingDestination, got %v", err)
	}
}

// uriFuzzSeedEdgeCases are malformed and boundary SEP-0007 URIs added to the
// fuzz corpus alongside the URIs from spec/vectors.json.
var uriFuzzSeedEdgeCases = []string{
	"",
	"web+stellar:",
	"web+stellar:pay",
	"web+stellar:pay?",
	"web+stellar:pay?destination=",
	"web+stellar:pay?destination=%20%20",
	"web+stellar:pay?destination=%zz",
	"web+stellar:pay?destination=GAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQADRSI&memo=%",
	"web+stellar:pay?destination=GAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQADRSI;memo=1",
	"web+stellar:pay?destination=MAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQACABAAAAAAAAAAEVIG&memo=1&memo_type=MEMO_ID",
	"web+stellar:pay?destination=GAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQADRSI&memo=18446744073709551616&memo_type=MEMO_ID",
	"web+stellar:pay?destination=GAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQADRSI&memo=AAAA&memo_type=MEMO_HASH",
	"web+stellar:pay?destination=a&destination=b&memo_type=memo_text&memo=" + strings.Repeat("x", 64),
	"web+stellar:pay??destination=GAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQADRSI",
	"web+stellar:pay#destination=GAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQADRSI",
	"web+stellar:PAY?destination=GAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQADRSI",
	"WEB+STELLAR:pay?destination=GAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQADRSI",
	"web+stellar:tx?xdr=AAAA",
	"web+stellar:pay?destination=\x00\xff\xfe",
	"web+stellar:pay?" + strings.Repeat("&", 4096),
	"web+stellar:pay?destination=" + strings.Repeat("%41", 2048),
}

// uriFuzzSeedsFromVectors returns every SEP-0007 URI in spec/vectors.json.
func uriFuzzSeedsFromVectors(f *testing.F) []string {
	f.Helper()

	data, err := os.ReadFile("../../../spec/vectors.json")
	if err != nil {
		f.Fatalf("failed to read spec/vectors.json: %v", err)
	}

	var vectors struct {
		Cases []struct {
			Input map[string]any `json:"input"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(data, &vectors); err != nil {
		f.Fatalf("failed to decode spec/vectors.json: %v", err)
	}

	var seeds []string
	for _, c := range vectors.Cases {
		if s, ok := c.Input["uri"].(string); ok {
			seeds = append(seeds, s)
		}
	}
	if len(seeds) == 0 {
		f.Fatal("spec/vectors.json has no extract_from_uri inputs")
	}
	return seeds
}

// FuzzExtractRoutingFromURI checks that arbitrary URI strings never panic and
// that every result is consistent and deterministic:
//
//	go test ./routing -run '^$' -fuzz FuzzExtractRoutingFromURI -fuzztime 30s
func FuzzExtractRoutingFromURI(f *testing.F) {
	for _, s := range uriFuzzSeedsFromVectors(f) {
		f.Add(s)
	}
	for _, s := range uriFuzzSeedEdgeCases {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, uri string) {
		result, params, err := ExtractRoutingFromURI(uri)

		again, againParams, againErr := ExtractRoutingFromURI(uri)
		if !reflect.DeepEqual(result, again) || params != againParams || !sameError(err, againErr) {
			t.Fatalf("ExtractRoutingFromURI(%q) is not deterministic", uri)
		}

		if err != nil {
			if result.Success {
				t.Fatalf("ExtractRoutingFromURI(%q) reported success with error %v", uri, err)
			}
			if result.ErrorMessage != err.Error() {
				t.Fatalf("ExtractRoutingFromURI(%q) ErrorMessage = %q, want %q", uri, result.ErrorMessage, err.Error())
			}
			if params != (URIParams{}) {
				t.Fatalf("ExtractRoutingFromURI(%q) returned params %+v with error %v", uri, params, err)
			}
			return
		}

		if !strings.HasPrefix(uri, "web+stellar:pay") {
			t.Fatalf("ExtractRoutingFromURI(%q) accepted a URI without the web+stellar:pay prefix", uri)
		}
		if params.Destination == "" || params.Destination != strings.TrimSpace(params.Destination) {
			t.Fatalf("ExtractRoutingFromURI(%q) destination %q is empty or untrimmed", uri, params.Destination)
		}
	})
}

func sameError(a, b error) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Error() == b.Error()
}
