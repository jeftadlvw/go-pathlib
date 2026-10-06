package pathlib

import (
	"cmp"
	"fmt"
	"io/fs"
	"math/rand/v2"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIoDefaults(t *testing.T) {
	t.Parallel()

	require.Equal(t, 0644, defaultOpenPermission, "default open permission mismatch")
	require.Equal(t, "rw", defaultOpenMode, "default open mode mismatch")
}

func TestOpenFile(t *testing.T) {
	t.Parallel()

	t.Run("missing file is created", func(t *testing.T) {
		t.Parallel()

		filePath := setupTempDir(t).JoinStrings("file.txt")

		file, err := OpenFile(filePath)
		require.NoError(t, err)
		require.NotNil(t, file)
		defer func() { _ = file.Close() }()

		require.Equal(t, filePath.String(), file.Name(), "file name is not correct")
		requireDefaultOpenPermission(t, file)
		requireWritable(t, file)
	})

	t.Run("existing directory is refused", func(t *testing.T) {
		t.Parallel()

		dirPath := createTempDir(t, setupTempDir(t), "dir")

		file, err := OpenFile(dirPath)
		require.Error(t, err)
		require.Nil(t, file)
	})

	t.Run("existing file is truncated", func(t *testing.T) {
		t.Parallel()

		filePath := writeTempFile(t, setupTempDir(t), "file.txt", "content")

		file, err := OpenFile(filePath)
		require.NoError(t, err)
		require.NotNil(t, file)
		defer func() { _ = file.Close() }()

		require.Equal(t, filePath.String(), file.Name(), "file name is not correct")
		requireDefaultOpenPermission(t, file)

		content, err := os.ReadFile(filePath.String())
		require.NoError(t, err)
		require.Empty(t, content)

		requireWritable(t, file)
	})

	t.Run("file can be opened twice", func(t *testing.T) {
		t.Parallel()

		filePath := writeTempFile(t, setupTempDir(t), "file.txt", "")

		first, err := OpenFile(filePath)
		require.NoError(t, err)
		require.NotNil(t, first)
		defer func() { _ = first.Close() }()

		second, err := OpenFile(filePath)
		require.NoError(t, err)
		require.NotNil(t, second)
		require.NoError(t, second.Close())
	})
}

// requireDefaultOpenPermission asserts that file has the default permission of
// OpenFile.
func requireDefaultOpenPermission(t *testing.T, file *os.File) {
	t.Helper()

	stats, err := file.Stat()
	require.NoError(t, err, "could not call stat()")
	require.Equal(t, effectiveFileMode(defaultOpenPermission).Perm(), stats.Mode().Perm(), "file permission mismatch")
}

// requireWritable asserts that file can be written to.
func requireWritable(t *testing.T, file *os.File) {
	t.Helper()

	_, err := file.WriteString(generateRandomString(5, 15))
	require.NoError(t, err, "expected file to be writable, but got error")
}

func TestOpenFileWithOptions(t *testing.T) {
	t.Parallel()

	// if false, OpenFile with non-existing path may throw error
	createIfNotExistCases := []bool{
		false, // first value is the default
		false,
		true,
	}

	// these are expected to work
	permissionCases := []os.FileMode{
		000, // first value is the default
		defaultOpenPermission,
		0600,
		0400,
		0440,
		0444,
		0744,
		0700,
		0000,
		744,
		700,
		378,
		211,
	}

	modeCases := []struct {
		Value string
		Ok    bool
	}{
		{Ok: true}, // first value is the default
		{"", true},
		{"r", true},
		{"w", true},
		{"rw", true},
		{"wa", true},
		{"rwa", true},
		{"a", false},
		{"ra", false},
		{"aa", false},
		{"rr", false},
		{"wr", false},
		{"awr", false},
		{"raw", false},
	}

	for createIfNotExistIdx, createIfNotExist := range createIfNotExistCases {
		for permissionIdx, permission := range permissionCases {
			for modeIdx, mode := range modeCases {
				c := openFileCase{
					options: OpenOptions{
						CreateIfNotExists: createIfNotExist,
						Permission:        permission,
						Mode:              mode.Value,
					},
					modeOk:     mode.Ok,
					mode:       cmp.Or(mode.Value, defaultOpenMode),
					permission: cmp.Or(permission, defaultOpenPermission),
				}

				t.Run(fmt.Sprintf("icreate:%d-iperm:%d-imode:%d", createIfNotExistIdx, permissionIdx, modeIdx), func(t *testing.T) {
					t.Parallel()

					runOpenFileCase(t, c)
				})
			}
		}
	}
}

// openFileCase is a combination of options passed to OpenFileWithOptions.
type openFileCase struct {
	// options are passed to OpenFileWithOptions.
	options OpenOptions
	// modeOk is whether options.Mode is supported.
	modeOk bool
	// mode is the open mode that applies, with the default filled in.
	mode string
	// permission is the permission that applies, with the default filled in.
	permission os.FileMode
}

// openFileRun is the state of one run of an openFileCase. Its steps run in
// order on one shared file.
type openFileRun struct {
	openFileCase

	filePath *Path
	contents string
	file     *os.File
}

// runOpenFileCase opens a missing and then an existing file with the options of
// c, and checks the results. Its steps run in order on one shared file.
func runOpenFileCase(t *testing.T, c openFileCase) {
	t.Helper()

	r := &openFileRun{
		openFileCase: c,
		filePath:     setupTempDir(t).JoinStrings("file.txt"),
		contents:     generateRandomString(5, 15),
	}

	t.Run("path is existing directory", func(t *testing.T) {
		err := os.Mkdir(r.filePath.String(), 0777) //nolint:gosec // Fixtures use common permissions.
		require.NoError(t, err)
		defer func() { _ = os.Remove(r.filePath.String()) }()

		file, err := OpenFileWithOptions(r.filePath, r.options)
		require.Error(t, err)
		require.Nil(t, file)
	})

	var err error
	r.file, err = OpenFileWithOptions(r.filePath, r.options)

	// Decimal cases like 744 (0o1350) set the Unix octal sticky bit,
	// which is not fs.ModeSticky and lies outside PermissionBits.
	permissionOutOfBounds := r.permission&^PermissionBits != 0

	t.Run("file did not exist", func(t *testing.T) {
		if r.options.CreateIfNotExists && r.modeOk && !permissionOutOfBounds {
			r.requireOpened(t, err)
			return
		}

		require.Error(t, err, "file may not be created or invalid open mode")
		require.Nil(t, r.file, "called function must return nil")
	})

	// stop test if mode is not okay or permissions out of bounds
	// file and err tested in "file did not exist" test
	if !r.modeOk || permissionOutOfBounds {
		return
	}

	if r.options.CreateIfNotExists {
		require.NoError(t, r.file.Close(), "could not close file")
	} else {
		// create file with fitting permissions and close handle
		r.createFile(t)
	}

	// reopen file to check handling on existing files
	r.file, err = OpenFileWithOptions(r.filePath, r.options)
	defer func() {
		if r.file != nil {
			require.NoError(t, r.file.Close(), "could not close file")
		}
	}()

	t.Run("file existed", func(t *testing.T) {
		r.checkExisting(t, err)
	})

	// The file is closed by the deferred function and removed with its
	// temporary directory.
}

// checkExisting checks the result of opening the existing file, with the
// error err.
func (r *openFileRun) checkExisting(t *testing.T, err error) {
	t.Helper()

	if r.expectPermissionError() {
		require.Error(t, err)
		require.Nil(t, r.file)
		return
	}

	r.requireOpened(t, err)

	if r.modeOk && isMode(r.mode, "w") && r.permission >= 0400 {
		// file is truncated after reopening
		// can only be tested if write mode is set and
		// permissions allow read-access to check the file contents
		t.Run("file contents truncated or appended", r.checkTruncatedOrAppended)
	}

	t.Run("can open file twice", func(t *testing.T) {
		localFile, localErr := OpenFileWithOptions(r.filePath, r.options)
		require.NoError(t, localErr)
		require.NotNil(t, localFile)
		require.NoError(t, localFile.Close())
	})
}

// expectPermissionError reports whether opening the existing file fails,
// because its permission does not allow the open mode.
func (r *openFileRun) expectPermissionError() bool {
	effectivePerm := effectiveFileMode(r.permission)
	return r.modeOk &&
		((isMode(r.mode, "r") && effectivePerm&0400 == 0) ||
			(isMode(r.mode, "w") && effectivePerm&0200 == 0) ||
			(isMode(r.mode, "rw") && effectivePerm&0600 != 0600))
}

// requireOpened checks that the file was opened without the error err, and
// that it is writable exactly if the open mode allows it. The permissions of the
// file are not checked, because they depend on the umask.
func (r *openFileRun) requireOpened(t *testing.T, err error) {
	t.Helper()

	require.NoError(t, err)
	require.NotNil(t, r.file)
	require.Equal(t, r.filePath.String(), r.file.Name(), "file name is not correct")

	t.Run("file writable", func(t *testing.T) {
		_, err := r.file.WriteString(r.contents)
		if r.modeOk && isMode(r.mode, "w") {
			require.NoError(t, err, "file should b writable")
		} else {
			require.Error(t, err, "file should not be writable")
		}
	})
}

// createFile creates the file with the permission and open mode of the case.
func (r *openFileRun) createFile(t *testing.T) {
	t.Helper()

	createOptions := r.options
	createOptions.CreateIfNotExists = true

	createdFile, err := OpenFileWithOptions(r.filePath, createOptions)
	require.NoError(t, err, "could not create file")
	require.NotNil(t, createdFile, "created file must not be nil")
	require.NoError(t, createdFile.Close())
}

// checkTruncatedOrAppended recreates the file, writes to it twice, and checks
// that the second write truncated or appended the content as the open mode
// says. It leaves the file of the run open again.
func (r *openFileRun) checkTruncatedOrAppended(t *testing.T) {
	require.NoError(t, r.file.Close(), "could not close file")

	// recreate file
	require.NoError(t, os.Remove(r.filePath.String()), "could not remove file")
	r.createFile(t)

	// write initial content twice
	for range 2 {
		localFile, err := OpenFileWithOptions(r.filePath, r.options)
		require.NoError(t, err)
		require.NotNil(t, localFile)

		_, err = localFile.WriteString(r.contents)
		require.NoError(t, err)

		require.NoError(t, localFile.Close(), "could not close file")
	}

	readFileContents, err := os.ReadFile(r.filePath.String())
	require.NoError(t, err)

	if isMode(r.mode, "a") {
		require.Equal(t, r.contents+r.contents, string(readFileContents))
	} else {
		require.Equal(t, r.contents, string(readFileContents))
	}

	// restore previously closed file
	r.file, err = OpenFileWithOptions(r.filePath, r.options)
	require.NoError(t, err)
	require.NotNil(t, r.file)

	_, err = r.file.Stat()
	require.NoError(t, err)
}

//nolint:paralleltest // The steps of a case run in order on one shared file.
func TestReadWriteAppendOperations(t *testing.T) {
	t.Parallel()

	// predefined strings
	predefinedStrings := []string{
		"Hello, world!",
		"The quick brown fox jumps over the lazy dog.",
		"1234567890",
		"Special characters: !@#$%^&*()",
		"Line 1\nLine 2\nLine 3",
		"Lorem ipsum dolor sit amet",
		"Unicode characters: 你好, こんにちは, Привет",
		"",
		"Go is a statically typed, compiled programming language.",
		"This is the final test string.",
	}

	// add random strings
	const randomStringCount = 5
	testStrings := make([]string, 0, len(predefinedStrings)+randomStringCount)
	testStrings = append(testStrings, predefinedStrings...)
	for range randomStringCount {
		testStrings = append(testStrings, generateRandomString(5, 80))
	}

	// combine strings randomly
	type TestCombinationCase struct {
		First  string
		Second string
	}

	testCombinationCases := make([]TestCombinationCase, 20)
	//nolint:gosec // Test inputs need no secure randomness.
	for i := range testCombinationCases {
		testCombinationCases[i] = TestCombinationCase{
			First:  testStrings[rand.IntN(len(testStrings))],
			Second: testStrings[rand.IntN(len(testStrings))],
		}
	}

	for idx, combination := range testCombinationCases {
		t.Run(strconv.Itoa(idx), func(t *testing.T) {
			t.Parallel()

			// Create a temporary path for testing
			tempFile, err := os.CreateTemp(t.TempDir(), "pathlib_io_test")
			require.NoError(t, err)

			tempFilePathStr := tempFile.Name()
			filePath := NewPath(tempFilePathStr)

			err = tempFile.Close()
			require.NoError(t, err)

			appendBytes := []byte(combination.Second)
			byteData := []byte(combination.First)

			t.Run("WriteString", func(t *testing.T) {
				written, err := WriteString(filePath, combination.First)
				require.NoError(t, err)
				require.Equal(t, len(combination.First), written)
			})

			t.Run("ReadFileToString", func(t *testing.T) {
				content, err := ReadFileToString(filePath)
				require.NoError(t, err)
				require.Equal(t, combination.First, content)
			})

			t.Run("AppendString", func(t *testing.T) {
				written, err := AppendString(filePath, combination.Second)
				require.NoError(t, err)
				require.Equal(t, len(combination.Second), written)
			})

			t.Run("Verify string appending", func(t *testing.T) {
				content, err := ReadFileToString(filePath)
				require.NoError(t, err)
				require.Equal(t, combination.First+combination.Second, content)
			})

			t.Run("WriteBytes", func(t *testing.T) {
				byteData := []byte(combination.First)
				written, err := WriteBytes(filePath, byteData)
				require.NoError(t, err)
				require.Equal(t, len(byteData), written)
			})

			t.Run("ReadFile", func(t *testing.T) {
				readBytes, err := ReadFile(filePath)
				require.NoError(t, err)
				require.Equal(t, byteData, readBytes)
			})

			t.Run("AppendBytes", func(t *testing.T) {
				written, err := AppendBytes(filePath, appendBytes)
				require.NoError(t, err)
				require.Equal(t, len(appendBytes), written)
			})

			t.Run("Verify byte appending", func(t *testing.T) {
				readBytes, err := ReadFile(filePath)
				require.NoError(t, err)
				require.Equal(t, append(byteData, appendBytes...), readBytes)
			})
		})
	}
}

func isMode(modeStr, requiredMode string) bool {
	return strings.Contains(modeStr, requiredMode)
}

func TestIoErrorsAreWrapped(t *testing.T) {
	t.Parallel()

	root := setupTempDir(t)
	missing := root.JoinStrings("missing.txt")

	_, err := ReadFile(missing)
	require.ErrorIs(t, err, ErrRead)
	require.ErrorIs(t, err, fs.ErrNotExist)
	require.ErrorAs(t, err, new(*PathlibError))

	_, err = OpenFileWithOptions(missing, OpenOptions{Mode: "r"})
	require.ErrorIs(t, err, ErrOpen)
	require.ErrorIs(t, err, ErrAccess)
	require.ErrorIs(t, err, fs.ErrNotExist)
	require.ErrorAs(t, err, new(*PathlibError))
}

// writeFuncs returns all write functions with the behavior they share: writing
// to a missing file creates it, and a symlink to a file is written through.
func writeFuncs() map[string]func(*Path, string) (int, error) {
	return map[string]func(*Path, string) (int, error){
		"WriteBytes":  func(p *Path, s string) (int, error) { return WriteBytes(p, []byte(s)) },
		"WriteString": WriteString,
		"WriteBytesWithOptions": func(p *Path, s string) (int, error) {
			return WriteBytesWithOptions(p, []byte(s), FileOptions{ExistOk: true})
		},
		"WriteStringWithOptions": func(p *Path, s string) (int, error) {
			return WriteStringWithOptions(p, s, FileOptions{ExistOk: true})
		},
		"AppendBytes":  func(p *Path, s string) (int, error) { return AppendBytes(p, []byte(s)) },
		"AppendString": AppendString,
	}
}

func TestWrite_CreatesMissingFile(t *testing.T) {
	t.Parallel()

	for name, write := range writeFuncs() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			file := setupTempDir(t).JoinStrings("new.txt")

			n, err := write(file, "content")
			require.NoError(t, err)
			require.Equal(t, len("content"), n)

			content, err := os.ReadFile(file.String())
			require.NoError(t, err)
			require.Equal(t, "content", string(content))

			info, err := file.Stat()
			require.NoError(t, err)
			require.Equal(t, DefaultFileMode().Perm(), info.Mode().Perm())
		})
	}
}

func TestWrite_SymlinkToFileIsWrittenThrough(t *testing.T) {
	t.Parallel()

	for name, write := range writeFuncs() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			root := setupTempDir(t)
			target := writeTempFile(t, root, "target.txt", "")
			link := createTempSymlinkAbs(t, root, "target.txt", "link")

			_, err := write(link, "content")
			require.NoError(t, err)
			require.True(t, link.IsSymlink(), "the link is kept")

			content, err := os.ReadFile(target.String())
			require.NoError(t, err)
			require.Equal(t, "content", string(content))
		})
	}
}

func TestWrite_NonFilePathIsRefused(t *testing.T) {
	t.Parallel()

	for name, write := range writeFuncs() {
		t.Run(name+"/directory", func(t *testing.T) {
			t.Parallel()

			dir := createTempDir(t, setupTempDir(t), "dir")
			_, err := write(dir, "content")
			require.ErrorIs(t, err, ErrNotFile)
		})

		t.Run(name+"/broken symlink", func(t *testing.T) {
			t.Parallel()

			root := setupTempDir(t)
			link := createTempSymlinkAbs(t, root, "target.txt", "link")

			_, err := write(link, "content")
			require.ErrorIs(t, err, ErrNotFile)
			requireLExists(t, false, root.JoinStrings("target.txt"), "the link target is not created")
		})
	}
}

func TestWriteBytesWithOptions(t *testing.T) {
	t.Parallel()

	t.Run("Mode applies to a created file", func(t *testing.T) {
		t.Parallel()

		file := setupTempDir(t).JoinStrings("new.txt")

		_, err := WriteBytesWithOptions(file, []byte("content"), FileOptions{Mode: 0600})
		require.NoError(t, err)

		info, err := file.Stat()
		require.NoError(t, err)
		require.Equal(t, effectiveFileMode(0600).Perm(), info.Mode().Perm())
	})

	t.Run("Mode does not change an existing file", func(t *testing.T) {
		t.Parallel()

		file := writeTempFile(t, setupTempDir(t), "file.txt", "old")

		_, err := WriteBytesWithOptions(file, []byte("new"), FileOptions{ExistOk: true, Mode: 0600})
		require.NoError(t, err)

		info, err := file.Stat()
		require.NoError(t, err)
		require.Equal(t, effectiveFileMode(0644).Perm(), info.Mode().Perm())
	})

	t.Run("ExistOk=true truncates an existing file", func(t *testing.T) {
		t.Parallel()

		file := writeTempFile(t, setupTempDir(t), "file.txt", "old content")

		_, err := WriteStringWithOptions(file, "new", FileOptions{ExistOk: true})
		require.NoError(t, err)

		content, err := ReadFileToString(file)
		require.NoError(t, err)
		require.Equal(t, "new", content)
	})

	t.Run("ExistOk=false refuses an existing file", func(t *testing.T) {
		t.Parallel()

		file := writeTempFile(t, setupTempDir(t), "file.txt", "old content")

		n, err := WriteStringWithOptions(file, "new", FileOptions{ExistOk: false})
		require.ErrorIs(t, err, ErrFileExist)
		require.Zero(t, n)

		content, err := ReadFileToString(file)
		require.NoError(t, err)
		require.Equal(t, "old content", content, "the file is left untouched")
	})

	t.Run("ExistOk=false creates a missing file", func(t *testing.T) {
		t.Parallel()

		file := setupTempDir(t).JoinStrings("new.txt")

		_, err := WriteStringWithOptions(file, "content", FileOptions{ExistOk: false})
		require.NoError(t, err)

		content, err := ReadFileToString(file)
		require.NoError(t, err)
		require.Equal(t, "content", content)
	})
}

func TestAppendBytes_AppendsToExistingFile(t *testing.T) {
	t.Parallel()

	file := writeTempFile(t, setupTempDir(t), "file.txt", "old")

	_, err := AppendString(file, "+new")
	require.NoError(t, err)

	content, err := ReadFileToString(file)
	require.NoError(t, err)
	require.Equal(t, "old+new", content)
}
