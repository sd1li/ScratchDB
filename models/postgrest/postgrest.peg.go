package postgrest

// Code generated by peg postgrest.peg DO NOT EDIT.

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
)

const endSymbol rune = 1114112

/* The rule types inferred from the grammar are below. */
type pegRule uint8

const (
	ruleUnknown pegRule = iota
	ruleQueryString
	ruleQueryParam
	ruleLimit
	ruleOffset
	ruleFilter
	ruleColumnName
	rulePredicate
	ruleNot
	ruleOperator
	ruleAnyAll
	ruleOperand
	ruleListOperand
	ruleListOperandItem
	ruleVectorOperand
	ruleVectorOperandItem
	ruleQuotedString
	ruleEscapedChar
	ruleScalarOperand
	ruleInteger
	ruleEND
)

var rul3s = [...]string{
	"Unknown",
	"QueryString",
	"QueryParam",
	"Limit",
	"Offset",
	"Filter",
	"ColumnName",
	"Predicate",
	"Not",
	"Operator",
	"AnyAll",
	"Operand",
	"ListOperand",
	"ListOperandItem",
	"VectorOperand",
	"VectorOperandItem",
	"QuotedString",
	"EscapedChar",
	"ScalarOperand",
	"Integer",
	"END",
}

type token32 struct {
	pegRule
	begin, end uint32
}

func (t *token32) String() string {
	return fmt.Sprintf("\x1B[34m%v\x1B[m %v %v", rul3s[t.pegRule], t.begin, t.end)
}

type node32 struct {
	token32
	up, next *node32
}

func (node *node32) print(w io.Writer, pretty bool, buffer string) {
	var print func(node *node32, depth int)
	print = func(node *node32, depth int) {
		for node != nil {
			for c := 0; c < depth; c++ {
				fmt.Fprintf(w, " ")
			}
			rule := rul3s[node.pegRule]
			quote := strconv.Quote(string(([]rune(buffer)[node.begin:node.end])))
			if !pretty {
				fmt.Fprintf(w, "%v %v\n", rule, quote)
			} else {
				fmt.Fprintf(w, "\x1B[36m%v\x1B[m %v\n", rule, quote)
			}
			if node.up != nil {
				print(node.up, depth+1)
			}
			node = node.next
		}
	}
	print(node, 0)
}

func (node *node32) Print(w io.Writer, buffer string) {
	node.print(w, false, buffer)
}

func (node *node32) PrettyPrint(w io.Writer, buffer string) {
	node.print(w, true, buffer)
}

type tokens32 struct {
	tree []token32
}

func (t *tokens32) Trim(length uint32) {
	t.tree = t.tree[:length]
}

func (t *tokens32) Print() {
	for _, token := range t.tree {
		fmt.Println(token.String())
	}
}

func (t *tokens32) AST() *node32 {
	type element struct {
		node *node32
		down *element
	}
	tokens := t.Tokens()
	var stack *element
	for _, token := range tokens {
		if token.begin == token.end {
			continue
		}
		node := &node32{token32: token}
		for stack != nil && stack.node.begin >= token.begin && stack.node.end <= token.end {
			stack.node.next = node.up
			node.up = stack.node
			stack = stack.down
		}
		stack = &element{node: node, down: stack}
	}
	if stack != nil {
		return stack.node
	}
	return nil
}

func (t *tokens32) PrintSyntaxTree(buffer string) {
	t.AST().Print(os.Stdout, buffer)
}

func (t *tokens32) WriteSyntaxTree(w io.Writer, buffer string) {
	t.AST().Print(w, buffer)
}

func (t *tokens32) PrettyPrintSyntaxTree(buffer string) {
	t.AST().PrettyPrint(os.Stdout, buffer)
}

func (t *tokens32) Add(rule pegRule, begin, end, index uint32) {
	tree, i := t.tree, int(index)
	if i >= len(tree) {
		t.tree = append(tree, token32{pegRule: rule, begin: begin, end: end})
		return
	}
	tree[i] = token32{pegRule: rule, begin: begin, end: end}
}

func (t *tokens32) Tokens() []token32 {
	return t.tree
}

type PostgrestParser struct {
	Buffer string
	buffer []rune
	rules  [21]func() bool
	parse  func(rule ...int) error
	reset  func()
	Pretty bool
	tokens32
}

func (p *PostgrestParser) Parse(rule ...int) error {
	return p.parse(rule...)
}

func (p *PostgrestParser) Reset() {
	p.reset()
}

type textPosition struct {
	line, symbol int
}

type textPositionMap map[int]textPosition

func translatePositions(buffer []rune, positions []int) textPositionMap {
	length, translations, j, line, symbol := len(positions), make(textPositionMap, len(positions)), 0, 1, 0
	sort.Ints(positions)

search:
	for i, c := range buffer {
		if c == '\n' {
			line, symbol = line+1, 0
		} else {
			symbol++
		}
		if i == positions[j] {
			translations[positions[j]] = textPosition{line, symbol}
			for j++; j < length; j++ {
				if i != positions[j] {
					continue search
				}
			}
			break search
		}
	}

	return translations
}

type parseError struct {
	p   *PostgrestParser
	max token32
}

func (e *parseError) Error() string {
	tokens, err := []token32{e.max}, "\n"
	positions, p := make([]int, 2*len(tokens)), 0
	for _, token := range tokens {
		positions[p], p = int(token.begin), p+1
		positions[p], p = int(token.end), p+1
	}
	translations := translatePositions(e.p.buffer, positions)
	format := "parse error near %v (line %v symbol %v - line %v symbol %v):\n%v\n"
	if e.p.Pretty {
		format = "parse error near \x1B[34m%v\x1B[m (line %v symbol %v - line %v symbol %v):\n%v\n"
	}
	for _, token := range tokens {
		begin, end := int(token.begin), int(token.end)
		err += fmt.Sprintf(format,
			rul3s[token.pegRule],
			translations[begin].line, translations[begin].symbol,
			translations[end].line, translations[end].symbol,
			strconv.Quote(string(e.p.buffer[begin:end])))
	}

	return err
}

func (p *PostgrestParser) PrintSyntaxTree() {
	if p.Pretty {
		p.tokens32.PrettyPrintSyntaxTree(p.Buffer)
	} else {
		p.tokens32.PrintSyntaxTree(p.Buffer)
	}
}

func (p *PostgrestParser) WriteSyntaxTree(w io.Writer) {
	p.tokens32.WriteSyntaxTree(w, p.Buffer)
}

func (p *PostgrestParser) SprintSyntaxTree() string {
	var bldr strings.Builder
	p.WriteSyntaxTree(&bldr)
	return bldr.String()
}

func Pretty(pretty bool) func(*PostgrestParser) error {
	return func(p *PostgrestParser) error {
		p.Pretty = pretty
		return nil
	}
}

func Size(size int) func(*PostgrestParser) error {
	return func(p *PostgrestParser) error {
		p.tokens32 = tokens32{tree: make([]token32, 0, size)}
		return nil
	}
}
func (p *PostgrestParser) Init(options ...func(*PostgrestParser) error) error {
	var (
		max                  token32
		position, tokenIndex uint32
		buffer               []rune
	)
	for _, option := range options {
		err := option(p)
		if err != nil {
			return err
		}
	}
	p.reset = func() {
		max = token32{}
		position, tokenIndex = 0, 0

		p.buffer = []rune(p.Buffer)
		if len(p.buffer) == 0 || p.buffer[len(p.buffer)-1] != endSymbol {
			p.buffer = append(p.buffer, endSymbol)
		}
		buffer = p.buffer
	}
	p.reset()

	_rules := p.rules
	tree := p.tokens32
	p.parse = func(rule ...int) error {
		r := 1
		if len(rule) > 0 {
			r = rule[0]
		}
		matches := p.rules[r]()
		p.tokens32 = tree
		if matches {
			p.Trim(tokenIndex)
			return nil
		}
		return &parseError{p, max}
	}

	add := func(rule pegRule, begin uint32) {
		tree.Add(rule, begin, position, tokenIndex)
		tokenIndex++
		if begin != position && position > max.end {
			max = token32{rule, begin, position}
		}
	}

	matchDot := func() bool {
		if buffer[position] != endSymbol {
			position++
			return true
		}
		return false
	}

	/*matchChar := func(c byte) bool {
		if buffer[position] == c {
			position++
			return true
		}
		return false
	}*/

	/*matchRange := func(lower byte, upper byte) bool {
		if c := buffer[position]; c >= lower && c <= upper {
			position++
			return true
		}
		return false
	}*/

	_rules = [...]func() bool{
		nil,
		/* 0 QueryString <- <(QueryParam? ('&' QueryParam)* END)> */
		func() bool {
			position0, tokenIndex0 := position, tokenIndex
			{
				position1 := position
				{
					position2, tokenIndex2 := position, tokenIndex
					if !_rules[ruleQueryParam]() {
						goto l2
					}
					goto l3
				l2:
					position, tokenIndex = position2, tokenIndex2
				}
			l3:
			l4:
				{
					position5, tokenIndex5 := position, tokenIndex
					if buffer[position] != rune('&') {
						goto l5
					}
					position++
					if !_rules[ruleQueryParam]() {
						goto l5
					}
					goto l4
				l5:
					position, tokenIndex = position5, tokenIndex5
				}
				if !_rules[ruleEND]() {
					goto l0
				}
				add(ruleQueryString, position1)
			}
			return true
		l0:
			position, tokenIndex = position0, tokenIndex0
			return false
		},
		/* 1 QueryParam <- <(Limit / Offset / Filter)> */
		func() bool {
			position6, tokenIndex6 := position, tokenIndex
			{
				position7 := position
				{
					position8, tokenIndex8 := position, tokenIndex
					if !_rules[ruleLimit]() {
						goto l9
					}
					goto l8
				l9:
					position, tokenIndex = position8, tokenIndex8
					if !_rules[ruleOffset]() {
						goto l10
					}
					goto l8
				l10:
					position, tokenIndex = position8, tokenIndex8
					if !_rules[ruleFilter]() {
						goto l6
					}
				}
			l8:
				add(ruleQueryParam, position7)
			}
			return true
		l6:
			position, tokenIndex = position6, tokenIndex6
			return false
		},
		/* 2 Limit <- <('l' 'i' 'm' 'i' 't' '=' Integer)> */
		func() bool {
			position11, tokenIndex11 := position, tokenIndex
			{
				position12 := position
				if buffer[position] != rune('l') {
					goto l11
				}
				position++
				if buffer[position] != rune('i') {
					goto l11
				}
				position++
				if buffer[position] != rune('m') {
					goto l11
				}
				position++
				if buffer[position] != rune('i') {
					goto l11
				}
				position++
				if buffer[position] != rune('t') {
					goto l11
				}
				position++
				if buffer[position] != rune('=') {
					goto l11
				}
				position++
				if !_rules[ruleInteger]() {
					goto l11
				}
				add(ruleLimit, position12)
			}
			return true
		l11:
			position, tokenIndex = position11, tokenIndex11
			return false
		},
		/* 3 Offset <- <('o' 'f' 'f' 's' 'e' 't' '=' Integer)> */
		func() bool {
			position13, tokenIndex13 := position, tokenIndex
			{
				position14 := position
				if buffer[position] != rune('o') {
					goto l13
				}
				position++
				if buffer[position] != rune('f') {
					goto l13
				}
				position++
				if buffer[position] != rune('f') {
					goto l13
				}
				position++
				if buffer[position] != rune('s') {
					goto l13
				}
				position++
				if buffer[position] != rune('e') {
					goto l13
				}
				position++
				if buffer[position] != rune('t') {
					goto l13
				}
				position++
				if buffer[position] != rune('=') {
					goto l13
				}
				position++
				if !_rules[ruleInteger]() {
					goto l13
				}
				add(ruleOffset, position14)
			}
			return true
		l13:
			position, tokenIndex = position13, tokenIndex13
			return false
		},
		/* 4 Filter <- <(ColumnName '=' Predicate)> */
		func() bool {
			position15, tokenIndex15 := position, tokenIndex
			{
				position16 := position
				if !_rules[ruleColumnName]() {
					goto l15
				}
				if buffer[position] != rune('=') {
					goto l15
				}
				position++
				if !_rules[rulePredicate]() {
					goto l15
				}
				add(ruleFilter, position16)
			}
			return true
		l15:
			position, tokenIndex = position15, tokenIndex15
			return false
		},
		/* 5 ColumnName <- <(!('=' / '&') .)+> */
		func() bool {
			position17, tokenIndex17 := position, tokenIndex
			{
				position18 := position
				{
					position21, tokenIndex21 := position, tokenIndex
					{
						position22, tokenIndex22 := position, tokenIndex
						if buffer[position] != rune('=') {
							goto l23
						}
						position++
						goto l22
					l23:
						position, tokenIndex = position22, tokenIndex22
						if buffer[position] != rune('&') {
							goto l21
						}
						position++
					}
				l22:
					goto l17
				l21:
					position, tokenIndex = position21, tokenIndex21
				}
				if !matchDot() {
					goto l17
				}
			l19:
				{
					position20, tokenIndex20 := position, tokenIndex
					{
						position24, tokenIndex24 := position, tokenIndex
						{
							position25, tokenIndex25 := position, tokenIndex
							if buffer[position] != rune('=') {
								goto l26
							}
							position++
							goto l25
						l26:
							position, tokenIndex = position25, tokenIndex25
							if buffer[position] != rune('&') {
								goto l24
							}
							position++
						}
					l25:
						goto l20
					l24:
						position, tokenIndex = position24, tokenIndex24
					}
					if !matchDot() {
						goto l20
					}
					goto l19
				l20:
					position, tokenIndex = position20, tokenIndex20
				}
				add(ruleColumnName, position18)
			}
			return true
		l17:
			position, tokenIndex = position17, tokenIndex17
			return false
		},
		/* 6 Predicate <- <(Not? ((Operator '.' Operand) / (Operator '(' AnyAll ')' '.' ListOperand)))> */
		func() bool {
			position27, tokenIndex27 := position, tokenIndex
			{
				position28 := position
				{
					position29, tokenIndex29 := position, tokenIndex
					if !_rules[ruleNot]() {
						goto l29
					}
					goto l30
				l29:
					position, tokenIndex = position29, tokenIndex29
				}
			l30:
				{
					position31, tokenIndex31 := position, tokenIndex
					if !_rules[ruleOperator]() {
						goto l32
					}
					if buffer[position] != rune('.') {
						goto l32
					}
					position++
					if !_rules[ruleOperand]() {
						goto l32
					}
					goto l31
				l32:
					position, tokenIndex = position31, tokenIndex31
					if !_rules[ruleOperator]() {
						goto l27
					}
					if buffer[position] != rune('(') {
						goto l27
					}
					position++
					if !_rules[ruleAnyAll]() {
						goto l27
					}
					if buffer[position] != rune(')') {
						goto l27
					}
					position++
					if buffer[position] != rune('.') {
						goto l27
					}
					position++
					if !_rules[ruleListOperand]() {
						goto l27
					}
				}
			l31:
				add(rulePredicate, position28)
			}
			return true
		l27:
			position, tokenIndex = position27, tokenIndex27
			return false
		},
		/* 7 Not <- <('n' 'o' 't' '.')> */
		func() bool {
			position33, tokenIndex33 := position, tokenIndex
			{
				position34 := position
				if buffer[position] != rune('n') {
					goto l33
				}
				position++
				if buffer[position] != rune('o') {
					goto l33
				}
				position++
				if buffer[position] != rune('t') {
					goto l33
				}
				position++
				if buffer[position] != rune('.') {
					goto l33
				}
				position++
				add(ruleNot, position34)
			}
			return true
		l33:
			position, tokenIndex = position33, tokenIndex33
			return false
		},
		/* 8 Operator <- <([a-z] / [A-Z])+> */
		func() bool {
			position35, tokenIndex35 := position, tokenIndex
			{
				position36 := position
				{
					position39, tokenIndex39 := position, tokenIndex
					if c := buffer[position]; c < rune('a') || c > rune('z') {
						goto l40
					}
					position++
					goto l39
				l40:
					position, tokenIndex = position39, tokenIndex39
					if c := buffer[position]; c < rune('A') || c > rune('Z') {
						goto l35
					}
					position++
				}
			l39:
			l37:
				{
					position38, tokenIndex38 := position, tokenIndex
					{
						position41, tokenIndex41 := position, tokenIndex
						if c := buffer[position]; c < rune('a') || c > rune('z') {
							goto l42
						}
						position++
						goto l41
					l42:
						position, tokenIndex = position41, tokenIndex41
						if c := buffer[position]; c < rune('A') || c > rune('Z') {
							goto l38
						}
						position++
					}
				l41:
					goto l37
				l38:
					position, tokenIndex = position38, tokenIndex38
				}
				add(ruleOperator, position36)
			}
			return true
		l35:
			position, tokenIndex = position35, tokenIndex35
			return false
		},
		/* 9 AnyAll <- <(('a' 'n' 'y') / ('a' 'l' 'l'))> */
		func() bool {
			position43, tokenIndex43 := position, tokenIndex
			{
				position44 := position
				{
					position45, tokenIndex45 := position, tokenIndex
					if buffer[position] != rune('a') {
						goto l46
					}
					position++
					if buffer[position] != rune('n') {
						goto l46
					}
					position++
					if buffer[position] != rune('y') {
						goto l46
					}
					position++
					goto l45
				l46:
					position, tokenIndex = position45, tokenIndex45
					if buffer[position] != rune('a') {
						goto l43
					}
					position++
					if buffer[position] != rune('l') {
						goto l43
					}
					position++
					if buffer[position] != rune('l') {
						goto l43
					}
					position++
				}
			l45:
				add(ruleAnyAll, position44)
			}
			return true
		l43:
			position, tokenIndex = position43, tokenIndex43
			return false
		},
		/* 10 Operand <- <(VectorOperand / ScalarOperand)> */
		func() bool {
			position47, tokenIndex47 := position, tokenIndex
			{
				position48 := position
				{
					position49, tokenIndex49 := position, tokenIndex
					if !_rules[ruleVectorOperand]() {
						goto l50
					}
					goto l49
				l50:
					position, tokenIndex = position49, tokenIndex49
					if !_rules[ruleScalarOperand]() {
						goto l47
					}
				}
			l49:
				add(ruleOperand, position48)
			}
			return true
		l47:
			position, tokenIndex = position47, tokenIndex47
			return false
		},
		/* 11 ListOperand <- <('{' ListOperandItem (',' ListOperandItem)* '}')> */
		func() bool {
			position51, tokenIndex51 := position, tokenIndex
			{
				position52 := position
				if buffer[position] != rune('{') {
					goto l51
				}
				position++
				if !_rules[ruleListOperandItem]() {
					goto l51
				}
			l53:
				{
					position54, tokenIndex54 := position, tokenIndex
					if buffer[position] != rune(',') {
						goto l54
					}
					position++
					if !_rules[ruleListOperandItem]() {
						goto l54
					}
					goto l53
				l54:
					position, tokenIndex = position54, tokenIndex54
				}
				if buffer[position] != rune('}') {
					goto l51
				}
				position++
				add(ruleListOperand, position52)
			}
			return true
		l51:
			position, tokenIndex = position51, tokenIndex51
			return false
		},
		/* 12 ListOperandItem <- <(QuotedString / (!(',' / '}' / '&' / '=') .)+)> */
		func() bool {
			position55, tokenIndex55 := position, tokenIndex
			{
				position56 := position
				{
					position57, tokenIndex57 := position, tokenIndex
					if !_rules[ruleQuotedString]() {
						goto l58
					}
					goto l57
				l58:
					position, tokenIndex = position57, tokenIndex57
					{
						position61, tokenIndex61 := position, tokenIndex
						{
							position62, tokenIndex62 := position, tokenIndex
							if buffer[position] != rune(',') {
								goto l63
							}
							position++
							goto l62
						l63:
							position, tokenIndex = position62, tokenIndex62
							if buffer[position] != rune('}') {
								goto l64
							}
							position++
							goto l62
						l64:
							position, tokenIndex = position62, tokenIndex62
							if buffer[position] != rune('&') {
								goto l65
							}
							position++
							goto l62
						l65:
							position, tokenIndex = position62, tokenIndex62
							if buffer[position] != rune('=') {
								goto l61
							}
							position++
						}
					l62:
						goto l55
					l61:
						position, tokenIndex = position61, tokenIndex61
					}
					if !matchDot() {
						goto l55
					}
				l59:
					{
						position60, tokenIndex60 := position, tokenIndex
						{
							position66, tokenIndex66 := position, tokenIndex
							{
								position67, tokenIndex67 := position, tokenIndex
								if buffer[position] != rune(',') {
									goto l68
								}
								position++
								goto l67
							l68:
								position, tokenIndex = position67, tokenIndex67
								if buffer[position] != rune('}') {
									goto l69
								}
								position++
								goto l67
							l69:
								position, tokenIndex = position67, tokenIndex67
								if buffer[position] != rune('&') {
									goto l70
								}
								position++
								goto l67
							l70:
								position, tokenIndex = position67, tokenIndex67
								if buffer[position] != rune('=') {
									goto l66
								}
								position++
							}
						l67:
							goto l60
						l66:
							position, tokenIndex = position66, tokenIndex66
						}
						if !matchDot() {
							goto l60
						}
						goto l59
					l60:
						position, tokenIndex = position60, tokenIndex60
					}
				}
			l57:
				add(ruleListOperandItem, position56)
			}
			return true
		l55:
			position, tokenIndex = position55, tokenIndex55
			return false
		},
		/* 13 VectorOperand <- <('(' VectorOperandItem (',' VectorOperandItem)* ')')> */
		func() bool {
			position71, tokenIndex71 := position, tokenIndex
			{
				position72 := position
				if buffer[position] != rune('(') {
					goto l71
				}
				position++
				if !_rules[ruleVectorOperandItem]() {
					goto l71
				}
			l73:
				{
					position74, tokenIndex74 := position, tokenIndex
					if buffer[position] != rune(',') {
						goto l74
					}
					position++
					if !_rules[ruleVectorOperandItem]() {
						goto l74
					}
					goto l73
				l74:
					position, tokenIndex = position74, tokenIndex74
				}
				if buffer[position] != rune(')') {
					goto l71
				}
				position++
				add(ruleVectorOperand, position72)
			}
			return true
		l71:
			position, tokenIndex = position71, tokenIndex71
			return false
		},
		/* 14 VectorOperandItem <- <(QuotedString / (!(',' / ')' / '&' / '=') .)+)> */
		func() bool {
			position75, tokenIndex75 := position, tokenIndex
			{
				position76 := position
				{
					position77, tokenIndex77 := position, tokenIndex
					if !_rules[ruleQuotedString]() {
						goto l78
					}
					goto l77
				l78:
					position, tokenIndex = position77, tokenIndex77
					{
						position81, tokenIndex81 := position, tokenIndex
						{
							position82, tokenIndex82 := position, tokenIndex
							if buffer[position] != rune(',') {
								goto l83
							}
							position++
							goto l82
						l83:
							position, tokenIndex = position82, tokenIndex82
							if buffer[position] != rune(')') {
								goto l84
							}
							position++
							goto l82
						l84:
							position, tokenIndex = position82, tokenIndex82
							if buffer[position] != rune('&') {
								goto l85
							}
							position++
							goto l82
						l85:
							position, tokenIndex = position82, tokenIndex82
							if buffer[position] != rune('=') {
								goto l81
							}
							position++
						}
					l82:
						goto l75
					l81:
						position, tokenIndex = position81, tokenIndex81
					}
					if !matchDot() {
						goto l75
					}
				l79:
					{
						position80, tokenIndex80 := position, tokenIndex
						{
							position86, tokenIndex86 := position, tokenIndex
							{
								position87, tokenIndex87 := position, tokenIndex
								if buffer[position] != rune(',') {
									goto l88
								}
								position++
								goto l87
							l88:
								position, tokenIndex = position87, tokenIndex87
								if buffer[position] != rune(')') {
									goto l89
								}
								position++
								goto l87
							l89:
								position, tokenIndex = position87, tokenIndex87
								if buffer[position] != rune('&') {
									goto l90
								}
								position++
								goto l87
							l90:
								position, tokenIndex = position87, tokenIndex87
								if buffer[position] != rune('=') {
									goto l86
								}
								position++
							}
						l87:
							goto l80
						l86:
							position, tokenIndex = position86, tokenIndex86
						}
						if !matchDot() {
							goto l80
						}
						goto l79
					l80:
						position, tokenIndex = position80, tokenIndex80
					}
				}
			l77:
				add(ruleVectorOperandItem, position76)
			}
			return true
		l75:
			position, tokenIndex = position75, tokenIndex75
			return false
		},
		/* 15 QuotedString <- <('"' (EscapedChar / (!('"' / '&' / '=') .))* '"')> */
		func() bool {
			position91, tokenIndex91 := position, tokenIndex
			{
				position92 := position
				if buffer[position] != rune('"') {
					goto l91
				}
				position++
			l93:
				{
					position94, tokenIndex94 := position, tokenIndex
					{
						position95, tokenIndex95 := position, tokenIndex
						if !_rules[ruleEscapedChar]() {
							goto l96
						}
						goto l95
					l96:
						position, tokenIndex = position95, tokenIndex95
						{
							position97, tokenIndex97 := position, tokenIndex
							{
								position98, tokenIndex98 := position, tokenIndex
								if buffer[position] != rune('"') {
									goto l99
								}
								position++
								goto l98
							l99:
								position, tokenIndex = position98, tokenIndex98
								if buffer[position] != rune('&') {
									goto l100
								}
								position++
								goto l98
							l100:
								position, tokenIndex = position98, tokenIndex98
								if buffer[position] != rune('=') {
									goto l97
								}
								position++
							}
						l98:
							goto l94
						l97:
							position, tokenIndex = position97, tokenIndex97
						}
						if !matchDot() {
							goto l94
						}
					}
				l95:
					goto l93
				l94:
					position, tokenIndex = position94, tokenIndex94
				}
				if buffer[position] != rune('"') {
					goto l91
				}
				position++
				add(ruleQuotedString, position92)
			}
			return true
		l91:
			position, tokenIndex = position91, tokenIndex91
			return false
		},
		/* 16 EscapedChar <- <('\\' .)> */
		func() bool {
			position101, tokenIndex101 := position, tokenIndex
			{
				position102 := position
				if buffer[position] != rune('\\') {
					goto l101
				}
				position++
				if !matchDot() {
					goto l101
				}
				add(ruleEscapedChar, position102)
			}
			return true
		l101:
			position, tokenIndex = position101, tokenIndex101
			return false
		},
		/* 17 ScalarOperand <- <(!('&' / '=') .)+> */
		func() bool {
			position103, tokenIndex103 := position, tokenIndex
			{
				position104 := position
				{
					position107, tokenIndex107 := position, tokenIndex
					{
						position108, tokenIndex108 := position, tokenIndex
						if buffer[position] != rune('&') {
							goto l109
						}
						position++
						goto l108
					l109:
						position, tokenIndex = position108, tokenIndex108
						if buffer[position] != rune('=') {
							goto l107
						}
						position++
					}
				l108:
					goto l103
				l107:
					position, tokenIndex = position107, tokenIndex107
				}
				if !matchDot() {
					goto l103
				}
			l105:
				{
					position106, tokenIndex106 := position, tokenIndex
					{
						position110, tokenIndex110 := position, tokenIndex
						{
							position111, tokenIndex111 := position, tokenIndex
							if buffer[position] != rune('&') {
								goto l112
							}
							position++
							goto l111
						l112:
							position, tokenIndex = position111, tokenIndex111
							if buffer[position] != rune('=') {
								goto l110
							}
							position++
						}
					l111:
						goto l106
					l110:
						position, tokenIndex = position110, tokenIndex110
					}
					if !matchDot() {
						goto l106
					}
					goto l105
				l106:
					position, tokenIndex = position106, tokenIndex106
				}
				add(ruleScalarOperand, position104)
			}
			return true
		l103:
			position, tokenIndex = position103, tokenIndex103
			return false
		},
		/* 18 Integer <- <[0-9]+> */
		func() bool {
			position113, tokenIndex113 := position, tokenIndex
			{
				position114 := position
				if c := buffer[position]; c < rune('0') || c > rune('9') {
					goto l113
				}
				position++
			l115:
				{
					position116, tokenIndex116 := position, tokenIndex
					if c := buffer[position]; c < rune('0') || c > rune('9') {
						goto l116
					}
					position++
					goto l115
				l116:
					position, tokenIndex = position116, tokenIndex116
				}
				add(ruleInteger, position114)
			}
			return true
		l113:
			position, tokenIndex = position113, tokenIndex113
			return false
		},
		/* 19 END <- <!.> */
		func() bool {
			position117, tokenIndex117 := position, tokenIndex
			{
				position118 := position
				{
					position119, tokenIndex119 := position, tokenIndex
					if !matchDot() {
						goto l119
					}
					goto l117
				l119:
					position, tokenIndex = position119, tokenIndex119
				}
				add(ruleEND, position118)
			}
			return true
		l117:
			position, tokenIndex = position117, tokenIndex117
			return false
		},
	}
	p.rules = _rules
	return nil
}
