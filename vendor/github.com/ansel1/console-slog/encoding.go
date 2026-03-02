package console

import (
	"bytes"
	"fmt"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/ansel1/console-slog/internal"
)

var encoderPool = &sync.Pool{
	New: func() any {
		e := new(encoder)
		e.groups = make([]string, 0, 10)
		e.buf = make(buffer, 0, 1024)
		e.attrBuf = make(buffer, 0, 1024)
		e.multilineAttrBuf = make(buffer, 0, 1024)
		e.headerAttrs = make([]slog.Attr, 0, 5)
		return e
	},
}

type encoder struct {
	h                              *Handler
	buf, attrBuf, multilineAttrBuf buffer
	groups                         []string
	headerAttrs                    []slog.Attr
}

func newEncoder(h *Handler) *encoder {
	e := encoderPool.Get().(*encoder)
	e.h = h
	if h.opts.ReplaceAttr != nil {
		e.groups = append(e.groups, h.groups...)
	}
	e.headerAttrs = slices.Grow(e.headerAttrs, len(h.headerFields))[:len(h.headerFields)]
	clear(e.headerAttrs)
	return e
}

func (e *encoder) free() {
	if e == nil {
		return
	}
	e.h = nil
	e.buf.Reset()
	e.attrBuf.Reset()
	e.multilineAttrBuf.Reset()
	e.groups = e.groups[:0]
	e.headerAttrs = e.headerAttrs[:0]
	encoderPool.Put(e)
}

func (e *encoder) encodeTimestamp(tt time.Time) {
	if tt.IsZero() {
		// elide, and skip ReplaceAttr
		return
	}

	if e.h.opts.ReplaceAttr != nil {
		attr := e.h.opts.ReplaceAttr(nil, slog.Time(slog.TimeKey, tt))
		attr.Value = attr.Value.Resolve()

		if attr.Value.Equal(slog.Value{}) {
			// elide
			return
		}

		if attr.Value.Kind() != slog.KindTime {
			// handle all non-time values by printing them like
			// an attr value
			e.writeColoredValue(&e.buf, attr.Value, e.h.opts.Theme.Timestamp)
			return
		}

		// most common case
		tt = attr.Value.Time()
		if tt.IsZero() {
			// elide
			return
		}
	}

	e.withColor(&e.buf, e.h.opts.Theme.Timestamp, func() {
		e.buf.AppendTime(tt, e.h.opts.TimeFormat)
	})
}

func (e *encoder) encodeMessage(level slog.Level, msg string) {
	style := e.h.opts.Theme.Message
	if level < slog.LevelInfo {
		style = e.h.opts.Theme.MessageDebug
	}

	if e.h.opts.ReplaceAttr != nil {
		attr := e.h.opts.ReplaceAttr(nil, slog.String(slog.MessageKey, msg))
		attr.Value = attr.Value.Resolve()
		if attr.Value.Equal(slog.Value{}) {
			// elide
			return
		}

		e.writeColoredValue(&e.buf, attr.Value, style)
		return
	}

	e.writeColoredString(&e.buf, strings.TrimSpace(msg), style)
}

func (e *encoder) encodeHeader(a slog.Attr, width int, rightAlign bool) {
	if a.Value.Equal(slog.Value{}) {
		// just pad as needed
		if width > 0 {
			e.buf.Pad(width, ' ')
		}
		return
	}

	e.withColor(&e.buf, e.h.opts.Theme.Header, func() {
		l := len(e.buf)
		e.writeValue(&e.buf, a.Value)
		if width <= 0 {
			return
		}
		// truncate or pad to required width
		remainingWidth := l + width - len(e.buf)
		if remainingWidth < 0 {
			// truncate
			e.buf = e.buf[:l+width]
		} else if remainingWidth > 0 {
			if rightAlign {
				// For right alignment, shift the text right in-place:
				// 1. Get the text length
				textLen := len(e.buf) - l
				// 2. Add padding to reach final width
				e.buf.Pad(remainingWidth, ' ')
				// 3. Move the text to the right by copying from end to start
				for i := 0; i < textLen; i++ {
					e.buf[len(e.buf)-1-i] = e.buf[l+textLen-1-i]
				}
				// 4. Fill the left side with spaces
				for i := 0; i < remainingWidth; i++ {
					e.buf[l+i] = ' '
				}
			} else {
				// Left align - just pad with spaces
				e.buf.Pad(remainingWidth, ' ')
			}
		}
	})
}

func (e *encoder) encodeLevel(l slog.Level, abbreviated bool) {
	var val slog.Value
	var writeVal bool

	if e.h.opts.ReplaceAttr != nil {
		attr := e.h.opts.ReplaceAttr(nil, slog.Any(slog.LevelKey, l))
		attr.Value = attr.Value.Resolve()

		if attr.Value.Equal(slog.Value{}) {
			// elide
			return
		}

		val = attr.Value
		writeVal = true

		if val.Kind() == slog.KindAny {
			if ll, ok := val.Any().(slog.Level); ok {
				// generally, we'll write the returned value, except in one
				// case: when the resolved value is itself a slog.Level
				l = ll
				writeVal = false
			}
		}
	}

	var style ANSIMod
	var str string
	var delta int
	switch {
	case l >= slog.LevelError:
		style = e.h.opts.Theme.LevelError
		str = "ERR"
		if !abbreviated {
			str = "ERROR"
		}
		delta = int(l - slog.LevelError)
	case l >= slog.LevelWarn:
		style = e.h.opts.Theme.LevelWarn
		str = "WRN"
		if !abbreviated {
			str = "WARN"
		}
		delta = int(l - slog.LevelWarn)
	case l >= slog.LevelInfo:
		style = e.h.opts.Theme.LevelInfo
		str = "INF"
		if !abbreviated {
			str = "INFO"
		}
		delta = int(l - slog.LevelInfo)
	case l >= slog.LevelDebug:
		style = e.h.opts.Theme.LevelDebug
		str = "DBG"
		if !abbreviated {
			str = "DEBUG"
		}
		delta = int(l - slog.LevelDebug)
	default:
		style = e.h.opts.Theme.LevelDebug
		str = "DBG"
		if !abbreviated {
			str = "DEBUG"
		}
		delta = int(l - slog.LevelDebug)
	}
	if writeVal {
		e.writeColoredValue(&e.buf, val, style)
	} else {
		if delta != 0 {
			str = fmt.Sprintf("%s%+d", str, delta)
		}
		e.writeColoredString(&e.buf, str, style)
	}
}

func (e *encoder) encodeSource(src slog.Source) {
	if src.File == "" && src.Line == 0 {
		// elide empty source
		return
	}

	v := slog.AnyValue(&src)

	if e.h.opts.ReplaceAttr != nil {
		attr := e.h.opts.ReplaceAttr(nil, slog.Attr{Key: slog.SourceKey, Value: v})
		attr.Value = attr.Value.Resolve()

		if attr.Value.Equal(slog.Value{}) {
			// elide
			return
		}
		v = attr.Value
	}
	// Use source style for the value
	e.writeColoredValue(&e.buf, v, e.h.opts.Theme.Source)
}

func (e *encoder) encodeAttr(groupPrefix string, a slog.Attr) {

	a.Value = a.Value.Resolve()
	if a.Value.Kind() != slog.KindGroup && e.h.opts.ReplaceAttr != nil {
		a = e.h.opts.ReplaceAttr(e.groups, a)
		a.Value = a.Value.Resolve()
	}
	// Elide empty Attrs.
	if a.Equal(slog.Attr{}) {
		return
	}

	value := a.Value

	if value.Kind() == slog.KindGroup {
		subgroup := a.Key
		if groupPrefix != "" {
			subgroup = groupPrefix + "." + a.Key
		}
		if e.h.opts.ReplaceAttr != nil {
			e.groups = append(e.groups, a.Key)
		}
		for _, attr := range value.Group() {
			e.encodeAttr(subgroup, attr)
		}
		if e.h.opts.ReplaceAttr != nil {
			e.groups = e.groups[:len(e.groups)-1]
		}
		return
	}

	for i, f := range e.h.headerFields {
		if f.key == a.Key && f.groupPrefix == groupPrefix {
			e.headerAttrs[i] = a
			return
		}
	}

	offset := len(e.attrBuf)
	valOffset := e.writeAttr(a, groupPrefix)

	// check if the last attr written has newlines in it
	// if so, move it to the trailerBuf
	if bytes.IndexByte(e.attrBuf[offset:], '\n') >= 0 {
		if internal.FeatureFlagNewMultilineAttrs {
			val := e.attrBuf[valOffset:]
			e.writeMultilineAttr(a.Key, groupPrefix, val)
		} else {
			e.multilineAttrBuf.Append(e.attrBuf[offset:])
		}

		// rewind the middle buffer
		e.attrBuf = e.attrBuf[:offset]
	}
}

func (e *encoder) withColor(b *buffer, c ANSIMod, f func()) {
	if c == "" || e.h.opts.NoColor {
		f()
		return
	}
	b.AppendString(string(c))
	f()
	b.AppendString(string(ResetMod))
}

func (e *encoder) writeColoredString(w *buffer, s string, c ANSIMod) {
	e.withColor(w, c, func() {
		w.AppendString(s)
	})
}

// writeAttr encodes the attr to the attrBuf.  The group will be prepended
// to the key, joined with a '.'
//
// returns the offset where the value starts, which may be used by the
// caller to split the key and value
func (e *encoder) writeAttr(a slog.Attr, group string) int {
	value := a.Value

	e.attrBuf.AppendByte(' ')
	e.withColor(&e.attrBuf, e.h.opts.Theme.AttrKey, func() {
		if group != "" {
			e.attrBuf.AppendString(group)
			e.attrBuf.AppendByte('.')
		}
		e.attrBuf.AppendString(a.Key)
		e.attrBuf.AppendByte('=')
	})

	style := e.h.opts.Theme.AttrValue
	if value.Kind() == slog.KindAny {
		if _, ok := value.Any().(error); ok {
			style = e.h.opts.Theme.AttrValueError
		}
	}
	valOffset := len(e.attrBuf)
	e.writeColoredValue(&e.attrBuf, value, style)
	return valOffset
}

func (e *encoder) writeMultilineAttr(key, group string, value []byte) {
	e.multilineAttrBuf.AppendByte('\n')
	e.withColor(&e.multilineAttrBuf, e.h.opts.Theme.AttrKey, func() {
		e.multilineAttrBuf.AppendString("=== ")
		if group != "" {
			e.multilineAttrBuf.AppendString(group)
			e.multilineAttrBuf.AppendByte('.')
		}
		e.multilineAttrBuf.AppendString(key)
		e.multilineAttrBuf.AppendString(" ===\n")
	})
	e.multilineAttrBuf.Append(value)
}

func (e *encoder) writeValue(buf *buffer, value slog.Value) {
	switch value.Kind() {
	case slog.KindInt64:
		buf.AppendInt(value.Int64())
	case slog.KindBool:
		buf.AppendBool(value.Bool())
	case slog.KindFloat64:
		buf.AppendFloat(value.Float64())
	case slog.KindTime:
		buf.AppendTime(value.Time(), e.h.opts.TimeFormat)
	case slog.KindUint64:
		buf.AppendUint(value.Uint64())
	case slog.KindDuration:
		buf.AppendDuration(value.Duration())
	case slog.KindAny:
		switch v := value.Any().(type) {
		case error:
			if _, ok := v.(fmt.Formatter); ok {
				fmt.Fprintf(buf, "%+v", v)
			} else {
				buf.AppendString(v.Error())
			}
			return
		case fmt.Stringer:
			buf.AppendString(v.String())
			return
		case *slog.Source:
			buf.AppendString(trimmedPath(v.File, cwd, e.h.opts.TruncateSourcePath))
			buf.AppendByte(':')
			buf.AppendInt(int64(v.Line))
			return
		}
		fallthrough
	case slog.KindString:
		fallthrough
	default:
		buf.AppendString(value.String())
	}
}

func (e *encoder) writeColoredValue(buf *buffer, value slog.Value, style ANSIMod) {
	e.withColor(buf, style, func() {
		e.writeValue(buf, value)
	})
}

func trimmedPath(path string, cwd string, truncate int) string {
	// if the file path appears to be under the current
	// working directory, then we're probably running
	// in a dev environment, and we can show the
	// path of the source file relative to the
	// working directory
	if cwd != "" && strings.HasPrefix(path, cwd) {
		if ff, err := filepath.Rel(cwd, path); err == nil {
			path = ff
		}
	}

	// Otherwise, show the full file path.
	// If truncate is > 0, then truncate to that last
	// number of path segments.
	// 1 = just the filename
	// 2 = the filename and its parent dir
	// 3 = the filename and its two parent dirs
	// ...etc
	//
	// Note that the go compiler always uses forward
	// slashes, even if the compiler was run on Windows.
	//
	// See https://github.com/golang/go/issues/3335
	// and https://github.com/golang/go/issues/18151

	var start int
	for idx := len(path); truncate > 0; truncate-- {
		idx = strings.LastIndexByte(path[:idx], '/')
		if idx == -1 {
			break
		}
		start = idx + 1
	}
	return path[start:]
}
