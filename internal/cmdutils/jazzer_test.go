package cmdutils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code-intelligence.com/cifuzz/internal/config"
	"code-intelligence.com/cifuzz/pkg/stubs"
	"code-intelligence.com/cifuzz/util/fileutil"
)

func TestListJVMFuzzTests(t *testing.T) {
	projectDir, err := os.MkdirTemp("", "list-jvm-files")
	require.NoError(t, err)
	defer fileutil.Cleanup(projectDir)

	testDir := filepath.Join(projectDir, "src", "test")

	// create some java files including one valid fuzz test
	javaDir := filepath.Join(testDir, "java", "com", "example")
	err = os.MkdirAll(javaDir, 0o755)
	require.NoError(t, err)
	err = stubs.Create(filepath.Join(javaDir, "FuzzTestCase1.java"), config.Java)
	require.NoError(t, err)
	_, err = os.Create(filepath.Join(javaDir, "UnitTestCase.java"))
	require.NoError(t, err)
	javaDirToFilter := filepath.Join(testDir, "java", "com", "filter", "me")
	err = os.MkdirAll(javaDirToFilter, 0o755)
	require.NoError(t, err)
	err = stubs.Create(filepath.Join(javaDirToFilter, "FuzzTestCase2.java"), config.Java)
	require.NoError(t, err)

	// create some kotlin files including one valid fuzz test
	kotlinDir := filepath.Join(testDir, "kotlin", "com", "example")
	err = os.MkdirAll(kotlinDir, 0o755)
	require.NoError(t, err)
	err = stubs.Create(filepath.Join(kotlinDir, "FuzzTestCase3.kt"), config.Kotlin)
	require.NoError(t, err)
	_, err = os.Create(filepath.Join(kotlinDir, "UnitTestCase.kt"))
	require.NoError(t, err)

	// create some extra files
	resDir := filepath.Join(testDir, "resources")
	err = os.MkdirAll(resDir, 0o755)
	require.NoError(t, err)
	_, err = os.Create(filepath.Join(resDir, "SomeTestData"))
	require.NoError(t, err)

	// Check result
	result, err := ListJVMFuzzTestsWithFilter(projectDir, "com.example")
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Contains(t, result, "com.example.FuzzTestCase1")
	assert.Contains(t, result, "com.example.FuzzTestCase3")

	// Check result without filter
	result, err = ListJVMFuzzTests(projectDir)
	require.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Contains(t, result, "com.example.FuzzTestCase1")
	assert.Contains(t, result, "com.filter.me.FuzzTestCase2")
	assert.Contains(t, result, "com.example.FuzzTestCase3")
}

func TestGetTargetMethodsFromJVMFuzzTestFileSingleMethod(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "jazzer-*")
	require.NoError(t, err)
	defer fileutil.Cleanup(tempDir)
	require.NoError(t, err)

	path := filepath.Join(tempDir, "FuzzTest1.java")
	err = os.WriteFile(path, []byte(`
package com.example;

import com.code_intelligence.jazzer.junit.FuzzTest;

class FuzzTest {
    @FuzzTest
    public static void fuzz(byte[] data) {}
}
`), 0o644)
	require.NoError(t, err)

	result, err := GetTargetMethodsFromJVMFuzzTestFile(path)
	require.NoError(t, err)
	assert.Equal(t, []string{"fuzz"}, result)

	path = filepath.Join(tempDir, "FuzzTest2.java")
	err = os.WriteFile(path, []byte(`
package com.example;

import com.code_intelligence.jazzer.junit.FuzzTest;

class FuzzTest {
    public static void fuzzerTestOneInput(byte[] data) {}
}
`), 0o644)
	require.NoError(t, err)

	result, err = GetTargetMethodsFromJVMFuzzTestFile(path)
	require.NoError(t, err)
	assert.Equal(t, []string{"fuzzerTestOneInput"}, result)
}

func TestGetTargetMethodsFromJVMFuzzTestFileMultipleMethods(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "jazzer-*")
	require.NoError(t, err)
	defer fileutil.Cleanup(tempDir)
	require.NoError(t, err)

	path := filepath.Join(tempDir, "FuzzTest.java")
	err = os.WriteFile(path, []byte(`
package com.example;

import com.code_intelligence.jazzer.junit.FuzzTest;

class FuzzTest {
    @FuzzTest
    public static void fuzz(byte[] data) {}

	@FuzzTest
	public static void fuzz2(byte[] data) {}
}
`), 0o644)
	require.NoError(t, err)

	result, err := GetTargetMethodsFromJVMFuzzTestFile(path)
	require.NoError(t, err)
	assert.Equal(t, []string{"fuzz", "fuzz2"}, result)
}
