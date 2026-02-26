package compile

import (
	"bytes"
	"strings"

	"github.com/securacore/codectx/core/manifest"
)

// rewriteLinks rewrites relative markdown links in content to use content-addressed
// object filenames. sourceRelPath is the docs-relative path of the source file
// (e.g., "topics/react/README.md"). pathToHash maps docs-relative file paths to
// their 16-char content hashes. The optional ext parameter specifies the object
// file extension (defaults to ".md"; pass ".cmdx" for compressed output).
//
// For each markdown link [text](target.md):
//   - HTTP/HTTPS links are left untouched.
//   - Links resolvable in pathToHash are rewritten to "{hash}{ext}".
//   - Links not in pathToHash use the "unresolved:" URI scheme: "unresolved:{target}".
//   - Fragment suffixes (#section) are preserved on rewritten links.
//
// The source file content is never modified; a new byte slice is returned.
func rewriteLinks(content []byte, sourceRelPath string, pathToHash map[string]string, ext ...string) []byte {
	objExt := ".md"
	if len(ext) > 0 && ext[0] != "" {
		objExt = ext[0]
	}
	matches := manifest.LinkPattern.FindAllSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return content
	}

	var buf bytes.Buffer
	buf.Grow(len(content))
	lastEnd := 0

	for _, loc := range matches {
		// loc[2]:loc[3] is the captured group 1 (the URL portion).
		target := string(content[loc[2]:loc[3]])

		// Skip absolute URLs (http/https links to .md files on the web).
		if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
			continue
		}

		// Separate fragment from path: "file.md#section" → "file.md", "#section".
		fragment := ""
		pathPart := target
		if idx := strings.Index(target, "#"); idx >= 0 {
			fragment = target[idx:]
			pathPart = target[:idx]
		}

		// Resolve the relative link to a docs-relative path.
		resolved := manifest.ResolveLink(sourceRelPath, pathPart)

		var newTarget string
		if resolved != "" {
			if hash, ok := pathToHash[resolved]; ok {
				// Target exists in the compiled object set.
				newTarget = hash + objExt + fragment
			} else {
				// Target is not in the compiled set.
				newTarget = "unresolved:" + target
			}
		} else {
			// Link escapes the package root directory.
			newTarget = "unresolved:" + target
		}

		// Write everything from the last replacement end up to the URL start,
		// then the rewritten URL.
		buf.Write(content[lastEnd:loc[2]])
		buf.WriteString(newTarget)
		lastEnd = loc[3]
	}

	// Write remaining content after the last match.
	buf.Write(content[lastEnd:])
	return buf.Bytes()
}
