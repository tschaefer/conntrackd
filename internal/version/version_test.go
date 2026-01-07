/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT license, see LICENSE in the project root for details.
*/
package version

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func __capture(f func()) string {
	originalStdout := os.Stdout

	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	_ = w.Close()
	os.Stdout = originalStdout

	var buf = make([]byte, 5096)
	n, _ := r.Read(buf)
	return string(buf[:n])
}

func releaseReturnsDevWhenVersionIsEmpty(t *testing.T) {
	Version = ""
	expected := "dev"
	assert.Equal(t, expected, Release())
}

func commitReturnsEmptyStringWhenGitCommitIsEmpty(t *testing.T) {
	GitCommit = ""
	expected := ""
	assert.Equal(t, expected, Commit())
}

func releaseReturnsVersionWhenVersionIsSet(t *testing.T) {
	Version = "v1.0.0"
	expected := "v1.0.0"
	assert.Equal(t, expected, Release())
}

func commitReturnsCommitHashWhenGitCommitIsSet(t *testing.T) {
	GitCommit = "f98352c5101f5097c183cb667401a4f459dc7221"
	expected := "f98352c5101f5097c183cb667401a4f459dc7221"
	assert.Equal(t, expected, Commit())
}

func bannerReturnsLogo(t *testing.T) {
	expected := `
                       _                  _       _
  ___ ___  _ __  _ __ | |_ _ __ ____  ___| | ____| |
 / __/ _ \| '_ \| '_ \| __| '__/ _  |/ __| |/ / _  |
| (_| (_) | | | | | | | |_| | | (_| | (__|   < (_| |
 \___\___/|_| |_|_| |_|\__|_|  \__,_|\___|_|\_\__,_|
 `
	assert.Equal(t, expected, Banner())
	assert.Len(t, Banner(), 266)
}

func printReturnsVersionAndCommit(t *testing.T) {
	Version = "v1.0.0"
	GitCommit = "f98352c5101f5097c183cb667401a4f459dc7221"

	output := __capture(func() {
		Print()
	})
	assert.Contains(t, output, "Release: "+Release())
	assert.Contains(t, output, "Commit:  "+Commit())
}

func TestVersion(t *testing.T) {
	t.Run("version.Release returns 'dev' when Version is empty", releaseReturnsDevWhenVersionIsEmpty)
	t.Run("version.Commit returns empty string when GitCommit is empty", commitReturnsEmptyStringWhenGitCommitIsEmpty)
	t.Run("version.Release returns Version when Version is set", releaseReturnsVersionWhenVersionIsSet)
	t.Run("version.Commit returns commit hash when GitCommit is set", commitReturnsCommitHashWhenGitCommitIsSet)
	t.Run("version.Banner returns the correct logo", bannerReturnsLogo)
	t.Run("version.Print prints logo, version and commit", printReturnsVersionAndCommit)
}
