package cue

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cuelang.org/go/cue"
	"github.com/lipence/config"
)

type cueIterator struct {
	rel *cue.Iterator
}

func (c *cueIterator) Next() bool {
	return c.rel.Next()
}

func (c *cueIterator) Value() config.Value {
	value := c.rel.Value()
	return &cueValue{real: value}
}

func (c *cueIterator) Label() string {
	return c.rel.Label()
}

func NewCueVal(value cue.Value) *cueValue {
	v := &cueValue{real: value}
	return v
}

type cueValue struct {
	real cue.Value
}

func (cfg *cueValue) Ref() string {
	return cfg.real.Pos().String()
}

func (cfg *cueValue) File() string {
	return cfg.real.Pos().Filename()
}

func (cfg *cueValue) Selector() []cue.Selector {
	return cfg.real.Path().Selectors()
}

func (cfg *cueValue) Kind() config.Kind {
	switch cfg.real.Kind() {
	case cue.BottomKind:
		return config.UndefinedKind
	case cue.NullKind:
		return config.NullKind
	case cue.BoolKind:
		return config.BoolKind
	case cue.StringKind:
		return config.StringKind
	case cue.BytesKind:
		return config.BytesKind
	case cue.StructKind:
		return config.StructKind
	case cue.ListKind:
		return config.ListKind
	case cue.FloatKind:
		return config.DecimalKind
	case cue.IntKind:
		return config.NumberKind
	}
	return config.UndefinedKind
}

func (cfg *cueValue) List() (config.Iterator, error) {
	var err error
	var itr cue.Iterator
	if itr, err = cfg.real.List(); err != nil {
		return nil, readErrors(err)
	}
	return &cueIterator{rel: &itr}, nil
}

func (cfg *cueValue) StringList() ([]string, error) {
	cfgItr, err := cfg.List()
	if err != nil {
		return nil, err
	}
	var result []string
	for cfgItr.Next() {
		var strItemVal string
		if strItemVal, err = cfgItr.Value().String(); err != nil {
			return nil, fmt.Errorf("%w, (field: %s)", err, cfgItr.Label())
		}
		result = append(result, strItemVal)
	}
	cfgItr = nil
	return result, nil
}

func (cfg *cueValue) Struct() (config.Iterator, error) {
	var err error
	var st *cue.Struct
	if st, err = cfg.real.Struct(); err != nil {
		return nil, readErrors(err)
	}
	return &cueIterator{rel: st.Fields()}, nil
}

func (cfg *cueValue) Lookup(path ...string) (config.Value, bool) {
	var cfgVal = cfg.real.LookupPath(cue.ParsePath(strings.Join(path, ".")))
	if !cfgVal.Exists() {
		return &cueValue{}, false
	}
	if cfgVal.Err() != nil {
		return &cueValue{}, false
	}
	return &cueValue{real: cfgVal}, true
}

func (cfg *cueValue) Marshal() (json []byte, err error) {
	if json, err = cfg.real.MarshalJSON(); err != nil {
		return nil, readErrors(err)
	}
	return json, nil
}

func (cfg *cueValue) String() (val string, err error) {
	if val, err = cfg.real.String(); err != nil {
		return "", readErrors(err)
	}
	return val, nil
}

func (cfg *cueValue) Bytes() (val []byte, err error) {
	if val, err = cfg.real.Bytes(); err != nil {
		return nil, readErrors(err)
	}
	return val, nil
}

func (cfg *cueValue) Bool() (val bool, err error) {
	if val, err = cfg.real.Bool(); err != nil {
		return val, readErrors(err)
	}
	return val, nil
}

func (cfg *cueValue) Float64() (val float64, err error) {
	if val, err = cfg.real.Float64(); err != nil {
		return val, readErrors(err)
	}
	return val, nil
}

func (cfg *cueValue) Uint64() (val uint64, err error) {
	if val, err = cfg.real.Uint64(); err != nil {
		return val, readErrors(err)
	}
	return val, nil
}

func (cfg *cueValue) Int64() (val int64, err error) {
	if val, err = cfg.real.Int64(); err != nil {
		return val, readErrors(err)
	}
	return val, nil
}

func (cfg *cueValue) Interface() (interface{}, error) {
	switch cfg.Kind() {
	case config.NullKind:
		return nil, nil
	case config.BoolKind:
		return cfg.Bool()
	case config.StringKind:
		return cfg.String()
	case config.BytesKind:
		return cfg.Bytes()
	case config.NumberKind:
		return cfg.Int64()
	case config.DecimalKind:
		return cfg.Float64()
	case config.StructKind:
		return cfg.Struct()
	case config.ListKind:
		return cfg.List()
	}
	return nil, fmt.Errorf("invalid source type")
}

type ctxBypass struct {
	context.Context
}

func (*ctxBypass) __bypass() {}

func (cfg *cueValue) Decode(target interface{}) error {
	return cfg.DecodeWithCtx(&ctxBypass{context.Background()}, target)
}

func (cfg *cueValue) DecodeWithCtx(ctx context.Context, target interface{}) (err error) {
	if decoder, ok := target.(config.Decoder); ok {
		if err = decoder.Decode(cfg); err != nil {
			return fmt.Errorf("%w (position: %s)", readErrors(err), cfg.Ref())
		}
		return nil
	}
	if decoder, ok := target.(config.CtxDecoder); ok {
		var _bypass *ctxBypass
		if _bypass, ok = ctx.(*ctxBypass); ok {
			ctx = _bypass.Context
		}
		if err = decoder.Decode(ctx, cfg); err != nil {
			return fmt.Errorf("%w (position: %s)", readErrors(err), cfg.Ref())
		}
		return nil
	}
	if decoder, ok := target.(config.CtxConfigDecoder); ok {
		var _bypass *ctxBypass
		if _bypass, ok = ctx.(*ctxBypass); ok {
			ctx = _bypass.Context
		}
		if err = decoder.DecodeConfig(ctx, cfg); err != nil {
			return fmt.Errorf("%w (position: %s)", readErrors(err), cfg.Ref())
		}
		return nil
	}
	switch _target := target.(type) {
	case *[]byte:
		var b []byte
		if b, err = cfg.Bytes(); err != nil {
			return readErrors(err)
		}
		*_target = b
		return nil
	case *string:
		var s string
		if s, err = cfg.String(); err != nil {
			return readErrors(err)
		}
		*_target = s
		return nil
	case *bool:
		if *_target, err = cfg.Bool(); err != nil {
			return readErrors(err)
		}
		return nil
	case *time.Duration:
		var _tar int64
		if _tar, err = cfg.Int64(); err != nil {
			return readErrors(err)
		}
		*_target = time.Duration(_tar)
		return nil
	case *int:
		var _tar int64
		if _tar, err = cfg.Int64(); err != nil {
			return readErrors(err)
		}
		*_target = int(_tar)
		return nil
	case *int8:
		var _tar int64
		if _tar, err = cfg.Int64(); err != nil {
			return readErrors(err)
		}
		*_target = int8(_tar)
		return nil
	case *int16:
		var _tar int64
		if _tar, err = cfg.Int64(); err != nil {
			return readErrors(err)
		}
		*_target = int16(_tar)
		return nil
	case *int32:
		var _tar int64
		if _tar, err = cfg.Int64(); err != nil {
			return readErrors(err)
		}
		*_target = int32(_tar)
		return nil
	case *int64:
		if *_target, err = cfg.Int64(); err != nil {
			return readErrors(err)
		}
		return nil
	case *uint:
		var _tar uint64
		if _tar, err = cfg.Uint64(); err != nil {
			return readErrors(err)
		}
		*_target = uint(_tar)
		return nil
	case *uint8:
		var _tar uint64
		if _tar, err = cfg.Uint64(); err != nil {
			return readErrors(err)
		}
		*_target = uint8(_tar)
		return nil
	case *uint16:
		var _tar uint64
		if _tar, err = cfg.Uint64(); err != nil {
			return readErrors(err)
		}
		*_target = uint16(_tar)
		return nil
	case *uint32:
		var _tar uint64
		if _tar, err = cfg.Uint64(); err != nil {
			return readErrors(err)
		}
		*_target = uint32(_tar)
		return nil
	case *uint64:
		if *_target, err = cfg.Uint64(); err != nil {
			return readErrors(err)
		}
		return nil
	}
	var _bytes []byte
	if _bytes, err = cfg.real.MarshalJSON(); err != nil {
		return readErrors(err)
	}
	if err = json.Unmarshal(_bytes, target); err != nil {
		return readErrors(err)
	}
	return nil
}
