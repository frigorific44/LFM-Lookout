package loparser

import (
  "fmt"
  "io"
)


type Parser struct {
  s *Scanner
  buf struct {
    tok Token
    lit string
    n int
  }
}

func NewParser(r io.Reader) *Parser {
  return &Parser{s: NewScanner(r)}
}

func (p *Parser) Parse() (*SearchStatement, error) {
  stmt := &SelectStatement{}
}

func (p *Parser) scan() (tok Token, lit string) {
  if p.buf.n != 0 {
    p.buf.n = 0
    return p.buf.tok, p.buf.lit
  }

  tok, lit = p.s.Scan()

  p.buf.tok, p.buf.lit = tok, lit

  return
}

func (p *Parser) unscan() {
  p.buf.n = 1
}

func (p *Parser) scanIgnoreWhitespace() (tok Token, lit string) {
  tok, lit = p.scan()
  if tok == WS {
    tok, lit = p.scan()
  }
  return
}
