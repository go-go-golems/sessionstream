package sessionstream

import (
	"strconv"

	"github.com/dop251/goja"
	ss "github.com/go-go-golems/sessionstream/pkg/sessionstream"
)

func (m *moduleRuntime) wrapTimelineView(view ss.TimelineView) goja.Value {
	obj := m.vm.NewObject()
	m.mustSet(obj, "ordinal", func() string {
		if view == nil {
			return "0"
		}
		return strconv.FormatUint(view.Ordinal(), 10)
	})
	m.mustSet(obj, "get", func(kind, id string) goja.Value {
		if view == nil {
			return goja.Null()
		}
		ent, ok := view.Get(kind, id)
		if !ok {
			return goja.Null()
		}
		converted, err := m.timelineEntityToJS(ent)
		if err != nil {
			panic(m.vm.NewGoError(err))
		}
		return m.vm.ToValue(converted)
	})
	m.mustSet(obj, "list", func(kind string) goja.Value {
		if view == nil {
			return m.vm.ToValue([]any{})
		}
		entities := view.List(kind)
		out := make([]map[string]any, 0, len(entities))
		for _, ent := range entities {
			converted, err := m.timelineEntityToJS(ent)
			if err != nil {
				panic(m.vm.NewGoError(err))
			}
			out = append(out, converted)
		}
		return m.vm.ToValue(out)
	})
	return obj
}
