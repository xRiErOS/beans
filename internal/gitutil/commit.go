package gitutil

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// ResolveCommit resolves ref (HEAD, branch, tag, short or full SHA) to a full
// 40-character commit SHA in the repository at dir.
func ResolveCommit(dir, ref string) (string, error) {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--verify", "--quiet", ref+"^{commit}")
	out, err := cmd.Output()
	sha := strings.TrimSpace(string(out))
	if err != nil || len(sha) != 40 {
		return "", fmt.Errorf("%q does not resolve to a commit in %s", ref, dir)
	}
	return sha, nil
}

// CommitsExist reports for each sha whether it names a commit object in the
// repository at dir, using a single `git cat-file --batch-check` process.
// ok is false when dir is not a git repository.
func CommitsExist(dir string, shas []string) (result map[string]bool, ok bool) {
	result = make(map[string]bool, len(shas))
	if len(shas) == 0 {
		return result, true
	}

	cmd := exec.Command("git", "-C", dir, "cat-file", "--batch-check")
	var stdin bytes.Buffer
	for _, sha := range shas {
		stdin.WriteString(sha)
		stdin.WriteString("^{commit}\n")
	}
	cmd.Stdin = &stdin

	out, err := cmd.Output()
	if err != nil {
		return result, false
	}

	scanner := bufio.NewScanner(bytes.NewReader(out))
	i := 0
	for scanner.Scan() && i < len(shas) {
		line := scanner.Text()
		result[shas[i]] = !strings.HasSuffix(line, " missing")
		i++
	}
	return result, true
}
