package internal

// FeatureFlagNewMultilineAttrs changes how attribute values containing newlines are handled.
//
// When true, multiline attributes are appended to the end of the log line, after everything in
// in the HeaderFormat, and the keys are printed on their own line, in the form "=== <key> ==="
//
// When false, multiline attribute are printed right after the regular attributes, where ever that
// may be in the HeaderFormat.  Keys are printed just like single-line attributes.
//
// In either case, multiline attributes will only be printed if the HeaderFormat contains an
// a %a directive.
var FeatureFlagNewMultilineAttrs = true
