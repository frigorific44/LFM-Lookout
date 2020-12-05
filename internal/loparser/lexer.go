package loparser

import (
  "bufio"
  "bytes"
  "io"
  "strings"
  "unicode"
)


type Scanner struct {
  r *bufio.Reader
}


func NewScanner(r io.Reader) *Scanner {
  return &Scanner{r: bufio.NewReader(r)}
}


func (s *Scanner) Scan() (tok Token, lit string) {
  c := s.read()

  if isWhitespace(c) {
    s.unread()
    return s.scanWhitespace()
  } else if c == '"'{
    s.unread()
    return s.scanPhrase()
  } else if isLetter(c) {
    s.unread()
    return s.scanOperator()
  }

  switch c {
  case eof:
    return EOF, ""
  case '(':
    return LPAR, string(c)
  case ')':
    return RPAR, string(c)
  }

  return ILLEGAL, string(c)
}


func (s *Scanner) scanWhitespace() (tok Token, lit string) {
  var buf bytes.Buffer
  buf.WriteRune(s.read())

  for {
    if c := s.read(); c == eof {
      break
    } else if !isWhitespace(c) {
      s.unread()
      break
    } else {
      buf.WriteRune(c)
    }
  }

  return WS, buf.String()
}


func (s *Scanner) scanPhrase() (tok Token, lit string) {
  var buf bytes.Buffer
  buf.WriteRune(s.read())

  for {
    if c:= s.read(); c == eof {
      break
    } else if c == '"' {
      buf.WriteRune(c)
      break
    } else {
      buf.WriteRune(c)
    }
  }

  if strings.HasSuffix(buf.String(), `"`) && (len(buf.String()) > 0) {
    return PHRASE, buf.String()
  }

  return INVALIDP, buf.String()
}


func (s *Scanner) scanOperator() (tok Token, lit string) {
  var buf bytes.Buffer
  buf.WriteRune(s.read())

  for {
    if c := s.read(); c == eof {
      break
    } else if !isLetter(c) {
      s.unread()
      break
    } else {
      _, _ = buf.WriteRune(c)
    }
  }

  switch strings.ToUpper(buf.String()) {
  case "AND":
    return AND, buf.String()
  case "OR":
    return OR, buf.String()
  }

  return INVALIDO, buf.String()
}


func (s *Scanner) read() rune {
  c, _, err := s.r.ReadRune()
  if err != nil {
    return eof
  }
  return c
}


func (s *Scanner) unread() {
  _ = s.r.UnreadRune()
}


func isWhitespace(c rune) bool {
  return unicode.IsSpace(c)
}


func isLetter(c rune) bool {
  return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}


var eof = rune(0)
