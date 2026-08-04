// Command bump-linuxaid-hiera bumps the linuxaid-cli version/checksums pinned
// in the LinuxAid Puppet control repo's hiera data.
//
// Usage: bump-linuxaid-hiera <path-to-LinuxAid-checkout> <version> <checksums.txt>
//
// Updates, per architecture:
//   - modules/enableit/common/data/common.yaml                (amd64, also holds the version pin)
//   - modules/enableit/common/data/architectures/armv7l.yaml  (armv7)
//   - modules/enableit/common/data/architectures/aarch64.yaml (arm64)
//
// Edits are done with targeted regexes rather than a full YAML
// parse/re-encode, so every other line and comment in these files is left
// byte-for-byte untouched.
//
// Keeps only the newest keepVersions checksum entries per file.
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const keepVersions = 4

const (
	versionKey   = "common::system::openvox::linuxaid_cli::version"
	checksumsKey = "common::system::openvox::linuxaid_cli::checksums"
)

// files maps each hiera file (relative to the LinuxAid repo root) to the
// release asset architecture suffix it tracks.
var files = map[string]string{
	"modules/enableit/common/data/common.yaml":                "amd64",
	"modules/enableit/common/data/architectures/armv7l.yaml":  "armv7",
	"modules/enableit/common/data/architectures/aarch64.yaml": "arm64",
}

var (
	checksumsBlockRe = regexp.MustCompile(
		`(?m)(^` + regexp.QuoteMeta(checksumsKey) + `:\n)((?:^  \d+\.\d+\.\d+: [0-9a-f]+\n)+)`,
	)
	entryLineRe   = regexp.MustCompile(`(?m)^  (\d+\.\d+\.\d+): ([0-9a-f]+)\n`)
	versionLineRe = regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(versionKey) + `: .*$`)
)

func parseChecksums(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sums := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		sums[fields[1]] = fields[0]
	}
	return sums, scanner.Err()
}

func shaForArch(sums map[string]string, arch, checksumsPath string) (string, error) {
	suffix := fmt.Sprintf("_linux_%s.tar.gz", arch)
	for name, sha := range sums {
		if strings.HasSuffix(name, suffix) {
			return sha, nil
		}
	}
	return "", fmt.Errorf("no checksum ending in %q found in %s", suffix, checksumsPath)
}

type checksumEntry struct {
	version string
	sha     string
}

func bumpChecksumsBlock(text, version, sha string) (string, error) {
	loc := checksumsBlockRe.FindStringSubmatchIndex(text)
	if loc == nil {
		return "", fmt.Errorf("could not find %q block", checksumsKey)
	}
	header := text[loc[2]:loc[3]]
	entriesBlock := text[loc[4]:loc[5]]

	var entries []checksumEntry
	for _, m := range entryLineRe.FindAllStringSubmatch(entriesBlock, -1) {
		if m[1] == version {
			continue
		}
		entries = append(entries, checksumEntry{version: m[1], sha: m[2]})
	}
	entries = append([]checksumEntry{{version: version, sha: sha}}, entries...)
	if len(entries) > keepVersions {
		entries = entries[:keepVersions]
	}

	var b strings.Builder
	b.WriteString(header)
	for _, e := range entries {
		fmt.Fprintf(&b, "  %s: %s\n", e.version, e.sha)
	}

	return text[:loc[0]] + b.String() + text[loc[1]:], nil
}

func bumpVersionLine(text, version string) string {
	return versionLineRe.ReplaceAllStringFunc(text, func(string) string {
		return versionKey + ": " + version
	})
}

func run(repoRoot, version, checksumsPath string) error {
	sums, err := parseChecksums(checksumsPath)
	if err != nil {
		return err
	}

	relPaths := make([]string, 0, len(files))
	for p := range files {
		relPaths = append(relPaths, p)
	}
	sort.Strings(relPaths)

	for _, relPath := range relPaths {
		arch := files[relPath]
		fullPath := filepath.Join(repoRoot, relPath)

		data, err := os.ReadFile(fullPath)
		if err != nil {
			return err
		}

		sha, err := shaForArch(sums, arch, checksumsPath)
		if err != nil {
			return err
		}

		text, err := bumpChecksumsBlock(string(data), version, sha)
		if err != nil {
			return fmt.Errorf("%s: %w", relPath, err)
		}
		text = bumpVersionLine(text, version)

		if err := os.WriteFile(fullPath, []byte(text), 0o644); err != nil {
			return err
		}
		fmt.Printf("updated %s (%s -> %s)\n", relPath, arch, sha)
	}

	return nil
}

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "usage: bump-linuxaid-hiera <path-to-LinuxAid-checkout> <version> <checksums.txt>")
		os.Exit(1)
	}

	if err := run(os.Args[1], os.Args[2], os.Args[3]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
