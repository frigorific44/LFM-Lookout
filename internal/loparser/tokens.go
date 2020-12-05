package loparser

type Token int

const (
  // Special tokens
  ILLEGAL Token = iota
  INVALIDP // an invalid phrase (w/o closing neutral quote)
  INVALIDO // an invalid operator (likely missing parenthesis or quotation mark)
  EOF
  WS

  // Seperators
  LPAR // (
  RPAR // )

  // Literals
  PHRASE // characters delimitted by a pair of quotes

  // Operators
  AND
  OR
)
