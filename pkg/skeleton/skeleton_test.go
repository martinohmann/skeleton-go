package skeleton

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsInsideSkeletonDir(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		expected    bool
		expectedErr error
	}{
		{
			name:     "not inside of skeleton dir",
			path:     "../testdata/repos/advanced/skeletons/notaskeleton/somefile",
			expected: false,
		},
		{
			name:     "inside of skeleton dir",
			path:     "../testdata/repos/advanced/skeletons/bar/subdir/somefile.txt",
			expected: true,
		},
		{
			name:     "path is a skeleton dir",
			path:     "../testdata/repos/advanced/skeletons/bar",
			expected: false,
		},
		{
			name:     "file does not exist",
			path:     "nonexistent",
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := IsInsideSkeletonDir(test.path)
			if test.expectedErr != nil {
				require.Error(t, err)
				assert.Equal(t, test.expectedErr.Error(), err.Error())
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expected, result)
			}
		})
	}
}

func TestFindSkeletonDir(t *testing.T) {
	pwd, _ := os.Getwd()

	tests := []struct {
		name        string
		path        string
		expected    string
		expectedErr error
	}{
		{
			name:        "not inside of skeleton dir",
			path:        "../testdata/repos/advanced/skeletons/notaskeleton/somefile",
			expectedErr: ErrDirNotFound,
		},
		{
			name:     "skeleton dir",
			path:     "../testdata/repos/advanced/skeletons/bar",
			expected: filepath.Join(pwd, "../testdata/repos/advanced/skeletons/bar"),
		},
		{
			name:     "dir inside of skeleton dir",
			path:     "../testdata/repos/advanced/skeletons/bar/subdir",
			expected: filepath.Join(pwd, "../testdata/repos/advanced/skeletons/bar"),
		},
		{
			name:     "file in dir inside of skeleton dir",
			path:     "../testdata/repos/advanced/skeletons/bar/subdir/somefile.txt",
			expected: filepath.Join(pwd, "../testdata/repos/advanced/skeletons/bar"),
		},
		{
			name:     "file inside of skeleton dir",
			path:     "../testdata/repos/advanced/skeletons/bar/.kickoff.yaml",
			expected: filepath.Join(pwd, "../testdata/repos/advanced/skeletons/bar"),
		},
		{
			name:     "nonexistent file inside of skeleton dir",
			path:     "../testdata/repos/advanced/skeletons/bar/baz/nonexistent.txt",
			expected: filepath.Join(pwd, "../testdata/repos/advanced/skeletons/bar"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := FindSkeletonDir(test.path)
			if test.expectedErr != nil {
				require.Error(t, err)
				assert.Equal(t, test.expectedErr.Error(), err.Error())
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expected, result)
			}
		})
	}
}
