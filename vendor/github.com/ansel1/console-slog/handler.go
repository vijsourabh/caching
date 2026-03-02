package console

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/ansel1/console-slog/internal"
)

var cwd string

func init() {
	cwd, _ = os.Getwd()
	// We compare cwd to the filepath in runtime.Frame.File
	// It turns out, an old legacy behavior of go is that runtime.Frame.File
	// will always contain file paths with forward slashes, even if compiled
	// on Windows.
	// See https://github.com/golang/go/issues/3335
	// and https://github.com/golang/go/issues/18151
	cwd = strings.ReplaceAll(cwd, "\\", "/")
}

// HandlerOptions are options for a ConsoleHandler.
// A zero HandlerOptions consists entirely of default values.
// ReplaceAttr works identically to [slog.HandlerOptions.ReplaceAttr]
type HandlerOptions struct {
	// AddSource causes the handler to compute the source code position
	// of the log statement and add a SourceKey attribute to the output.
	AddSource bool

	// Level reports the minimum record level that will be logged.
	// The handler discards records with lower levels.
	// If Level is nil, the handler assumes LevelInfo.
	// The handler calls Level.Level for each record processed;
	// to adjust the minimum level dynamically, use a LevelVar.
	Level slog.Leveler

	// Disable colorized output
	NoColor bool

	// TimeFormat is the format used for time.DateTime
	TimeFormat string

	// Theme defines the colorized output using ANSI escape sequences
	Theme Theme

	// ReplaceAttr is called to rewrite each non-group attribute before it is logged.
	// See [slog.HandlerOptions]
	ReplaceAttr func(groups []string, a slog.Attr) slog.Attr

	// TruncateSourcePath shortens the source file path, if AddSource=true.
	// If 0, no truncation is done.
	// If >0, the file path is truncated to that many trailing path segments.
	// For example:
	//
	//     users.go:34						// TruncateSourcePath = 1
	//     models/users.go:34				// TruncateSourcePath = 2
	//     ...etc
	TruncateSourcePath int

	// HeaderFormat specifies the format of the log header.
	//
	// The default format is "%t %l %[source]h > %m".
	//
	// The format is a string containing verbs, which are expanded as follows:
	//
	//	%t	       timestamp
	//	%l	       abbreviated level (e.g. "INF")
	//	%L	       level (e.g. "INFO")
	//	%m	       message
	//	%s	       source (if omitted, source is just handled as an attribute)
	//	%a	       attributes
	//	%[key]h	   header with the given key.
	//  %{         group open
	//  %(style){  group open with style - applies the specified Theme style to any strings in the group
	//  %}         group close
	//
	// Headers print the value of the attribute with the given key, and remove that
	// attribute from the end of the log line.
	//
	// Headers can be customized with width and alignment modifiers,
	// similar to fmt.Printf verbs. For example:
	//
	//	%[key]10h		// left-aligned, width 10
	//	%[key]-10h		// right-aligned, width 10
	//
	// Groups will omit their contents if all the fields in that group are omitted.  For example:
	//
	//	"%l %{%[logger]h %[source]h > %} %m"
	//
	// will print "INF main main.go:123 > msg" if the either the logger or source attribute is present.  But if the
	// both attributes are not present, or were elided by ReplaceAttr, then this will print "INF msg".  Groups can
	// be nested.
	//
	// Groups can also be styled using the Theme styles by specifying a style in parentheses after the percent sign:
	//
	//	"%l %(source){ %[logger]h %} %m"
	//
	// will apply the source style from the Theme to the fixed strings in the group. By default, the Header style is used.
	//
	// Whitespace is generally merged to leave a single space between fields.  Leading and trailing whitespace is trimmed.
	//
	// Examples:
	//
	//	"%t %l %m"                         // timestamp, level, message
	//	"%t [%l] %m"                       // timestamp, level in brackets, message
	//	"%t %l:%m"                         // timestamp, level:message
	//	"%t %l %[key]h %m"                 // timestamp, level, header with key "key", message
	//	"%t %l %[key1]h %[key2]h %m"       // timestamp, level, header with key "key1", header with key "key2", message
	//	"%t %l %[key]10h %m"               // timestamp, level, header with key "key" and width 10, message
	//	"%t %l %[key]-10h %m"              // timestamp, level, right-aligned header with key "key" and width 10, message
	//	"%t %l %L %m"                      // timestamp, abbreviated level, non-abbreviated level, message
	//	"%t %l %L- %m"                     // timestamp, abbreviated level, right-aligned non-abbreviated level, message
	//	"%t %l %m string literal"          // timestamp, level, message, and then " string literal"
	//	"prefix %t %l %m suffix"           // "prefix ", timestamp, level, message, and then " suffix"
	//	"%% %t %l %m"                      // literal "%", timestamp, level, message
	//  "%{[%t]%} %{[%l]%} %m"             // timestamp and level in brackets, message, brackets will be omitted if empty
	HeaderFormat string
}

const defaultHeaderFormat = "%t %l %{%s >%} %m %a"

type Handler struct {
	opts                      HandlerOptions
	out                       io.Writer
	groupPrefix               string
	groups                    []string
	context, multilineContext buffer
	fields                    []any
	headerFields              []headerField
	sourceAsAttr              bool
	mu                        *sync.Mutex
}

type timestampField struct{}

type headerField struct {
	groupPrefix string
	key         string
	width       int
	rightAlign  bool
	memo        string
}

type levelField struct {
	abbreviated bool
}
type messageField struct{}

type attrsField struct{}

type groupOpen struct {
	style string
}
type groupClose struct{}

type spacer struct {
	hard bool
}

type sourceField struct{}

var _ slog.Handler = (*Handler)(nil)

// NewHandler creates a Handler that writes to w,
// using the given options.
// If opts is nil, the default options are used.
func NewHandler(out io.Writer, opts *HandlerOptions) *Handler {
	if opts == nil {
		opts = new(HandlerOptions)
	}
	if opts.Level == nil {
		opts.Level = slog.LevelInfo
	}
	if opts.TimeFormat == "" {
		opts.TimeFormat = time.DateTime
	}
	if opts.Theme.Name == "" {
		opts.Theme = NewDefaultTheme()
	}
	if opts.HeaderFormat == "" {
		opts.HeaderFormat = defaultHeaderFormat // default format
	}

	fields, headerFields := parseFormat(opts.HeaderFormat, opts.Theme)

	// find spocerFields adjacent to string fields and mark them
	// as hard spaces.  hard spaces should not be skipped, only
	// coalesced
	var wasString bool
	lastSpace := -1
	for i, f := range fields {
		switch f.(type) {
		case headerField, levelField, messageField, timestampField:
			wasString = false
			lastSpace = -1
		case string:
			if lastSpace != -1 {
				// string immediately followed space, so the
				// space is hard.
				fields[lastSpace] = spacer{hard: true}
			}
			wasString = true
			lastSpace = -1
		case spacer:
			if wasString {
				// space immedately followed a string, so the space
				// is hard
				fields[i] = spacer{hard: true}
			}
			lastSpace = i
			wasString = false
		}
	}

	// Check if the parsed fields include any sourceField instances
	// If not, set sourceAsAttr to true so source is handled as a regular attribute
	sourceAsAttr := true
	for _, f := range fields {
		if _, ok := f.(sourceField); ok {
			sourceAsAttr = false
			break
		}
	}

	return &Handler{
		opts:         *opts, // Copy struct
		out:          out,
		groupPrefix:  "",
		context:      nil,
		fields:       fields,
		headerFields: headerFields,
		sourceAsAttr: sourceAsAttr,
		mu:           &sync.Mutex{},
	}
}

// Enabled implements slog.Handler.
func (h *Handler) Enabled(_ context.Context, l slog.Level) bool {
	return l >= h.opts.Level.Level()
}

func (h *Handler) Handle(ctx context.Context, rec slog.Record) error {
	enc := newEncoder(h)

	var src slog.Source

	if h.opts.AddSource && rec.PC > 0 {
		frame, _ := runtime.CallersFrames([]uintptr{rec.PC}).Next()
		src.Function = frame.Function
		src.File = frame.File
		src.Line = frame.Line

		if h.sourceAsAttr {
			// the source attr should not be inside any open groups
			groups := enc.groups
			enc.groups = nil
			enc.encodeAttr("", slog.Any(slog.SourceKey, &src))
			enc.groups = groups
		}
	}

	enc.attrBuf.Append(h.context)
	enc.multilineAttrBuf.Append(h.multilineContext)

	rec.Attrs(func(a slog.Attr) bool {
		enc.encodeAttr(h.groupPrefix, a)
		return true
	})

	headerIdx := 0
	var state encodeState
	// use a fixed size stack to avoid allocations, 3 deep nested groups should be enough for most cases
	stackArr := [3]encodeState{}
	stack := stackArr[:0]
	var attrsFieldSeen bool
	for _, f := range h.fields {
		switch f := f.(type) {
		case groupOpen:
			stack = append(stack, state)
			state.groupStart = len(enc.buf)
			state.printedField = false
			state.seenFields = 0
			// Store the style to use for this group
			state.style = f.style
			continue
		case groupClose:
			if len(stack) == 0 {
				// missing group open
				// no-op
				continue
			}

			if state.printedField || state.seenFields == 0 {
				// merge the current state with the prior state
				lastState := stack[len(stack)-1]
				state.groupStart = lastState.groupStart
				state.style = lastState.style
				state.seenFields += lastState.seenFields
			} else {
				// no fields were printed in this group, so
				// rollback the entire group and pop back to
				// the outer state
				enc.buf = enc.buf[:state.groupStart]
				state = stack[len(stack)-1]
			}
			// pop a state off the stack
			stack = stack[:len(stack)-1]
			continue
		case spacer:
			if len(enc.buf) == 0 {
				// special case, always skip leading space
				continue
			}

			if f.hard {
				state.pendingHardSpace = true
			} else {
				// only queue a soft space if the last
				// thing printed was not a string field.
				state.pendingSpace = state.anchored
			}

			continue
		case string:
			if state.pendingHardSpace {
				enc.buf.AppendByte(' ')
			}
			state.pendingHardSpace = false
			state.pendingSpace = false
			state.anchored = false

			// Use the style specified for the group if available
			style, _ := getThemeStyleByName(h.opts.Theme, state.style)
			enc.withColor(&enc.buf, style, func() {
				enc.buf.AppendString(f)
			})
			continue
		}
		if state.pendingSpace || state.pendingHardSpace {
			enc.buf.AppendByte(' ')
		}
		l := len(enc.buf)
		state.seenFields++
		switch f := f.(type) {
		case headerField:
			hf := h.headerFields[headerIdx]
			if enc.headerAttrs[headerIdx].Equal(slog.Attr{}) && hf.memo != "" {
				enc.buf.AppendString(hf.memo)
			} else {
				enc.encodeHeader(enc.headerAttrs[headerIdx], hf.width, hf.rightAlign)
			}
			headerIdx++

		case levelField:
			enc.encodeLevel(rec.Level, f.abbreviated)
		case messageField:
			enc.encodeMessage(rec.Level, rec.Message)
		case attrsField:
			// trim the attrBuf and multilineAttrBuf to remove leading spaces
			// but leave a space between attrBuf and multilineAttrBuf
			if len(enc.attrBuf) > 0 {
				enc.attrBuf = bytes.TrimSpace(enc.attrBuf)
			} else if len(enc.multilineAttrBuf) > 0 && !internal.FeatureFlagNewMultilineAttrs {
				enc.multilineAttrBuf = bytes.TrimSpace(enc.multilineAttrBuf)
			}
			attrsFieldSeen = true
			enc.buf.Append(enc.attrBuf)
			if !internal.FeatureFlagNewMultilineAttrs {
				enc.buf.Append(enc.multilineAttrBuf)
			}
		case sourceField:
			enc.encodeSource(src)
		case timestampField:
			enc.encodeTimestamp(rec.Time)
		}
		printed := len(enc.buf) > l
		state.printedField = state.printedField || printed
		if printed {
			state.pendingSpace = false
			state.pendingHardSpace = false
			state.anchored = true
		} else if state.pendingSpace || state.pendingHardSpace {
			// chop the last space
			enc.buf = bytes.TrimSpace(enc.buf)
			// leave state.spacePending as is for next
			// field to handle
		}
	}

	if internal.FeatureFlagNewMultilineAttrs && attrsFieldSeen && len(enc.multilineAttrBuf) > 0 {
		enc.buf.Append(enc.multilineAttrBuf)
	}

	enc.buf.AppendByte('\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	if _, err := enc.buf.WriteTo(h.out); err != nil {
		return err
	}

	enc.free()
	return nil
}

type encodeState struct {
	// index in buffer of where the currently open group started.
	// if group ends up being elided, buffer will rollback to this
	// index
	groupStart int
	// whether any field in this group has not been elided.  When a group
	// closes, if this is false, the entire group will be elided
	printedField bool
	// number of fields seen in this group.  If this is 0, then
	// the group only contains fixed strings, and no fields, adn
	// should not be elided.
	seenFields int

	anchored, pendingSpace, pendingHardSpace bool
	style                                    string
}

// WithAttrs implements slog.Handler.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	enc := newEncoder(h)

	for _, a := range attrs {
		enc.encodeAttr(h.groupPrefix, a)
	}

	headerFields := memoizeHeaders(enc, h.headerFields)

	newCtx := h.context
	newMultiCtx := h.multilineContext
	if len(enc.attrBuf) > 0 {
		newCtx = append(newCtx, enc.attrBuf...)
		newCtx = slices.Clip(newCtx)
	}
	if len(enc.multilineAttrBuf) > 0 {
		newMultiCtx = append(newMultiCtx, enc.multilineAttrBuf...)
		newMultiCtx = slices.Clip(newMultiCtx)
	}

	enc.free()

	return &Handler{
		opts:             h.opts,
		out:              h.out,
		groupPrefix:      h.groupPrefix,
		context:          newCtx,
		multilineContext: newMultiCtx,
		groups:           h.groups,
		fields:           h.fields,
		headerFields:     headerFields,
		sourceAsAttr:     h.sourceAsAttr,
		mu:               h.mu,
	}
}

// WithGroup implements slog.Handler.
func (h *Handler) WithGroup(name string) slog.Handler {
	name = strings.TrimSpace(name)
	groupPrefix := name
	if h.groupPrefix != "" {
		groupPrefix = h.groupPrefix + "." + name
	}
	return &Handler{
		opts:         h.opts,
		out:          h.out,
		groupPrefix:  groupPrefix,
		context:      h.context,
		groups:       append(h.groups, name),
		fields:       h.fields,
		headerFields: h.headerFields,
		sourceAsAttr: h.sourceAsAttr,
		mu:           h.mu,
	}
}

func memoizeHeaders(enc *encoder, headerFields []headerField) []headerField {
	newFields := make([]headerField, len(headerFields))
	copy(newFields, headerFields)

	for i := range newFields {
		if !enc.headerAttrs[i].Equal(slog.Attr{}) {
			enc.buf.Reset()
			enc.encodeHeader(enc.headerAttrs[i], newFields[i].width, newFields[i].rightAlign)
			newFields[i].memo = enc.buf.String()
		}
	}
	return newFields
}

// parseFormat parses a format string into a list of fields and the number of headerFields.
//
// Supported format verbs:
//
//		%t	- timestampField
//		%h	- headerField, requires the [name] modifier.
//		      Supports width, right-alignment (-) modifiers.
//		%m	- messageField
//		%l	- abbreviated levelField: The log level in abbreviated form (e.g., "INF").
//		%L	- non-abbreviated levelField: The log level in full form (e.g., "INFO").
//		%{	- groupOpen
//		%}	- groupClose
//	    %s  - sourceField
//
// Modifiers:
//
//	[name] (for %h): The key of the attribute to capture as a header. This modifier is required for the %h verb.
//	width (for %h): An integer specifying the fixed width of the header. This modifier is optional.
//	- (for %h): Indicates right-alignment of the header. This modifier is optional.
//
// Examples:
//
//			"%t %l %m"                         // timestamp, level, message
//			"%t [%l] %m"                       // timestamp, level in brackets, message
//			"%t %l:%m"                         // timestamp, level:message
//			"%t %l %[key]h %m"                 // timestamp, level, header with key "key", message
//			"%t %l %[key1]h %[key2]h %m"       // timestamp, level, header with key "key1", header with key "key2", message
//			"%t %l %[key]10h %m"               // timestamp, level, header with key "key" and width 10, message
//			"%t %l %[key]-10h %m"              // timestamp, level, right-aligned header with key "key" and width 10, message
//			"%t %l %L %m"                      // timestamp, abbreviated level, non-abbreviated level, message
//			"%t %l %L- %m"                     // timestamp, abbreviated level, right-aligned non-abbreviated level, message
//			"%t %l %m string literal"          // timestamp, level, message, and then " string literal"
//			"prefix %t %l %m suffix"           // "prefix ", timestamp, level, message, and then " suffix"
//			"%% %t %l %m"                      // literal "%", timestamp, level, message
//			"%t %l %s"                         // timestamp, level, source location (e.g., "file.go:123 functionName")
//		    "%t %l %m %(source){→ %s%}"        // timestamp, level, message, and then source wrapped in a group with a custom string.
//	                                           // The string in the group will use the "source" style, and the group will be omitted if the source attribute is not present
func parseFormat(format string, theme Theme) (fields []any, headerFields []headerField) {
	fields = make([]any, 0)
	headerFields = make([]headerField, 0)

	format = strings.TrimSpace(format)
	lastWasSpace := false

	for i := 0; i < len(format); i++ {
		if format[i] == ' ' {
			if !lastWasSpace {
				fields = append(fields, spacer{})
				lastWasSpace = true
			}
			continue
		}
		lastWasSpace = false

		if format[i] != '%' {
			// Find the next % or space or end of string
			start := i
			for i < len(format) && format[i] != '%' && format[i] != ' ' {
				i++
			}
			fields = append(fields, format[start:i])
			i-- // compensate for loop increment
			continue
		}

		// Handle %% escape
		if i+1 < len(format) && format[i+1] == '%' {
			fields = append(fields, "%")
			i++
			continue
		}

		// Parse format verb and any modifiers
		i++
		if i >= len(format) {
			fields = append(fields, "%!(MISSING_VERB)")
			break
		}

		// Check for modifiers before verb
		var width int
		var rightAlign bool
		var key string
		var style string
		var styleSeen, keySeen, widthSeen bool

		// Look for (style) modifier for groupOpen
		if format[i] == '(' {
			styleSeen = true
			// Find the next ) or end of string
			end := i + 1
			for end < len(format) && format[end] != ')' && format[end] != ' ' {
				end++
			}
			if end >= len(format) || format[end] != ')' {
				fields = append(fields, fmt.Sprintf("%%!%s(MISSING_CLOSING_PARENTHESIS)", format[i:end]))
				i = end - 1 // Position just before the next character to process
				continue
			}
			style = format[i+1 : end]
			i = end + 1
		}

		// Look for [name] modifier
		if format[i] == '[' {
			keySeen = true
			// Find the next ] or end of string
			end := i + 1
			for end < len(format) && format[end] != ']' && format[end] != ' ' {
				end++
			}
			if end >= len(format) || format[end] != ']' {
				fields = append(fields, fmt.Sprintf("%%!%s(MISSING_CLOSING_BRACKET)", format[i:end]))
				i = end - 1 // Position just before the next character to process
				continue
			}
			key = format[i+1 : end]
			i = end + 1
		}

		// Look for modifiers
		for i < len(format) {
			if format[i] == '-' {
				rightAlign = true
				i++
			} else if format[i] >= '0' && format[i] <= '9' {
				widthSeen = true
				width = 0
				for i < len(format) && format[i] >= '0' && format[i] <= '9' {
					width = width*10 + int(format[i]-'0')
					i++
				}
			} else {
				break
			}
		}

		if i >= len(format) {
			fields = append(fields, "%!(MISSING_VERB)")
			break
		}

		var field any

		// Parse the verb
		switch format[i] {
		case ' ':
			fields = append(fields, "%!(MISSING_VERB)")
			// backtrack so the space is included in the next field
			i--
			continue
		case 't':
			field = timestampField{}
		case 'h':
			if key == "" {
				fields = append(fields, "%!h(MISSING_HEADER_NAME)")
				continue
			}
			hf := headerField{
				key:        key,
				width:      width,
				rightAlign: rightAlign,
			}
			if idx := strings.LastIndexByte(key, '.'); idx > -1 {
				hf.groupPrefix = key[:idx]
				hf.key = key[idx+1:]
			}
			field = hf
		case 'm':
			field = messageField{}
		case 'l':
			field = levelField{abbreviated: true}
		case 'L':
			field = levelField{abbreviated: false}
		case '{':
			if _, ok := getThemeStyleByName(theme, style); !ok {
				fields = append(fields, fmt.Sprintf("%%!{(%s)(INVALID_STYLE_MODIFIER)", style))
				continue
			}
			field = groupOpen{style: style}
		case '}':
			field = groupClose{}
		case 's':
			field = sourceField{}
		case 'a':
			field = attrsField{}
		default:
			fields = append(fields, fmt.Sprintf("%%!%c(INVALID_VERB)", format[i]))
			continue
		}

		// Check for invalid combinations
		switch {
		case styleSeen && format[i] != '{':
			fields = append(fields, fmt.Sprintf("%%!((INVALID_MODIFIER)%c", format[i]))
			continue
		case keySeen && format[i] != 'h':
			fields = append(fields, fmt.Sprintf("%%![(INVALID_MODIFIER)%c", format[i]))
			continue
		case widthSeen && format[i] != 'h':
			fields = append(fields, fmt.Sprintf("%%!%d(INVALID_MODIFIER)%c", width, format[i]))
			continue
		case rightAlign && format[i] != 'h':
			fields = append(fields, fmt.Sprintf("%%!-(INVALID_MODIFIER)%c", format[i]))
			continue
		}

		fields = append(fields, field)
		if _, ok := field.(headerField); ok {
			headerFields = append(headerFields, field.(headerField))
		}
	}

	return fields, headerFields
}

// Helper function to get style from theme by name
func getThemeStyleByName(theme Theme, name string) (ANSIMod, bool) {
	switch name {
	case "":
		return theme.Header, true
	case "timestamp":
		return theme.Timestamp, true
	case "header":
		return theme.Header, true
	case "source":
		return theme.Source, true
	case "message":
		return theme.Message, true
	case "messageDebug":
		return theme.MessageDebug, true
	case "attrKey":
		return theme.AttrKey, true
	case "attrValue":
		return theme.AttrValue, true
	case "attrValueError":
		return theme.AttrValueError, true
	case "levelError":
		return theme.LevelError, true
	case "levelWarn":
		return theme.LevelWarn, true
	case "levelInfo":
		return theme.LevelInfo, true
	case "levelDebug":
		return theme.LevelDebug, true
	default:
		return theme.Header, false // Default to header style, but indicate style was not recognized
	}
}
