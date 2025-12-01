/*
Copyright (c) 2025 Tobias Schäfer. All rights reserved.
Licensed under the MIT license, see LICENSE in the project root for details.
*/
package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
	Version = "1.0.0"
	expected := "1.0.0"
	assert.Equal(t, expected, Release())
}

func commitReturnsCommitHashWhenGitCommitIsSet(t *testing.T) {
	GitCommit = "f98352c5101f5097c183cb667401a4f459dc7221"
	expected := "f98352c5101f5097c183cb667401a4f459dc7221"
	assert.Equal(t, expected, Commit())
}

func TestVersion(t *testing.T) {
	t.Run("version.Release returns 'dev' when Version is empty", releaseReturnsDevWhenVersionIsEmpty)
	t.Run("version.Commit returns empty string when GitCommit is empty", commitReturnsEmptyStringWhenGitCommitIsEmpty)
	t.Run("version.Release returns Version when Version is set", releaseReturnsVersionWhenVersionIsSet)
	t.Run("version.Commit returns commit hash when GitCommit is set", commitReturnsCommitHashWhenGitCommitIsSet)
}
