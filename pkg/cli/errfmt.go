package cli

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/m-mizutani/zenv/v2/pkg/model"
)

// ANSI color helpers. Active only when the caller passes color=true.
const (
	ansiReset  = "\x1b[0m"
	ansiBold   = "\x1b[1m"
	ansiRed    = "\x1b[31m"
	ansiCyan   = "\x1b[36m"
	ansiYellow = "\x1b[33m"
	ansiDim    = "\x1b[2m"
)

// FormatError renders err for human consumption on a terminal.
//
//   - verbose=true attaches the underlying goerr trace (%+v) for debugging.
//   - color=true emits ANSI escape sequences.
//
// The returned string does not have a trailing newline; callers may append
// one when writing it out.
func FormatError(err error, verbose, color bool) string {
	if err == nil {
		return ""
	}
	w := &errWriter{color: color}
	w.writeHeader("Error", err)
	formatErrorBody(w, err, 1)
	if verbose {
		w.writeRaw("\n")
		w.writeDim("--- debug ---")
		w.writeRaw("\n")
		w.writeRaw(fmt.Sprintf("%+v", err))
	}
	return strings.TrimRight(w.b.String(), "\n")
}

// errWriter buffers formatted output and centralizes the color toggle.
type errWriter struct {
	b     strings.Builder
	color bool
}

func (w *errWriter) writeRaw(s string) { w.b.WriteString(s) }
func (w *errWriter) writeLine(indent int, s string) {
	w.b.WriteString(strings.Repeat("  ", indent) + s + "\n")
}
func (w *errWriter) writeBoldRed(s string) string { return w.colored(ansiBold+ansiRed, s) }
func (w *errWriter) writeCyan(s string) string    { return w.colored(ansiCyan, s) }
func (w *errWriter) writeYellow(s string) string  { return w.colored(ansiYellow, s) }
func (w *errWriter) writeDim(s string)            { w.b.WriteString(w.colored(ansiDim, s)) }
func (w *errWriter) colored(code, s string) string {
	if !w.color {
		return s
	}
	return code + s + ansiReset
}

// writeHeader renders the top "Error: <summary>" line based on err's outermost
// typed shape.
func (w *errWriter) writeHeader(label string, err error) {
	prefix := w.writeBoldRed(label + ":")
	summary := topSummary(err)
	w.b.WriteString(prefix + " " + summary + "\n")
}

// topSummary produces a one-line "what failed" for the outermost typed error.
// It deliberately does not include details that belong on the indented body.
func topSummary(err error) string {
	var le *model.CommandLaunchError
	if errors.As(err, &le) {
		name := launchName(le)
		switch le.Reason {
		case model.LaunchNotFound:
			return fmt.Sprintf("Command %q not found in PATH", name)
		case model.LaunchPermissionDenied:
			return fmt.Sprintf("Command %q is not executable", name)
		default:
			return fmt.Sprintf("Command %q failed to launch", name)
		}
	}
	var ve *model.VariableError
	if errors.As(err, &ve) {
		if ve.Key != "" {
			return fmt.Sprintf("Failed to resolve variable %q", ve.Key)
		}
		return "Failed to resolve variable"
	}
	var cfe *model.ConfigFileError
	if errors.As(err, &cfe) {
		switch cfe.Reason {
		case model.ReasonNotReadable:
			return fmt.Sprintf("Cannot read %s config file", cfe.Format)
		case model.ReasonParseError:
			return fmt.Sprintf("Cannot parse %s config file", cfe.Format)
		case model.ReasonInvalidSchema:
			return fmt.Sprintf("Invalid %s configuration", cfe.Format)
		case model.ReasonNotFound:
			return fmt.Sprintf("Missing %s config file", cfe.Format)
		}
	}
	var pe *model.ProfileNotFoundError
	if errors.As(err, &pe) {
		return fmt.Sprintf("Profile %q is not defined", pe.Profile)
	}
	var le2 *model.InvalidLogLevelError
	if errors.As(err, &le2) {
		return fmt.Sprintf("Invalid log level %q", le2.Value)
	}
	return err.Error()
}

// launchName returns the executable name component for display. It falls back
// to a generic word when the command vector is empty or its first element is
// an empty string, so the rendered message does not contain an empty `""`.
func launchName(le *model.CommandLaunchError) string {
	if len(le.Command) > 0 && le.Command[0] != "" {
		return le.Command[0]
	}
	return "command"
}

// formatErrorBody walks the cause chain and writes the indented "Source" /
// "Cause" / "Hint" sections.
func formatErrorBody(w *errWriter, err error, indent int) {
	// Source line (file/profile) comes only from the outermost VariableError or
	// ConfigFileError; deeper nesting does not repeat it.
	if indent == 1 {
		if src := sourceLine(err, w); src != "" {
			w.writeLine(indent, src)
		}
	}

	// Walk the cause chain, picking the next "interesting" typed node.
	// Steps that produce no visible output (e.g. the outer VariableError whose
	// summary already appeared in the header) must NOT consume the "first"
	// flag, otherwise the first visible cause loses its "Cause:" label.
	cur := err
	depth := indent
	first := true
	for cur != nil {
		line, ok, next := describeNode(cur, w, first)
		if !ok {
			break
		}
		if line != "" {
			w.writeLine(depth, line)
			depth++
			first = false
		}
		if next == nil {
			break
		}
		cur = next
	}

	// Hint
	if h := hintFor(err, w); h != "" {
		w.writeRaw("\n")
		w.writeLine(indent, h)
	}
}

// sourceLine formats the "Source:" header for the outermost error.
func sourceLine(err error, w *errWriter) string {
	var ve *model.VariableError
	if errors.As(err, &ve) {
		switch {
		case ve.Path != "" && ve.Profile != "":
			return w.writeCyan("Source: ") + ve.Path + "  (profile: " + ve.Profile + ")"
		case ve.Path != "":
			return w.writeCyan("Source: ") + ve.Path
		case ve.Profile != "":
			return w.writeCyan("Source: ") + "profile: " + ve.Profile
		default:
			return ""
		}
	}
	var cfe *model.ConfigFileError
	if errors.As(err, &cfe) {
		if cfe.Path == "" {
			return ""
		}
		return w.writeCyan("Source: ") + cfe.Path
	}
	var pe *model.ProfileNotFoundError
	if errors.As(err, &pe) && len(pe.Paths) > 0 {
		return w.writeCyan("Source: ") + strings.Join(pe.Paths, ", ")
	}
	return ""
}

// describeNode formats the current error node and returns (line, more, nextCause).
// When more=false, the traversal stops without printing line. When more=true and
// line is non-empty, the caller prints line and descends into nextCause.
func describeNode(err error, w *errWriter, isFirstCause bool) (string, bool, error) {
	var ve *model.VariableError
	if errors.As(err, &ve) && ve == err {
		// The outer header already mentioned "Failed to resolve variable",
		// so for the outer-most VariableError we just descend.
		return "", true, ve.Cause
	}

	var ref *model.ReferenceError
	if errors.As(err, &ref) && ref == err {
		label := w.writeCyan(causeLabel(isFirstCause))
		switch ref.Reason {
		case model.RefNotFound:
			line := label + fmt.Sprintf("Reference %q is not defined", ref.Ref)
			if hint := suggestSimilar(ref.Ref, ref.Available); hint != "" {
				line += "  (did you mean " + hint + "?)"
			}
			return line, true, ref.Cause
		case model.RefCircular:
			chain := strings.Join(ref.Chain, " -> ")
			return label + fmt.Sprintf("Circular reference: %s", chain), true, ref.Cause
		case model.RefResolveFailed:
			return label + fmt.Sprintf("Reference %q could not be resolved", ref.Ref), true, ref.Cause
		}
	}

	var re *model.ResolveError
	if errors.As(err, &re) && re == err {
		label := w.writeCyan(causeLabel(isFirstCause))
		line := label + fmt.Sprintf("%s failed", capitalize(re.Op.String()))
		if re.Target != "" {
			line += ": " + re.Target
		}
		return line, true, re.Cause
	}

	var cmd *model.CommandExecError
	if errors.As(err, &cmd) && cmd == err {
		label := w.writeCyan(causeLabel(isFirstCause))
		line := label + "Command failed: " + strings.Join(cmd.Command, " ")
		if cmd.ExitCode != 0 {
			line += fmt.Sprintf("  (exit %d)", cmd.ExitCode)
		}
		// stderr tail prints on additional indented lines
		if cmd.Stderr != "" {
			line += "\n" + indentStderr(cmd.Stderr, w)
		}
		// Do not descend into cmd.Cause. The Cause is usually a redundant
		// restatement of the exit code (e.g. "exit status 1") and the
		// exit code + stderr already convey what happened.
		return line, true, nil
	}

	var cfe *model.ConfigFileError
	if errors.As(err, &cfe) && cfe == err {
		label := w.writeCyan(causeLabel(isFirstCause))
		line := label + cfe.Error()
		return line, true, cfe.Cause
	}

	var le *model.CommandLaunchError
	if errors.As(err, &le) && le == err {
		// The summary header already says "Command \"x\" not found / not
		// executable". The cause line carries the underlying OS error verbatim
		// so the user can copy-paste it into a search.
		if le.Cause == nil {
			return "", true, nil
		}
		label := w.writeCyan(causeLabel(isFirstCause))
		return label + le.Cause.Error(), true, nil
	}

	// Fallback: leaf or unknown error
	if isFirstCause {
		// No structured cause-of-cause to dig into; just print Error().
		return w.writeCyan("Cause: ") + err.Error(), true, nil
	}
	// Inner unknown leaf — just append its message
	return err.Error(), true, nil
}

func causeLabel(first bool) string {
	if first {
		return "Cause:  "
	}
	return "└ "
}

// capitalize uppercases the first rune of s. It walks runes (not bytes), so
// multi-byte leading characters are handled correctly.
func capitalize(s string) string {
	for i, r := range s {
		return string(unicode.ToUpper(r)) + s[i+len(string(r)):]
	}
	return ""
}

// indentStderr inserts a prefix on each line of stderr output.
func indentStderr(s string, w *errWriter) string {
	prefix := strings.Repeat("  ", 3) + w.colored(ansiDim, "stderr> ")
	var out strings.Builder
	for i, line := range strings.Split(strings.TrimRight(s, "\n"), "\n") {
		if i > 0 {
			out.WriteString("\n")
		}
		out.WriteString(prefix + line)
	}
	return out.String()
}

// hintFor builds an actionable hint based on what's inside err.
func hintFor(err error, w *errWriter) string {
	var le *model.CommandLaunchError
	if errors.As(err, &le) {
		name := launchName(le)
		switch le.Reason {
		case model.LaunchNotFound:
			line := fmt.Sprintf("Hint:  ensure %q is installed and visible from your shell PATH", name)
			return w.writeYellow(line)
		case model.LaunchPermissionDenied:
			line := fmt.Sprintf("Hint:  check that %q has the executable bit set (e.g. chmod +x)", name)
			return w.writeYellow(line)
		default:
			line := fmt.Sprintf("Hint:  verify that %q can be launched in your environment", name)
			return w.writeYellow(line)
		}
	}
	var cmd *model.CommandExecError
	if errors.As(err, &cmd) {
		line := "Hint:  ensure `" + strings.Join(cmd.Command, " ") + "` runs successfully in your shell"
		return w.writeYellow(line)
	}
	var ref *model.ReferenceError
	if errors.As(err, &ref) && ref.Reason == model.RefNotFound {
		line := fmt.Sprintf("Hint:  define %q in your config, .env, or system environment", ref.Ref)
		return w.writeYellow(line)
	}
	if errors.As(err, &ref) && ref.Reason == model.RefCircular {
		return w.writeYellow("Hint:  break the cycle by removing or restructuring one of the references")
	}
	var cfe *model.ConfigFileError
	if errors.As(err, &cfe) {
		switch cfe.Reason {
		case model.ReasonParseError:
			return w.writeYellow("Hint:  fix the syntax error reported above and retry")
		case model.ReasonNotFound:
			return w.writeYellow(fmt.Sprintf("Hint:  create the file at %q or remove the flag that points to it", cfe.Path))
		}
	}
	var pe *model.ProfileNotFoundError
	if errors.As(err, &pe) {
		switch {
		case len(pe.Available) > 0:
			return w.writeYellow("Hint:  pick one of the available profiles: " + strings.Join(pe.Available, ", "))
		case len(pe.Paths) == 0:
			return w.writeYellow("Hint:  no configuration file was loaded; create one or omit --profile")
		default:
			return w.writeYellow("Hint:  define a profile block in your configuration or omit --profile")
		}
	}
	var ll *model.InvalidLogLevelError
	if errors.As(err, &ll) && len(ll.Allowed) > 0 {
		return w.writeYellow("Hint:  use one of " + strings.Join(ll.Allowed, ", "))
	}
	return ""
}

// suggestSimilar returns a quoted candidate from available that most closely
// matches ref. The match is intentionally simple (substring or short edit
// distance); good enough for typo hints. Returns "" if nothing useful.
func suggestSimilar(ref string, available []string) string {
	if ref == "" || len(available) == 0 {
		return ""
	}
	type cand struct {
		name string
		dist int
	}
	var best []cand
	for _, name := range available {
		if name == "" {
			continue
		}
		d := editDistance(strings.ToLower(ref), strings.ToLower(name))
		// Heuristic: only suggest when within ceiling
		if d <= 3 || strings.Contains(strings.ToLower(name), strings.ToLower(ref)) {
			best = append(best, cand{name, d})
		}
	}
	if len(best) == 0 {
		return ""
	}
	sort.Slice(best, func(i, j int) bool { return best[i].dist < best[j].dist })
	limit := min(3, len(best))
	names := make([]string, 0, limit)
	for _, c := range best[:limit] {
		names = append(names, fmt.Sprintf("%q", c.name))
	}
	return strings.Join(names, ", ")
}

// editDistance is a small Levenshtein implementation used for typo hints.
func editDistance(a, b string) int {
	if a == b {
		return 0
	}
	ar, br := []rune(a), []rune(b)
	prev := make([]int, len(br)+1)
	cur := make([]int, len(br)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ar); i++ {
		cur[0] = i
		for j := 1; j <= len(br); j++ {
			cost := 1
			if ar[i-1] == br[j-1] {
				cost = 0
			}
			del := prev[j] + 1
			ins := cur[j-1] + 1
			sub := prev[j-1] + cost
			cur[j] = min(del, ins, sub)
		}
		prev, cur = cur, prev
	}
	return prev[len(br)]
}
