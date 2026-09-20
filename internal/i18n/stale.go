package i18n

import (
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// LastCommit returns the timestamp of the last commit that touched a path, and
// whether such a commit exists.
type LastCommit func(path string) (time.Time, bool, error)

// Stale reports whether dst is older than src according to last. ok is false
// when either file has no recorded commit, so no conclusion can be drawn.
func Stale(last LastCommit, src, dst string) (stale, ok bool, err error) {
	srcTime, srcOK, err := last(src)
	if err != nil {
		return false, false, err
	}
	dstTime, dstOK, err := last(dst)
	if err != nil {
		return false, false, err
	}
	if !srcOK || !dstOK {
		return false, false, nil
	}
	return dstTime.Before(srcTime), true, nil
}

// GitLastCommit returns a LastCommit that reads commit timestamps from the git
// repository at root. Paths are relative to root. A missing git binary, a
// non-repository root, or a path with no commits all report ok=false.
func GitLastCommit(root string) LastCommit {
	return func(path string) (time.Time, bool, error) {
		cmd := exec.Command("git", "-C", root, "log", "-1", "--format=%ct", "--", path)
		out, err := cmd.Output()
		if err != nil {
			return time.Time{}, false, nil
		}
		secs := strings.TrimSpace(string(out))
		if secs == "" {
			return time.Time{}, false, nil
		}
		n, err := strconv.ParseInt(secs, 10, 64)
		if err != nil {
			return time.Time{}, false, nil
		}
		return time.Unix(n, 0).UTC(), true, nil
	}
}
