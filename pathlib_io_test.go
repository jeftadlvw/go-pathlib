package pathlib

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"math/rand/v2"
	"os"
	"strings"
	"testing"
)

func TestIoDefaults(t *testing.T) {
	require.Equal(t, 0644, defaultOpenPermission, "default open permission mismatch")
	require.Equal(t, "rw", defaultOpenMode, "default open mode mismatch")
}

func TestOpenFile(t *testing.T) {
	fileNameStr := generateRandomString(5, 15)
	filePath := NewPath(fileNameStr)

	// should return error, because file does not exist
	t.Run("ensure random file name does not exist", func(t *testing.T) {
		_, err := os.OpenFile(filePath.String(), os.O_RDONLY, defaultOpenPermission)
		require.Error(t, err)
	})

	t.Run("path is existing directory", func(t *testing.T) {
		err := os.Mkdir(filePath.String(), 0777)
		require.NoError(t, err)
		defer os.Remove(filePath.String())

		file, err := OpenFile(filePath)
		require.Error(t, err)
		require.Nil(t, file)
	})

	file, err := OpenFile(filePath)
	randomFileContents := generateRandomString(5, 15)

	filePermissionTest := func(t *testing.T, file *os.File) {
		stats, err := file.Stat()
		require.NoError(t, err, "could not call stat()")

		require.Equal(t, uint32(defaultOpenPermission), uint32(stats.Mode().Perm()), "file permission mismatch")
	}

	fileWritableTest := func(t *testing.T, file *os.File) {
		_, writeErr := file.WriteString(randomFileContents)
		require.NoError(t, writeErr, "expected file to be writable, but got error")
	}

	t.Run("file did not exist", func(t *testing.T) {
		// no errors
		require.NoError(t, err)
		require.NotNil(t, file)

		// file checks
		require.Equal(t, fileNameStr, file.Name(), "file name is not correct")
		filePermissionTest(t, file)
		fileWritableTest(t, file)
	})

	// reopen file to check handling on existing files
	require.NoError(t, file.Close(), "could not close file")
	file, err = OpenFile(filePath)

	t.Run("file existed", func(t *testing.T) {
		require.NoError(t, err)
		require.NotNil(t, file)

		require.Equal(t, fileNameStr, file.Name(), "file name is not correct")
		filePermissionTest(t, file)

		// file is truncated after reopening
		t.Run("file contents truncated", func(t *testing.T) {
			readFileContents, err := os.ReadFile(file.Name())
			require.NoError(t, err)
			require.Empty(t, readFileContents)
		})

		// test if file is still writable
		fileWritableTest(t, file)
	})

	t.Run("can open file twice", func(t *testing.T) {
		localFile, localErr := OpenFile(filePath)
		require.NoError(t, localErr)
		require.NotNil(t, localFile)
		require.NoError(t, localFile.Close())
	})

	require.NoError(t, file.Close(), "could not close file")
	require.NoError(t, os.Remove(filePath.String()), "could not remove file")
}

func TestOpenFileWithOptions(t *testing.T) {

	// if false, OpenFile with non-existing path may throw error
	createIfNotExistCases := []bool{
		false, // first value is ignored
		false,
		true,
	}

	// these are expected to work
	permissionCases := []os.FileMode{
		000, // first value is ignored
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
		{Ok: true}, // first value is ignored
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
		for permissionIdx, expectedPermission := range permissionCases {
			for modeIdx, expectedMode := range modeCases {

				// build option struct
				testOptions := OpenOptions{}

				if createIfNotExistIdx > 0 {
					testOptions.CreateIfNotExists = createIfNotExist
				}

				if permissionIdx > 0 {
					testOptions.Permission = expectedPermission
				}

				if modeIdx > 0 {
					testOptions.Mode = expectedMode.Value
				}

				// default values
				fileNameStr := generateRandomString(15, 20)
				filePath := NewPath(fileNameStr)

				t.Run(fmt.Sprintf("icreate:%d-iperm:%d-imode:%d", createIfNotExistIdx, permissionIdx, modeIdx), func(t *testing.T) {
					// should return error, because file does not exist
					t.Run("ensure random file name does not exist", func(t *testing.T) {
						_, err := os.OpenFile(filePath.String(), os.O_RDONLY, defaultOpenPermission)
						require.Error(t, err)
					})

					t.Run("path is existing directory", func(t *testing.T) {
						err := os.Mkdir(filePath.String(), 0777)
						require.NoError(t, err)
						defer os.Remove(filePath.String())

						file, err := OpenFileWithOptions(filePath, testOptions)
						require.Error(t, err)
						require.Nil(t, file)
					})

					file, err := OpenFileWithOptions(filePath, testOptions)
					defer func() {
						_ = os.Remove(filePath.String())
					}()

					randomFileContents := generateRandomString(5, 15)

					localModeCaseValue := expectedMode.Value
					if modeIdx == 0 || expectedMode.Value == "" {
						localModeCaseValue = defaultOpenMode
					}

					localPermissionCase := expectedPermission // mask with maximum permission
					if expectedPermission == 0 {
						localPermissionCase = defaultOpenPermission
					}

					// os.FileMode is uint32, so the < 0 check is always false.
					// It is kept for clarity to document the intended bounds check.
					localPermissionCaseOutOfBounds := localPermissionCase < 0 || localPermissionCase > 0777

					filePermissionTest := func(t *testing.T, file *os.File) {
						t.Run("file permissions", func(t *testing.T) {

							// This should be tested, but it's difficult because of varying umasks.

							/*
								stats, err := file.Stat()
								require.NoError(t, err, "could not call stat()")

								fmt.Println(fmt.Sprintf("%O", localPermissionCase))
								fmt.Println(fmt.Sprintf("%O", stats.Mode().Perm()))

								require.Equal(t, uint32(localPermissionCase), uint32(stats.Mode().Perm()), "file permission mismatch")
							*/
						})
					}

					fileWritableTest := func(t *testing.T, file *os.File) {
						t.Run("file writable", func(t *testing.T) {
							_, err = file.WriteString(randomFileContents)
							if expectedMode.Ok && isMode(localModeCaseValue, "w") {
								require.NoError(t, err, "file should b writable")
							} else {
								require.Error(t, err, "file should not be writable")
							}
						})
					}

					t.Run("file did not exist", func(t *testing.T) {
						if createIfNotExist && expectedMode.Ok && !localPermissionCaseOutOfBounds {
							// no errors
							require.NoError(t, err)
							require.NotNil(t, file)

							// file checks
							require.Equal(t, fileNameStr, file.Name(), "file name is not correct")
							filePermissionTest(t, file)
							fileWritableTest(t, file)
						} else {
							require.Error(t, err, "file may not be created or invalid open mode")
							require.Nil(t, file, "called function must return nil")
						}
					})

					// stop test if mode is not okay or permissions out of bounds
					// file and err tested in "file did not exist" test
					if !expectedMode.Ok || localPermissionCaseOutOfBounds {
						return
					}

					if createIfNotExist {
						require.NoError(t, file.Close(), "could not close file")
					} else {
						localOptions := OpenOptions{
							true,
							testOptions.Permission,
							testOptions.Mode,
						}

						// create file with fitting permissions and close handle
						createdFile, creationErr := OpenFileWithOptions(filePath, localOptions)
						require.NoError(t, creationErr, "could not create file")
						require.NotNil(t, createdFile, "created file must not be nil")
						require.NoError(t, createdFile.Close())
					}

					// reopen file to check handling on existing files
					file, err = OpenFileWithOptions(filePath, testOptions)
					defer func() {
						if file != nil {
							require.NoError(t, file.Close(), "could not close file")
						}
					}()

					t.Run("file existed", func(t *testing.T) {
						if expectedMode.Ok && ((isMode(localModeCaseValue, "r") && localPermissionCase < 0400) ||
							(isMode(localModeCaseValue, "w") && (localPermissionCase < 0200 || (localPermissionCase >= 0400 && localPermissionCase < 0600))) ||
							(isMode(localModeCaseValue, "rw") && localPermissionCase < 0600)) {
							require.Error(t, err)
							require.Nil(t, file)
							return
						}

						require.NoError(t, err)
						require.NotNil(t, file)

						require.Equal(t, fileNameStr, file.Name(), "file name is not correct")

						filePermissionTest(t, file)
						fileWritableTest(t, file)

						if expectedMode.Ok && isMode(localModeCaseValue, "w") && localPermissionCase >= 0400 {
							// file is truncated after reopening
							// can only be tested if write mode is set and
							// permissions allow read-access to check the file contents

							t.Run("file contents truncated or appended", func(t *testing.T) {
								err := file.Close()
								require.NoError(t, err, "could not close file")

								// recreate file
								err = os.Remove(filePath.String())
								require.NoError(t, err, "could not remove file")

								createFile, err := OpenFileWithOptions(filePath, OpenOptions{
									CreateIfNotExists: true,
									Permission:        expectedPermission,
									Mode:              expectedMode.Value,
								})
								require.NoError(t, err)
								require.NotNil(t, createFile)

								err = createFile.Close()
								require.NoError(t, err)

								// write initial content twice
								for i := 0; i < 2; i++ {
									localFile, err := OpenFileWithOptions(filePath, testOptions)
									require.NoError(t, err)
									require.NotNil(t, localFile)

									_, err = localFile.WriteString(randomFileContents)
									require.NoError(t, err)

									err = localFile.Close()
									require.NoError(t, err, "could not close file")

								}

								readFileContents, err := os.ReadFile(filePath.String())
								readFileContentsStr := string(readFileContents)

								if isMode(localModeCaseValue, "a") {
									require.Equal(t, randomFileContents+randomFileContents, readFileContentsStr)
								} else {
									require.Equal(t, randomFileContents, readFileContentsStr)
								}

								// restore previously closed file
								file, err = OpenFileWithOptions(filePath, testOptions)
								require.NoError(t, err)
								require.NotNil(t, file)

								_, err = file.Stat()
								require.NoError(t, err)
							})
						}

						t.Run("can open file twice", func(t *testing.T) {
							localFile, localErr := OpenFileWithOptions(filePath, testOptions)
							require.NoError(t, localErr)
							require.NotNil(t, localFile)
							require.NoError(t, localFile.Close())
						})
					})

					// file is closed and removed by upper defer statements
				})
			}
		}
	}
}

func TestReadWriteAppendOperations(t *testing.T) {

	// predefined strings
	testStrings := []string{
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
	for i := 0; i < 5; i++ {
		testStrings = append(testStrings, generateRandomString(5, 80))
	}

	// combine strings randomly
	type TestCombinationCase struct {
		First  string
		Second string
	}

	var testCombinationCases []TestCombinationCase
	for i := 0; i < 20; i++ {
		testCombinationCases = append(testCombinationCases, TestCombinationCase{
			First:  testStrings[rand.IntN(len(testStrings))],
			Second: testStrings[rand.IntN(len(testStrings))],
		})
	}

	for idx, combination := range testCombinationCases {
		t.Run(fmt.Sprint(idx), func(t *testing.T) {
			// Create a temporary path for testing
			tempFile, err := os.CreateTemp("", "pathlib_io_test")
			require.NoError(t, err)

			tempFilePathStr := tempFile.Name()
			filePath := NewPath(tempFilePathStr)

			err = tempFile.Close()
			require.NoError(t, err)
			defer os.Remove(tempFilePathStr)

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
