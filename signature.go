package main

import (
	"github.com/Bananenpro/embe/generator"
	"github.com/Bananenpro/embe/parser"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func textDocumentSignatureHelp(context *glsp.Context, params *protocol.SignatureHelpParams) (*protocol.SignatureHelp, error) {
	document, ok := getDocument(params.TextDocument.URI)
	if !ok {
		return nil, nil
	}

	pos := params.Position
	startPos := pos
	startPos.Character = 0
	endPos := startPos.EndOfLineIn(document.content)

	line := document.content[startPos.IndexIn(document.content):endPos.IndexIn(document.content)]

	parenIndices := make([]int, 0, 5)
	for i, c := range line {
		if i >= int(pos.Character) {
			break
		}
		if c == '(' {
			parenIndices = append(parenIndices, i)
		} else if c == ')' && len(parenIndices) > 0 {
			parenIndices = parenIndices[:len(parenIndices)-1]
		}
	}
	if len(parenIndices) == 0 {
		return nil, nil
	}

	var identifier parser.Token
	var identifierIndex int
	for i, t := range document.tokens {
		if i == 0 {
			continue
		}
		if t.Line == int(params.Position.Line) && t.Column == parenIndices[len(parenIndices)-1] {
			identifier = document.tokens[i-1]
			identifierIndex = i
			break
		}
	}

	if (identifier == parser.Token{}) {
		return nil, nil
	}

	var signatures []generator.Signature
	if f, ok := generator.ExprFuncCalls[identifier.Lexeme]; ok {
		signatures = f.Signatures
	} else if f, ok := generator.FuncCalls[identifier.Lexeme]; ok {
		signatures = f.Signatures
	} else if f, ok := document.functions[identifier.Lexeme]; ok {
		params := make([]generator.Param, 0)
		for _, p := range f.Params {
			params = append(params, generator.Param{
				Name: p.Name.Lexeme,
				Type: p.Type.DataType,
			})
		}
		signatures = []generator.Signature{
			{
				FuncName: f.Name.Lexeme,
				Params:   params,
			},
		}
	} else {
		return nil, nil
	}

	parens := 1
	paramCount := 1
	var paramIndex uint32
	for i := identifierIndex + 2; i < len(document.tokens) && parens > 0; i++ {
		token := document.tokens[i]
		switch token.Type {
		case parser.TkOpenParen:
			parens++
		case parser.TkCloseParen:
			parens--
		case parser.TkComma:
			paramCount++
			if token.Line == int(pos.Line) && token.Column <= int(pos.Character) {
				paramIndex++
			}
		}
	}
	if identifierIndex+2 < len(document.tokens) && document.tokens[identifierIndex+2].Type == parser.TkCloseParen {
		paramCount = 0
	}

	var activeSignature uint32
	for i, s := range signatures {
		if len(s.Params) == paramCount {
			activeSignature = uint32(i)
			break
		}
	}

	signatureInformation := make([]protocol.SignatureInformation, len(signatures))
	for i, s := range signatures {
		parameters := make([]protocol.ParameterInformation, len(s.Params))
		for j, p := range s.Params {
			parameters[j] = protocol.ParameterInformation{
				Label: p.Name + ": " + string(p.Type),
			}
		}
		signatureInformation[i] = protocol.SignatureInformation{
			Label:      s.String(),
			Parameters: parameters,
		}
	}

	return &protocol.SignatureHelp{
		Signatures:      signatureInformation,
		ActiveSignature: &activeSignature,
		ActiveParameter: &paramIndex,
	}, nil
}
