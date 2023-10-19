package clog

import "log/slog"

//////////////////////////////////////////////////

type ParamGroup map[string]any

func (pg ParamGroup) LogValue() (value slog.Value) {
	if len(pg) == 0 {
		return slog.GroupValue()
	}

	var attrs []slog.Attr

	for key, any := range pg {
		var attr slog.Attr

		switch lv := any.(type) {
		case slog.LogValuer:
			attr = slog.Any(key, lv.LogValue())

		default:
			attr = slog.Any(key, lv)
		}

		attrs = append(attrs, attr)
	}

	return slog.GroupValue(attrs...)
}

func (pg ParamGroup) Count() int {
	return len(pg)
}

func (pg ParamGroup) Get(key string) (v any, exists bool) {
	if pg == nil {
		return
	}

	v, exists = pg[key]
	return
}

func (pg *ParamGroup) Set(key string, v any) {
	if pg == nil {
		return
	}

	if *pg == nil {
		*pg = make(ParamGroup)
	}

	(*pg)[key] = v
}

func (pg ParamGroup) Delete(key string) {
	if pg == nil {
		return
	}

	delete(pg, key)
}

func (pg ParamGroup) Exists(key string) bool {
	if pg == nil {
		return false
	}

	_, exists := pg[key]
	return exists
}

func (pg ParamGroup) Clone() ParamGroup {
	if pg == nil {
		return nil
	}

	c := ParamGroup{}

	for key, any := range pg {
		if g, ok := any.(ParamGroup); ok {
			c[key] = g.Clone()
		} else {
			c[key] = any
		}
	}

	return c
}

type Params struct {
	Message string
	Level   slog.Level
	Err     error

	ForceLevel bool
	ExcludeErr bool

	Values ParamGroup
}

func (lp *Params) LogValue() slog.Value {
	if lp == nil {
		return slog.GroupValue()
	}

	return lp.Values.LogValue()
}

func (lp *Params) Serialize() (attrs []slog.Attr, level slog.Level) {
	v := lp.Values.LogValue()

	if v.Kind() == slog.KindGroup {
		attrs = v.Group()
	} else {
		attrs = []slog.Attr{}
	}

	level = lp.Level
	if !lp.ExcludeErr && lp.Err != nil {
		if !lp.ForceLevel {
			level = slog.LevelError
		}

		attrs = append(attrs, slog.Any("err", lp.Err))
	}

	return
}

func (lp *Params) Get(key string) (v any, exists bool) {
	if lp == nil {
		return
	}

	return lp.Values.Get(key)
}

func (lp *Params) Set(key string, v any) {
	if lp == nil {
		return
	}

	lp.Values.Set(key, v)
}

func (lp *Params) Delete(key string) {
	if lp == nil {
		return
	}

	lp.Values.Delete(key)
}

func (lp *Params) Exists(key string) bool {
	if lp == nil {
		return false
	}

	return lp.Values.Exists(key)
}

func (lp *Params) Clone() Params {
	if lp == nil {
		return Params{}
	}

	c := Params{
		Message: lp.Message,
		Level:   lp.Level,
		Err:     lp.Err,

		ForceLevel: lp.ForceLevel,
		ExcludeErr: lp.ExcludeErr,

		Values: lp.Values.Clone(),
	}
	return c
}
