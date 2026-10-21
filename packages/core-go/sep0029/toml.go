// Package sep0029 fetches and parses a domain's stellar.toml for the fields
// that deposit routing cares about: the organization name and which of the
// domain's accounts require a memo (the off-chain counterpart of the
// SEP-0029 config.memo_required account flag).
//
// The TOML parser handles the subset of TOML used by stellar.toml files and
// has no dependency beyond the standard library.
package sep0029

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

// MaxTOMLSize is the largest stellar.toml accepted, in bytes. SEP-0001 asks
// for files well under this; anything larger is rejected before parsing.
const MaxTOMLSize = 100 * 1024

var (
	// ErrTOMLTooLarge reports a stellar.toml larger than MaxTOMLSize.
	ErrTOMLTooLarge = errors.New("sep0029: stellar.toml exceeds 100KB")

	// ErrInvalidTOML reports a stellar.toml that is not valid TOML or not
	// valid UTF-8. The concrete error is a *SyntaxError.
	ErrInvalidTOML = errors.New("sep0029: invalid stellar.toml")
)

// SyntaxError describes where a stellar.toml failed to parse. It matches
// ErrInvalidTOML with errors.Is.
type SyntaxError struct {
	Line int    // 1-based line of the failure
	Msg  string // what was wrong
}

func (e *SyntaxError) Error() string {
	return fmt.Sprintf("sep0029: invalid stellar.toml: line %d: %s", e.Line, e.Msg)
}

// Is makes errors.Is(err, ErrInvalidTOML) true for every *SyntaxError.
func (e *SyntaxError) Is(target error) bool { return target == ErrInvalidTOML }

// StellarTOML holds the routing-relevant fields of a stellar.toml.
type StellarTOML struct {
	// OrgName is DOCUMENTATION.ORG_NAME, or "" if the file has none.
	OrgName string

	// NetworkPassphrase is the top-level NETWORK_PASSPHRASE, if present.
	NetworkPassphrase string

	// FederationServer is the top-level FEDERATION_SERVER, if present.
	FederationServer string

	// Accounts is the top-level ACCOUNTS list.
	Accounts []string

	// MemoRequiredAccounts is the top-level MEMO_REQUIRED_ACCOUNTS list: the
	// domain's accounts that reject payments without a memo. It is nil when
	// the key is absent, which means "unknown", not "no memo needed".
	MemoRequiredAccounts []string

	// Values holds every parsed key, addressed as "KEY" for top-level keys,
	// "TABLE.KEY" for keys in a [TABLE], and "TABLE.N.KEY" for the Nth
	// (0-based) [[TABLE]] entry. Values are string, int64, float64, bool,
	// []any or map[string]any.
	Values map[string]any
}

// RequiresMemo reports whether account is listed in MEMO_REQUIRED_ACCOUNTS.
// known is false when the file does not publish the list at all.
func (t *StellarTOML) RequiresMemo(account string) (required, known bool) {
	if t == nil || t.MemoRequiredAccounts == nil {
		return false, false
	}
	for _, a := range t.MemoRequiredAccounts {
		if a == account {
			return true, true
		}
	}
	return false, true
}

// ParseTOML parses a stellar.toml document. A UTF-8 byte order mark is
// ignored. Errors are ErrTOMLTooLarge or a *SyntaxError (ErrInvalidTOML).
// Missing fields are not errors: they are left empty.
func ParseTOML(data []byte) (*StellarTOML, error) {
	if len(data) > MaxTOMLSize {
		return nil, ErrTOMLTooLarge
	}
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	if !utf8.Valid(data) {
		return nil, &SyntaxError{Line: 1, Msg: "not valid UTF-8"}
	}

	p := &parser{
		src:     string(data),
		line:    1,
		values:  map[string]any{},
		lines:   map[string]int{},
		defined: map[string]bool{},
	}
	if err := p.document(); err != nil {
		return nil, err
	}

	t := &StellarTOML{Values: p.values}
	var err error
	if t.OrgName, err = stringField(p, "DOCUMENTATION.ORG_NAME"); err != nil {
		return nil, err
	}
	if t.NetworkPassphrase, err = stringField(p, "NETWORK_PASSPHRASE"); err != nil {
		return nil, err
	}
	if t.FederationServer, err = stringField(p, "FEDERATION_SERVER"); err != nil {
		return nil, err
	}
	if t.Accounts, err = stringList(p, "ACCOUNTS"); err != nil {
		return nil, err
	}
	if t.MemoRequiredAccounts, err = stringList(p, "MEMO_REQUIRED_ACCOUNTS"); err != nil {
		return nil, err
	}
	return t, nil
}

func stringField(p *parser, key string) (string, error) {
	v, ok := p.values[key]
	if !ok {
		return "", nil
	}
	s, ok := v.(string)
	if !ok {
		return "", &SyntaxError{Line: p.lines[key], Msg: key + " must be a string"}
	}
	return s, nil
}

func stringList(p *parser, key string) ([]string, error) {
	v, ok := p.values[key]
	if !ok {
		return nil, nil
	}
	items, ok := v.([]any)
	if !ok {
		return nil, &SyntaxError{Line: p.lines[key], Msg: key + " must be an array of strings"}
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		s, ok := item.(string)
		if !ok {
			return nil, &SyntaxError{Line: p.lines[key], Msg: key + " must be an array of strings"}
		}
		out = append(out, s)
	}
	return out, nil
}

// parser is a recursive-descent parser over the whole document, so
// multi-line strings and arrays need no special line handling.
type parser struct {
	src  string
	pos  int
	line int

	prefix  string          // "" or "TABLE." / "TABLE.N." for the current table
	values  map[string]any  // flattened keys, see StellarTOML.Values
	lines   map[string]int  // line on which each key in values was defined
	defined map[string]bool // table headers already seen
	arrays  map[string]int  // next index of each [[TABLE]]
}

func (p *parser) fail(format string, args ...any) error {
	return &SyntaxError{Line: p.line, Msg: fmt.Sprintf(format, args...)}
}

func (p *parser) eof() bool { return p.pos >= len(p.src) }

func (p *parser) peek() byte {
	if p.eof() {
		return 0
	}
	return p.src[p.pos]
}

func (p *parser) advance() byte {
	c := p.src[p.pos]
	p.pos++
	if c == '\n' {
		p.line++
	}
	return c
}

// skipSpace skips spaces and tabs on the current line.
func (p *parser) skipSpace() {
	for !p.eof() && (p.peek() == ' ' || p.peek() == '\t') {
		p.pos++
	}
}

// skipComment skips a # comment up to, not including, the newline.
func (p *parser) skipComment() {
	if p.peek() == '#' {
		for !p.eof() && p.peek() != '\n' {
			p.pos++
		}
	}
}

// endOfLine requires the rest of the line to be blank or a comment.
func (p *parser) endOfLine() error {
	p.skipSpace()
	p.skipComment()
	if p.eof() {
		return nil
	}
	if strings.HasPrefix(p.src[p.pos:], "\r\n") {
		p.pos++
	}
	if p.peek() != '\n' {
		return p.fail("unexpected %q after value", p.peek())
	}
	p.advance()
	return nil
}

func (p *parser) document() error {
	for {
		// Skip blank lines and comments.
		for !p.eof() {
			p.skipSpace()
			p.skipComment()
			if c := p.peek(); c == '\n' || c == '\r' {
				p.advance()
				continue
			}
			break
		}
		if p.eof() {
			return nil
		}

		var err error
		if p.peek() == '[' {
			err = p.tableHeader()
		} else {
			err = p.keyValue()
		}
		if err != nil {
			return err
		}
		if err := p.endOfLine(); err != nil {
			return err
		}
	}
}

func (p *parser) tableHeader() error {
	p.advance() // [
	array := p.peek() == '['
	if array {
		p.advance()
	}
	p.skipSpace()
	name, err := p.key()
	if err != nil {
		return err
	}
	p.skipSpace()
	closing := "]"
	if array {
		closing = "]]"
	}
	if !strings.HasPrefix(p.src[p.pos:], closing) {
		return p.fail("unterminated table header %q", name)
	}
	p.pos += len(closing)

	if array {
		if p.defined[name] {
			return p.fail("[[%s]] conflicts with table [%s]", name, name)
		}
		if p.arrays == nil {
			p.arrays = map[string]int{}
		}
		n := p.arrays[name]
		p.arrays[name] = n + 1
		p.prefix = name + "." + strconv.Itoa(n) + "."
		return nil
	}
	if p.defined[name] || p.arrays[name] > 0 {
		return p.fail("table [%s] defined twice", name)
	}
	p.defined[name] = true
	p.prefix = name + "."
	return nil
}

// key reads a bare or quoted key, joining dotted parts with ".".
func (p *parser) key() (string, error) {
	var parts []string
	for {
		p.skipSpace()
		var part string
		switch c := p.peek(); {
		case c == '"':
			s, err := p.basicString()
			if err != nil {
				return "", err
			}
			part = s
		case c == '\'':
			s, err := p.literalString()
			if err != nil {
				return "", err
			}
			part = s
		default:
			start := p.pos
			for !p.eof() && isBareKeyChar(p.peek()) {
				p.pos++
			}
			if p.pos == start {
				return "", p.fail("expected a key, found %q", p.peek())
			}
			part = p.src[start:p.pos]
		}
		parts = append(parts, part)
		p.skipSpace()
		if p.peek() != '.' {
			return strings.Join(parts, "."), nil
		}
		p.advance()
	}
}

func isBareKeyChar(c byte) bool {
	return c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_' || c == '-'
}

func (p *parser) keyValue() error {
	line := p.line
	k, err := p.key()
	if err != nil {
		return err
	}
	p.skipSpace()
	if p.peek() != '=' {
		return p.fail("expected '=' after key %q", k)
	}
	p.advance()
	p.skipSpace()
	v, err := p.value(0)
	if err != nil {
		return err
	}
	full := p.prefix + k
	if _, dup := p.values[full]; dup {
		return &SyntaxError{Line: line, Msg: fmt.Sprintf("key %q defined twice", full)}
	}
	p.values[full] = v
	p.lines[full] = line
	return nil
}

// maxNesting bounds array/inline-table nesting so hostile input cannot
// exhaust the stack.
const maxNesting = 32

func (p *parser) value(depth int) (any, error) {
	if depth > maxNesting {
		return nil, p.fail("values nested too deeply")
	}
	switch c := p.peek(); {
	case p.eof() || c == '\n' || c == '\r' || c == '#':
		return nil, p.fail("missing value")
	case strings.HasPrefix(p.src[p.pos:], `"""`):
		return p.multilineBasicString()
	case strings.HasPrefix(p.src[p.pos:], `'''`):
		return p.multilineLiteralString()
	case c == '"':
		return p.basicString()
	case c == '\'':
		return p.literalString()
	case c == '[':
		return p.array(depth)
	case c == '{':
		return p.inlineTable(depth)
	default:
		return p.scalar()
	}
}

func (p *parser) basicString() (string, error) {
	p.advance() // "
	var b strings.Builder
	for {
		if p.eof() || p.peek() == '\n' {
			return "", p.fail("unterminated string")
		}
		c := p.advance()
		switch c {
		case '"':
			return b.String(), nil
		case '\\':
			if err := p.escape(&b); err != nil {
				return "", err
			}
		default:
			b.WriteByte(c)
		}
	}
}

func (p *parser) escape(b *strings.Builder) error {
	if p.eof() {
		return p.fail("unterminated escape")
	}
	switch c := p.advance(); c {
	case 'b':
		b.WriteByte('\b')
	case 't':
		b.WriteByte('\t')
	case 'n':
		b.WriteByte('\n')
	case 'f':
		b.WriteByte('\f')
	case 'r':
		b.WriteByte('\r')
	case '"':
		b.WriteByte('"')
	case '\\':
		b.WriteByte('\\')
	case 'u', 'U':
		n := 4
		if c == 'U' {
			n = 8
		}
		if len(p.src)-p.pos < n {
			return p.fail("short unicode escape")
		}
		code, err := strconv.ParseUint(p.src[p.pos:p.pos+n], 16, 32)
		if err != nil || !utf8.ValidRune(rune(code)) {
			return p.fail("invalid unicode escape")
		}
		p.pos += n
		b.WriteRune(rune(code))
	default:
		return p.fail("invalid escape \\%c", c)
	}
	return nil
}

func (p *parser) literalString() (string, error) {
	p.advance() // '
	start := p.pos
	for {
		if p.eof() || p.peek() == '\n' {
			return "", p.fail("unterminated string")
		}
		if p.advance() == '\'' {
			return p.src[start : p.pos-1], nil
		}
	}
}

// trimLeadingNewline drops the newline directly after an opening """ or ”'.
func (p *parser) trimLeadingNewline() {
	if strings.HasPrefix(p.src[p.pos:], "\r\n") {
		p.pos++
	}
	if p.peek() == '\n' {
		p.advance()
	}
}

func (p *parser) multilineBasicString() (string, error) {
	p.pos += 3
	p.trimLeadingNewline()
	var b strings.Builder
	for {
		if p.eof() {
			return "", p.fail("unterminated multi-line string")
		}
		if strings.HasPrefix(p.src[p.pos:], `"""`) {
			p.pos += 3
			return b.String(), nil
		}
		c := p.advance()
		if c != '\\' {
			b.WriteByte(c)
			continue
		}
		// A backslash at the end of a line trims the newline and the
		// whitespace that follows it.
		rest := p.src[p.pos:]
		trimmed := strings.TrimLeft(rest, " \t")
		if strings.HasPrefix(trimmed, "\n") || strings.HasPrefix(trimmed, "\r\n") {
			for !p.eof() && strings.ContainsRune(" \t\r\n", rune(p.peek())) {
				p.advance()
			}
			continue
		}
		if err := p.escape(&b); err != nil {
			return "", err
		}
	}
}

func (p *parser) multilineLiteralString() (string, error) {
	p.pos += 3
	p.trimLeadingNewline()
	start := p.pos
	for {
		if p.eof() {
			return "", p.fail("unterminated multi-line string")
		}
		if strings.HasPrefix(p.src[p.pos:], "'''") {
			s := p.src[start:p.pos]
			p.pos += 3
			return s, nil
		}
		p.advance()
	}
}

// skipArraySpace skips whitespace, newlines and comments inside an array.
func (p *parser) skipArraySpace() {
	for !p.eof() {
		switch p.peek() {
		case ' ', '\t', '\r', '\n':
			p.advance()
		case '#':
			p.skipComment()
		default:
			return
		}
	}
}

func (p *parser) array(depth int) ([]any, error) {
	p.advance() // [
	items := []any{}
	for {
		p.skipArraySpace()
		if p.eof() {
			return nil, p.fail("unterminated array")
		}
		if p.peek() == ']' {
			p.advance()
			return items, nil
		}
		v, err := p.value(depth + 1)
		if err != nil {
			return nil, err
		}
		items = append(items, v)
		p.skipArraySpace()
		switch p.peek() {
		case ',':
			p.advance()
		case ']':
		default:
			return nil, p.fail("expected ',' or ']' in array")
		}
	}
}

func (p *parser) inlineTable(depth int) (map[string]any, error) {
	p.advance() // {
	table := map[string]any{}
	p.skipSpace()
	if p.peek() == '}' {
		p.advance()
		return table, nil
	}
	for {
		k, err := p.key()
		if err != nil {
			return nil, err
		}
		p.skipSpace()
		if p.peek() != '=' {
			return nil, p.fail("expected '=' after key %q", k)
		}
		p.advance()
		p.skipSpace()
		v, err := p.value(depth + 1)
		if err != nil {
			return nil, err
		}
		if _, dup := table[k]; dup {
			return nil, p.fail("key %q defined twice", k)
		}
		table[k] = v
		p.skipSpace()
		switch p.peek() {
		case ',':
			p.advance()
			p.skipSpace()
		case '}':
			p.advance()
			return table, nil
		default:
			return nil, p.fail("expected ',' or '}' in inline table")
		}
	}
}

// scalar reads a boolean, integer or float up to the next delimiter.
func (p *parser) scalar() (any, error) {
	start := p.pos
	for !p.eof() && !strings.ContainsRune(" \t\r\n#,]}", rune(p.peek())) {
		p.pos++
	}
	raw := p.src[start:p.pos]
	switch raw {
	case "true":
		return true, nil
	case "false":
		return false, nil
	}
	// Underscores may only separate digits: not lead, trail or repeat.
	clean := strings.ReplaceAll(raw, "_", "")
	if clean != "" && raw[0] != '_' && raw[len(raw)-1] != '_' && !strings.Contains(raw, "__") {
		if n, err := strconv.ParseInt(clean, 10, 64); err == nil {
			return n, nil
		}
		if f, err := strconv.ParseFloat(clean, 64); err == nil && strings.ContainsAny(clean, "0123456789") {
			return f, nil
		}
	}
	return nil, p.fail("invalid value %q", raw)
}
