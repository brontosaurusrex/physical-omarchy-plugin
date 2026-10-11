// Package qtbridge adapts the original Go renderer to a native Qt command stream.
// It does not embed a JavaScript runtime, browser, DOM, or network client.
package qtbridge

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Type int

const (
	TypeUndefined Type = iota
	TypeNull
	TypeBoolean
	TypeNumber
	TypeString
	TypeObject
	TypeFunction
)

type node struct {
	kind     string
	props    map[string]Value
	ops      [][]any
	id       int
	revision int
}
type Value struct{ raw any }
type Func struct{ Value }
type callback func(Value, []Value) any

var global = object("global")
var document = object("document")
var mainCanvas Value
var canvases []*node
var resources fs.FS
var storage = map[string]string{}
var statePath string
var lastStorageError error
var clockStart = time.Now()
var clockMS float64
var sentRevisions = map[int]int{}

func object(kind string) Value { return Value{&node{kind: kind, props: map[string]Value{}}} }
func Undefined() Value         { return Value{} }
func Null() Value              { return Value{TypeNull} }
func Global() Value            { return global }
func ValueOf(v any) Value {
	switch x := v.(type) {
	case Value:
		return x
	case Func:
		return x.Value
	case map[string]any:
		o := object("object")
		for k, value := range x {
			o.Set(k, value)
		}
		return o
	default:
		return Value{v}
	}
}
func FuncOf(fn func(Value, []Value) interface{}) Func { return Func{Value{callback(fn)}} }
func (f Func) Release()                               {}
func (v Value) IsUndefined() bool                     { return v.raw == nil }
func (v Value) IsNull() bool                          { t, ok := v.raw.(Type); return ok && t == TypeNull }
func (v Value) Type() Type {
	if v.IsUndefined() {
		return TypeUndefined
	}
	if v.IsNull() {
		return TypeNull
	}
	switch v.raw.(type) {
	case bool:
		return TypeBoolean
	case string:
		return TypeString
	case int, int64, float32, float64:
		return TypeNumber
	case callback:
		return TypeFunction
	default:
		return TypeObject
	}
}
func (v Value) String() string {
	if v.IsUndefined() {
		return ""
	}
	return fmt.Sprint(v.raw)
}
func (v Value) Float() float64 {
	switch x := v.raw.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case int:
		return float64(x)
	case int64:
		return float64(x)
	default:
		f, _ := strconv.ParseFloat(v.String(), 64)
		return f
	}
}
func (v Value) Int() int   { return int(v.Float()) }
func (v Value) Bool() bool { b, _ := v.raw.(bool); return b }
func (v Value) Equal(other Value) bool {
	a, aok := v.raw.(*node)
	b, bok := other.raw.(*node)
	if aok || bok {
		return a == b
	}
	return v.String() == other.String() && v.Type() == other.Type()
}
func (v Value) Length() int { a, _ := v.raw.([]Value); return len(a) }
func (v Value) Index(index int) Value {
	a, _ := v.raw.([]Value)
	if index >= 0 && index < len(a) {
		return a[index]
	}
	return Undefined()
}
func (v Value) Get(key string) Value {
	n, ok := v.raw.(*node)
	if !ok {
		return Undefined()
	}
	if result, found := n.props[key]; found {
		return result
	}
	if n.kind == "global" {
		switch key {
		case "window":
			return global
		case "document":
			return document
		case "performance", "console", "navigator", "Date", "localStorage", "XMLHttpRequest", "Object", "Array", "Uint8Array":
			return object(key)
		}
	}
	if key == "style" || key == "classList" || key == "body" || key == "head" || key == "documentElement" {
		n.props[key] = object(key)
		return n.props[key]
	}
	if n.kind == "document" && key == "hidden" {
		return ValueOf(false)
	}
	if n.kind == "navigator" && key == "userAgent" {
		return ValueOf("Physical Qt native")
	}
	return Undefined()
}
func raw(v any) any {
	switch x := v.(type) {
	case Value:
		return x.raw
	case Func:
		return nil
	default:
		return v
	}
}
func (v Value) Set(key string, value any) {
	n, ok := v.raw.(*node)
	if !ok {
		return
	}
	n.props[key] = ValueOf(value)
	if n.kind == "context" {
		n.ops = append(n.ops, []any{"set", key, raw(value)})
		n.revision++
	}
}
func (v Value) New(args ...any) Value {
	n, ok := v.raw.(*node)
	if !ok {
		return object("object")
	}
	if n.kind == "XMLHttpRequest" {
		return object("xhr")
	}
	if n.kind == "Array" {
		return Value{[]Value{}}
	}
	return object(n.kind)
}
func (v Value) Invoke(args ...any) Value {
	fn, ok := v.raw.(callback)
	if !ok {
		return Undefined()
	}
	a := make([]Value, len(args))
	for i, x := range args {
		a[i] = ValueOf(x)
	}
	return ValueOf(fn(Undefined(), a))
}
func canvas() Value {
	o := object("canvas")
	c := object("context")
	n := c.raw.(*node)
	n.id = len(canvases)
	canvases = append(canvases, n)
	o.Set("context", c)
	return o
}
func (v Value) Call(method string, args ...any) Value {
	n, ok := v.raw.(*node)
	if !ok {
		return Undefined()
	}
	switch n.kind {
	case "context":
		if method == "measureText" {
			size := 16.0
			fmt.Sscanf(n.props["font"].String(), "%fpx", &size)
			return ValueOf(map[string]any{"width": float64(len([]rune(fmt.Sprint(args[0])))) * size * 0.6})
		}
		op := []any{method}
		for _, a := range args {
			op = append(op, raw(a))
		}
		if method == "drawImage" {
			other := ValueOf(args[0]).Get("context").raw.(*node)
			op = []any{"cache", other.id}
		}
		if method == "clearRect" {
			n.ops = nil
		}
		n.ops = append(n.ops, op)
		n.revision++
		return Undefined()
	case "canvas":
		if method == "getContext" {
			return v.Get("context")
		}
		if method == "getBoundingClientRect" {
			return ValueOf(map[string]any{"left": 0, "top": 0, "width": v.Get("width").Float(), "height": v.Get("height").Float()})
		}
	case "document":
		if method == "getElementById" {
			if fmt.Sprint(args[0]) == "gameCanvas" {
				return mainCanvas
			}
			return Null()
		}
		if method == "createElement" {
			if fmt.Sprint(args[0]) == "canvas" {
				return canvas()
			}
			return object("element")
		}
	case "performance":
		if method == "now" {
			return ValueOf(clockMS)
		}
	case "Date":
		if method == "now" {
			return ValueOf(float64(time.Now().UnixMilli()))
		}
	case "console":
		fmt.Fprintln(os.Stderr, args...)
	case "localStorage":
		if method == "getItem" {
			if x, found := storage[fmt.Sprint(args[0])]; found {
				return ValueOf(x)
			}
			return Null()
		}
		if method == "setItem" {
			storage[fmt.Sprint(args[0])] = fmt.Sprint(args[1])
			data, _ := json.Marshal(storage)
			if err := os.MkdirAll(filepath.Dir(statePath), 0700); err != nil {
				lastStorageError = err
				return Undefined()
			}
			tmp := statePath + ".tmp"
			if err := os.WriteFile(tmp, data, 0600); err != nil {
				lastStorageError = err
				return Undefined()
			}
			lastStorageError = os.Rename(tmp, statePath)
		}
	case "xhr":
		if method == "open" {
			n.props["url"] = ValueOf(args[1])
		}
		if method == "send" {
			path := strings.Split(n.props["url"].String(), "?")[0]
			data, err := fs.ReadFile(resources, path)
			if err != nil {
				n.props["status"] = ValueOf(404)
			} else {
				n.props["status"] = ValueOf(200)
				n.props["responseText"] = ValueOf(string(data))
			}
		}
	}
	return Undefined()
}
func CopyBytesToJS(v Value, data []byte) int { return len(data) }

func Initialize(resourceFS fs.FS, path string) Value {
	resources = resourceFS
	statePath = path
	if data, err := os.ReadFile(path); err == nil {
		if err = json.Unmarshal(data, &storage); err != nil {
			lastStorageError = fmt.Errorf("saved progression: %w", err)
		}
	}
	mainCanvas = canvas()
	mainCanvas.Set("width", 1800)
	mainCanvas.Set("height", 900)
	return mainCanvas
}
func SetClock(ms float64) { clockMS = ms }
func Now() float64        { return float64(time.Since(clockStart).Microseconds()) / 1000 }
func BeginFrame() {
	if len(canvases) > 0 {
		canvases[0].ops = nil
	}
}
func Frame() ([][]any, map[int][][]any) {
	caches := map[int][][]any{}
	for _, c := range canvases[1:] {
		if sentRevisions[c.id] != c.revision {
			caches[c.id] = c.ops
			sentRevisions[c.id] = c.revision
		}
	}
	return canvases[0].ops, caches
}
func StorageError() string {
	if lastStorageError != nil {
		return lastStorageError.Error()
	}
	return ""
}
