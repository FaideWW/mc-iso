package block

import (
	"encoding/json"
	"errors"
)

type BlockStateVariantModel struct {
	Model  string `json:"model"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Uv     bool   `json:"uv"`
	Weight int    `json:"weight"`
}

type BlockStateVariant []BlockStateVariantModel

func (v *BlockStateVariant) UnmarshalJSON(data []byte) error {
	// Since blockstate variants can either be a single model or a list of
	// models, we need to handle unmarshalling specially in order to coerce
	// things into a uniform type. For single models, we simply create a slice of
	// length 1 for that model.

	// Seek forward until we see either a square bracket (indicates an array) or
	// a curly brace (indicates an object). Discard whitespace, and error on
	// anything else.
	for _, b := range data {
		switch b {
		case ' ', '\n', '\r', '\t':
			continue
		case '[':
			return json.Unmarshal(data, (*[]BlockStateVariantModel)(v))
		case '{':
			var model BlockStateVariantModel
			if err := json.Unmarshal(data, &model); err != nil {
				return err
			}
			*v = []BlockStateVariantModel{model}
			return nil
		default:
			return errors.New("blockstate variant must be an array or an object")
		}
	}
	return errors.New("reached end of json without receiving a blockstate variant")
}

const (
	CondOr = iota
	CondAnd
	CondConds
)

type MultipartCondition struct {
	Type  int
	Or    []MultipartCondition
	And   []MultipartCondition
	Conds map[string]string
}

func (c *MultipartCondition) UnmarshalJSON(data []byte) error {
	rawConds := map[string]json.RawMessage{}
	err := json.Unmarshal(data, &rawConds)
	if err != nil {
		return err
	}

	var optsSet = 0
	if orCond, ok := rawConds["OR"]; ok {
		if err = json.Unmarshal(orCond, &c.Or); err != nil {
			return err
		}
		c.Type = CondOr
		optsSet++
		delete(rawConds, "OR")
	}

	if andCond, ok := rawConds["AND"]; ok {
		if err = json.Unmarshal(andCond, &c.And); err != nil {
			return err
		}
		c.Type = CondAnd
		optsSet++
		delete(rawConds, "AND")
	}

	c.Conds = map[string]string{}
	for k, v := range rawConds {
		var str string
		err := json.Unmarshal(v, &str)
		if err != nil {
			return err
		}
		c.Conds[k] = str
	}
	if len(c.Conds) > 0 {
		c.Type = CondConds
		optsSet++
	}

	if optsSet > 1 {
		return errors.New("when condition uses more than one of: OR, AND, raw predicates")
	}

	return nil
}

type BlockStateMultipartCase struct {
	When  MultipartCondition `json:"when"`
	Apply BlockStateVariant  `json:"apply"`
}

type BlockState struct {
	Variants  map[string]BlockStateVariant `json:"variants"`
	Multipart []BlockStateMultipartCase    `json:"multipart"`
}
