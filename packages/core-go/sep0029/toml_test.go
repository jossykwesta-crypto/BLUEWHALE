package sep0029

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

const (
	acctA = "GAYCUYT553C5LHVE2XPW5GMEJT4BXGM7AHMJWLAPZP53KJO7EIQADRSI"
	acctB = "GBRPYHIL2CI3FNQ4BXLFMNDLFJUNPU2HY3ZMFSHONUCEOASW7QC7OX2H"
)

func TestParseTOML(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		check   func(t *testing.T, got *StellarTOML)
		wantErr error
		errLine int
	}{
		{
			name: "full document",
			input: `NETWORK_PASSPHRASE = "Public Global Stellar Network ; September 2015"
FEDERATION_SERVER = "https://example.com/federation"
ACCOUNTS = ["` + acctA + `", "` + acctB + `"]
MEMO_REQUIRED_ACCOUNTS = ["` + acctA + `"]

[DOCUMENTATION]
ORG_NAME = "Example Exchange"
ORG_URL = "https://example.com"
`,
			check: func(t *testing.T, got *StellarTOML) {
				want := &StellarTOML{
					OrgName:              "Example Exchange",
					NetworkPassphrase:    "Public Global Stellar Network ; September 2015",
					FederationServer:     "https://example.com/federation",
					Accounts:             []string{acctA, acctB},
					MemoRequiredAccounts: []string{acctA},
				}
				got.Values = nil
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("got %+v, want %+v", got, want)
				}
			},
		},
		{
			name:  "missing ORG_NAME is empty, not an error",
			input: "ACCOUNTS = []\n[DOCUMENTATION]\nORG_URL = \"https://example.com\"\n",
			check: func(t *testing.T, got *StellarTOML) {
				if got.OrgName != "" {
					t.Fatalf("OrgName = %q, want empty", got.OrgName)
				}
			},
		},
		{
			name:  "missing memo keys means unknown",
			input: "[DOCUMENTATION]\nORG_NAME = \"x\"\n",
			check: func(t *testing.T, got *StellarTOML) {
				if got.MemoRequiredAccounts != nil {
					t.Fatalf("MemoRequiredAccounts = %v, want nil", got.MemoRequiredAccounts)
				}
				if required, known := got.RequiresMemo(acctA); required || known {
					t.Fatalf("RequiresMemo = (%v, %v), want (false, false)", required, known)
				}
			},
		},
		{
			name:  "empty memo list is known",
			input: "MEMO_REQUIRED_ACCOUNTS = []\n",
			check: func(t *testing.T, got *StellarTOML) {
				if required, known := got.RequiresMemo(acctA); required || !known {
					t.Fatalf("RequiresMemo = (%v, %v), want (false, true)", required, known)
				}
			},
		},
		{
			name:  "memo required lookup",
			input: "MEMO_REQUIRED_ACCOUNTS = [\"" + acctA + "\"]\n",
			check: func(t *testing.T, got *StellarTOML) {
				if required, known := got.RequiresMemo(acctA); !required || !known {
					t.Fatalf("RequiresMemo(listed) = (%v, %v), want (true, true)", required, known)
				}
				if required, _ := got.RequiresMemo(acctB); required {
					t.Fatal("RequiresMemo(unlisted) = true")
				}
			},
		},
		{
			name:  "comments everywhere",
			input: "# header\n[DOCUMENTATION] # table\nORG_NAME = \"a # not a comment\" # trailing\n  # indented\n",
			check: func(t *testing.T, got *StellarTOML) {
				if got.OrgName != "a # not a comment" {
					t.Fatalf("OrgName = %q", got.OrgName)
				}
			},
		},
		{
			name:  "UTF-8 BOM is ignored",
			input: "\xef\xbb\xbf[DOCUMENTATION]\nORG_NAME = \"Bom Org\"\n",
			check: func(t *testing.T, got *StellarTOML) {
				if got.OrgName != "Bom Org" {
					t.Fatalf("OrgName = %q", got.OrgName)
				}
			},
		},
		{
			name:  "CRLF line endings",
			input: "[DOCUMENTATION]\r\nORG_NAME = \"Windows\"\r\nORG_URL = 'https://example.com'\r\n",
			check: func(t *testing.T, got *StellarTOML) {
				if got.OrgName != "Windows" || got.Values["DOCUMENTATION.ORG_URL"] != "https://example.com" {
					t.Fatalf("got %+v", got)
				}
			},
		},
		{
			name: "multi-line strings",
			input: "[DOCUMENTATION]\nORG_NAME = \"\"\"\nLine one \\\n    continued\"\"\"\n" +
				"ORG_DESCRIPTION = '''\nraw \\n stays\nsecond line'''\n",
			check: func(t *testing.T, got *StellarTOML) {
				if got.OrgName != "Line one continued" {
					t.Fatalf("OrgName = %q", got.OrgName)
				}
				if d := got.Values["DOCUMENTATION.ORG_DESCRIPTION"]; d != "raw \\n stays\nsecond line" {
					t.Fatalf("ORG_DESCRIPTION = %q", d)
				}
			},
		},
		{
			name:  "multi-line array with trailing comma and comments",
			input: "ACCOUNTS = [\n  \"" + acctA + "\", # first\n  \"" + acctB + "\",\n]\n",
			check: func(t *testing.T, got *StellarTOML) {
				if !reflect.DeepEqual(got.Accounts, []string{acctA, acctB}) {
					t.Fatalf("Accounts = %v", got.Accounts)
				}
			},
		},
		{
			name:  "escapes and unicode",
			input: "[DOCUMENTATION]\nORG_NAME = \"Caf\\u00e9 \\\"Q\\\" \\U0001F680\"\n",
			check: func(t *testing.T, got *StellarTOML) {
				if got.OrgName != "Café \"Q\" 🚀" {
					t.Fatalf("OrgName = %q", got.OrgName)
				}
			},
		},
		{
			name: "array tables, scalars and inline tables",
			input: "VERSION = 2\n[[CURRENCIES]]\ncode = \"USDC\"\ndecimals = 7\nis_asset_anchored = true\n" +
				"[[CURRENCIES]]\ncode = \"EURT\"\nfee = 0.5\nlimits = { min = 1_000, max = 5 }\n",
			check: func(t *testing.T, got *StellarTOML) {
				want := map[string]any{
					"VERSION":                        int64(2),
					"CURRENCIES.0.code":              "USDC",
					"CURRENCIES.0.decimals":          int64(7),
					"CURRENCIES.0.is_asset_anchored": true,
					"CURRENCIES.1.code":              "EURT",
					"CURRENCIES.1.fee":               0.5,
					"CURRENCIES.1.limits":            map[string]any{"min": int64(1000), "max": int64(5)},
				}
				if !reflect.DeepEqual(got.Values, want) {
					t.Fatalf("Values = %#v", got.Values)
				}
			},
		},
		{
			name:  "empty document",
			input: "",
			check: func(t *testing.T, got *StellarTOML) {
				if got.OrgName != "" || got.Accounts != nil || len(got.Values) != 0 {
					t.Fatalf("got %+v", got)
				}
			},
		},
		{name: "missing equals", input: "[DOCUMENTATION]\nORG_NAME \"x\"\n", wantErr: ErrInvalidTOML, errLine: 2},
		{name: "unterminated string", input: "\n\nORG = \"open\n", wantErr: ErrInvalidTOML, errLine: 3},
		{name: "unterminated table header", input: "[DOCUMENTATION\nORG_NAME = \"x\"\n", wantErr: ErrInvalidTOML, errLine: 1},
		{name: "duplicate key", input: "A = 1\nB = 2\nA = 3\n", wantErr: ErrInvalidTOML, errLine: 3},
		{name: "duplicate table", input: "[T]\nA = 1\n[T]\nB = 2\n", wantErr: ErrInvalidTOML, errLine: 3},
		{name: "garbage after value", input: "A = \"x\" y\n", wantErr: ErrInvalidTOML, errLine: 1},
		{name: "invalid bare value", input: "A = maybe\n", wantErr: ErrInvalidTOML, errLine: 1},
		{name: "ORG_NAME of the wrong type", input: "[DOCUMENTATION]\nORG_NAME = 42\n", wantErr: ErrInvalidTOML, errLine: 2},
		{name: "memo list of the wrong type", input: "MEMO_REQUIRED_ACCOUNTS = [1, 2]\n", wantErr: ErrInvalidTOML, errLine: 1},
		{name: "invalid UTF-8", input: "A = \"\xff\xfe\"\n", wantErr: ErrInvalidTOML, errLine: 1},
		{name: "invalid escape", input: "A = \"\\q\"\n", wantErr: ErrInvalidTOML, errLine: 1},
		{name: "oversized file", input: "A = \"" + strings.Repeat("x", MaxTOMLSize) + "\"\n", wantErr: ErrTOMLTooLarge},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTOML([]byte(tt.input))
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("ParseTOML: %v", err)
				}
				tt.check(t, got)
				return
			}

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if got != nil {
				t.Fatalf("got %+v with error, want nil", got)
			}
			var syntaxErr *SyntaxError
			if errors.As(err, &syntaxErr) && syntaxErr.Line != tt.errLine {
				t.Fatalf("error line = %d, want %d (%v)", syntaxErr.Line, tt.errLine, err)
			}
			if tt.wantErr == ErrInvalidTOML && syntaxErr == nil {
				t.Fatalf("err = %T, want *SyntaxError", err)
			}
		})
	}
}

func TestParseTOMLSizeBoundary(t *testing.T) {
	// Exactly MaxTOMLSize bytes is accepted.
	prefix := "A = \""
	input := prefix + strings.Repeat("x", MaxTOMLSize-len(prefix)-2) + "\"\n"
	if len(input) != MaxTOMLSize {
		t.Fatalf("test input is %d bytes, want %d", len(input), MaxTOMLSize)
	}
	if _, err := ParseTOML([]byte(input)); err != nil {
		t.Fatalf("ParseTOML at the limit: %v", err)
	}
}

func TestParseTOMLDeepNestingIsRejected(t *testing.T) {
	input := "A = " + strings.Repeat("[", 1000) + strings.Repeat("]", 1000) + "\n"
	if _, err := ParseTOML([]byte(input)); !errors.Is(err, ErrInvalidTOML) {
		t.Fatalf("err = %v, want ErrInvalidTOML", err)
	}
}
