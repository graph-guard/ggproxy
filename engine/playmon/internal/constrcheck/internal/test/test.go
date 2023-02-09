package test

import (
	"bytes"
	"embed"
	"fmt"
	"path/filepath"

	"github.com/graph-guard/ggproxy/gqlparse"
	"github.com/graph-guard/gqlscan"
	"gopkg.in/yaml.v3"
)

type T struct {
	Schema   string           `yaml:"schema"`
	Template string           `yaml:"template"`
	Inputs   map[string]Input `yaml:"inputs"`
}

type Input struct {
	Match           *bool  `yaml:"match"`
	MatchSchemaless *bool  `yaml:"match-schemaless"`
	Tokens          Tokens `yaml:"tokens"`
}

type Tokens []gqlparse.Token

func (t *Tokens) UnmarshalYAML(value *yaml.Node) error {
	for _, pair := range value.Content {
		if len(pair.Content) < 1 {
			return fmt.Errorf("invalid tokens dictionary")
		}
		tp, err := parseTokenType(pair.Content[0].Value)
		if err != nil {
			return err
		}
		*t = append(*t, gqlparse.Token{ID: tp, Value: []byte(pair.Content[1].Value)})
	}
	return nil
}

func Parse(fs embed.FS, fileName string) (ts T, err error) {
	f, err := fs.ReadFile(filepath.Join("tests", fileName))
	if err != nil {
		return ts, fmt.Errorf("reading YAML test file: %w", err)
	}

	d := yaml.NewDecoder(bytes.NewReader(f))
	d.KnownFields(true)
	if err := d.Decode(&ts); err != nil {
		return ts, fmt.Errorf("parsing YAML test definition: %w", err)
	}

	// Make sure each input has only one value
	for path, i := range ts.Inputs {
		if len(i.Tokens) < 1 {
			return ts, fmt.Errorf("%q: missing input value", path)
		}
	}

	return ts, nil
}

func parseTokenType(name string) (gqlscan.Token, error) {
	switch name {
	case "DefQry":
		return gqlscan.TokenDefQry, nil
	case "DefMut":
		return gqlscan.TokenDefMut, nil
	case "DefSub":
		return gqlscan.TokenDefSub, nil
	case "DefFrag":
		return gqlscan.TokenDefFrag, nil
	case "OprName":
		return gqlscan.TokenOprName, nil
	case "DirName":
		return gqlscan.TokenDirName, nil
	case "VarList":
		return gqlscan.TokenVarList, nil
	case "VarListEnd":
		return gqlscan.TokenVarListEnd, nil
	case "ArgList":
		return gqlscan.TokenArgList, nil
	case "ArgListEnd":
		return gqlscan.TokenArgListEnd, nil
	case "Set":
		return gqlscan.TokenSet, nil
	case "SetEnd":
		return gqlscan.TokenSetEnd, nil
	case "FragTypeCond":
		return gqlscan.TokenFragTypeCond, nil
	case "FragName":
		return gqlscan.TokenFragName, nil
	case "FragInline":
		return gqlscan.TokenFragInline, nil
	case "NamedSpread":
		return gqlscan.TokenNamedSpread, nil
	case "FieldAlias":
		return gqlscan.TokenFieldAlias, nil
	case "Field":
		return gqlscan.TokenField, nil
	case "ArgName":
		return gqlscan.TokenArgName, nil
	case "EnumVal":
		return gqlscan.TokenEnumVal, nil
	case "Arr":
		return gqlscan.TokenArr, nil
	case "ArrEnd":
		return gqlscan.TokenArrEnd, nil
	case "Str":
		return gqlscan.TokenStr, nil
	case "StrBlock":
		return gqlscan.TokenStrBlock, nil
	case "Int":
		return gqlscan.TokenInt, nil
	case "Float":
		return gqlscan.TokenFloat, nil
	case "True":
		return gqlscan.TokenTrue, nil
	case "False":
		return gqlscan.TokenFalse, nil
	case "Null":
		return gqlscan.TokenNull, nil
	case "VarName":
		return gqlscan.TokenVarName, nil
	case "VarTypeName":
		return gqlscan.TokenVarTypeName, nil
	case "VarTypeArr":
		return gqlscan.TokenVarTypeArr, nil
	case "VarTypeArrEnd":
		return gqlscan.TokenVarTypeArrEnd, nil
	case "VarTypeNotNull":
		return gqlscan.TokenVarTypeNotNull, nil
	case "VarRef":
		return gqlscan.TokenVarRef, nil
	case "Obj":
		return gqlscan.TokenObj, nil
	case "ObjEnd":
		return gqlscan.TokenObjEnd, nil
	case "ObjField":
		return gqlscan.TokenObjField, nil
	}
	return 0, fmt.Errorf("unknown token type: %q", name)
}
