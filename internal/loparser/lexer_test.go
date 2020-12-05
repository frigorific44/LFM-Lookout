package loparser_test

import (
  "strings"
  "testing"

  "lfm_lookout/internal/loparser"
)

func TestScanner_Scan(t *testing.T) {
  var tests = []struct {
    s string
    tok loparser.Token
    lit string
  }{
    // ILLEGAL
    {s: `,`, tok: loparser.ILLEGAL, lit: `,`},
    {s: `.`, tok: loparser.ILLEGAL, lit: `.`},
    {s: `!`, tok: loparser.ILLEGAL, lit: `!`},
    // INVALIDP
    {s: `"Where do I end`, tok: loparser.INVALIDP, lit: `"Where do I end`},
    // INVALIDO
    {s: `OiOiOi`, tok: loparser.INVALIDO, lit: "OiOiOi"},
    {s: `AndOr`, tok: loparser.INVALIDO, lit: "AndOr"},
    // EOF
    {s: ``, tok: loparser.EOF},
    // WS
		{s: ` `, tok: loparser.WS, lit: " "},
		{s: "\t", tok: loparser.WS, lit: "\t"},
		{s: "\n", tok: loparser.WS, lit: "\n"},

    // Seperators
    // LPAR
    {s: `(`, tok: loparser.LPAR, lit:"("},
    // RPAR
    {s: `)`, tok: loparser.RPAR, lit:")"},

    // Literals
    // PHRASE
    {s: `"Ah, complete."`, tok: loparser.PHRASE, lit: "\"Ah, complete.\""},

    // Operators
    // AND
    {s: `AND`, tok: loparser.AND, lit:"AND"},
    {s: `and`, tok: loparser.AND, lit:"and"},
    {s: `And`, tok: loparser.AND, lit:"And"},
    // OR
    {s: `OR`, tok: loparser.OR, lit:"OR"},
    {s: `or`, tok: loparser.OR, lit:"or"},
    {s: `Or`, tok: loparser.OR, lit:"Or"},
  }

  for i, tt := range tests {
    s := loparser.NewScanner(strings.NewReader(tt.s))
    tok, lit := s.Scan()
    if tt.tok != tok {
      t.Errorf("%d. %q token mismatch: exp=%q got=%q <%q>", i, tt.s, tt.tok, tok, lit)
    } else if tt.lit != lit {
      t.Errorf("%d. %q literal mismatch: exp=%q got=%q", i, tt.s, tt.lit, lit)
    }
  }
}
