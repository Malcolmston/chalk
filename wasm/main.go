//go:build js && wasm

// Command chalk (wasm) exposes the chalk library to JavaScript. Built with
// GOOS=js GOARCH=wasm it registers a `__mgo_chalk` object on the JS global with
// the library's portable, pure functions — ANSI styling, strip, and figlet
// rendering — so the very same Go implementation runs in Go and in the browser
// or Node. See chalk.mjs for the idiomatic JS wrapper.
package main

import (
	"reflect"
	"syscall/js"

	"github.com/malcolmston/chalk"
	"github.com/malcolmston/chalk/figlet"
)

func main() {
	// wasm has no TTY, so force truecolor output; otherwise chalk would emit
	// plain text. Callers can still strip codes with strip().
	chalk.SetLevel(chalk.LevelTrueColor)

	obj := js.Global().Get("Object").New()
	obj.Set("style", js.FuncOf(styleFn))
	obj.Set("strip", js.FuncOf(func(_ js.Value, a []js.Value) any {
		if len(a) == 0 {
			return ""
		}
		return chalk.Strip(a[0].String())
	}))
	obj.Set("visibleLength", js.FuncOf(func(_ js.Value, a []js.Value) any {
		if len(a) == 0 {
			return 0
		}
		return chalk.VisibleLength(a[0].String())
	}))
	obj.Set("figlet", js.FuncOf(func(_ js.Value, a []js.Value) any {
		if len(a) == 0 {
			return ""
		}
		return figlet.Render(a[0].String())
	}))
	obj.Set("figletFont", js.FuncOf(func(_ js.Value, a []js.Value) any {
		if len(a) < 2 {
			return ""
		}
		out, err := figlet.RenderFont(a[0].String(), a[1].String())
		if err != nil {
			return figlet.Render(a[1].String())
		}
		return out
	}))
	obj.Set("fonts", js.FuncOf(func(_ js.Value, _ []js.Value) any {
		names := figlet.Fonts()
		arr := js.Global().Get("Array").New(len(names))
		for i, n := range names {
			arr.SetIndex(i, n)
		}
		return arr
	}))
	js.Global().Set("__mgo_chalk", obj)

	select {} // keep the Go runtime alive so the exported funcs stay callable
}

// styleFn(text, styleNames[], opts?) applies a chain of named styles plus
// optional hex / bgHex / rgb from opts, and returns the ANSI string.
func styleFn(_ js.Value, args []js.Value) any {
	if len(args) == 0 {
		return ""
	}
	text := args[0].String()
	s := chalk.New()

	if len(args) > 1 && !args[1].IsUndefined() && !args[1].IsNull() {
		names := args[1]
		for i := 0; i < names.Length(); i++ {
			s = applyNamed(s, names.Index(i).String())
		}
	}
	if len(args) > 2 && args[2].Type() == js.TypeObject {
		opts := args[2]
		if v := opts.Get("hex"); v.Type() == js.TypeString {
			s = s.Hex(v.String())
		}
		if v := opts.Get("bgHex"); v.Type() == js.TypeString {
			s = s.BgHex(v.String())
		}
		if v := opts.Get("rgb"); v.Type() == js.TypeObject && v.Length() == 3 {
			s = s.RGB(v.Index(0).Int(), v.Index(1).Int(), v.Index(2).Int())
		}
		if v := opts.Get("bgRgb"); v.Type() == js.TypeObject && v.Length() == 3 {
			s = s.BgRGB(v.Index(0).Int(), v.Index(1).Int(), v.Index(2).Int())
		}
	}
	return s.Sprint(text)
}

// applyNamed calls the zero-arg *Style method matching a JS style name
// (e.g. "red" -> Red, "bgBlue" -> BgBlue, "brightRed" -> BrightRed). Unknown
// names are ignored so the JS side degrades gracefully.
func applyNamed(s *chalk.Style, name string) *chalk.Style {
	if name == "" {
		return s
	}
	method := string(upper(name[0])) + name[1:]
	m := reflect.ValueOf(s).MethodByName(method)
	if !m.IsValid() || m.Type().NumIn() != 0 || m.Type().NumOut() != 1 {
		return s
	}
	if ns, ok := m.Call(nil)[0].Interface().(*chalk.Style); ok {
		return ns
	}
	return s
}

func upper(b byte) byte {
	if b >= 'a' && b <= 'z' {
		return b - 32
	}
	return b
}
