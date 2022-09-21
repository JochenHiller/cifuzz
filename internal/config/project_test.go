package config

import (
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/hectane/go-acl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code-intelligence.com/cifuzz/util/fileutil"
)

var baseTempDir string

func TestMain(m *testing.M) {
	var err error
	baseTempDir, err = os.MkdirTemp("", "project-config-test-")
	if err != nil {
		log.Fatalf("Failed to create temp dir for tests: %+v", err)
	}
	defer fileutil.Cleanup(baseTempDir)

	m.Run()
}

func TestCreateProjectConfig(t *testing.T) {
	projectDir, err := os.MkdirTemp(baseTempDir, "project-")
	require.NoError(t, err)
	defer fileutil.Cleanup(projectDir)

	path, err := CreateProjectConfig(projectDir)
	assert.NoError(t, err)
	expectedPath := filepath.Join(projectDir, "cifuzz.yaml")
	assert.Equal(t, expectedPath, path)

	// file created?
	exists, err := fileutil.Exists(expectedPath)
	assert.NoError(t, err)
	assert.True(t, exists)

	// check for content
	content, err := os.ReadFile(expectedPath)
	assert.NoError(t, err)
	assert.NotEmpty(t, content)
	assert.Contains(t, string(content), "Configuration for")

}

// Should return error if not allowed to write to directory
func TestCreateProjectConfig_NoPerm(t *testing.T) {
	// create read only project dir
	projectDir, err := os.MkdirTemp(baseTempDir, "project-")
	require.NoError(t, err)
	defer fileutil.Cleanup(projectDir)

	err = acl.Chmod(projectDir, 0555)
	require.NoError(t, err)

	path, err := CreateProjectConfig(projectDir)
	assert.Error(t, err)
	assert.ErrorIs(t, err, os.ErrPermission)
	assert.Empty(t, path)

	// file should not exists
	exists, err := fileutil.Exists("cifuzz.yaml")
	assert.NoError(t, err)
	assert.False(t, exists)
}

// Should return error if file already exists
func TestCreateProjectConfig_Exists(t *testing.T) {
	projectDir, err := os.MkdirTemp(baseTempDir, "project-")
	require.NoError(t, err)
	defer fileutil.Cleanup(projectDir)

	existingPath := filepath.Join(projectDir, "cifuzz.yaml")
	err = os.WriteFile(existingPath, []byte{}, 0644)
	require.NoError(t, err)

	path, err := CreateProjectConfig(filepath.Dir(existingPath))
	assert.Error(t, err)
	// check if path of the existing config is return and the error indicates it too
	assert.ErrorIs(t, err, os.ErrExist)
	assert.Equal(t, existingPath, path)

	// file should not exists
	exists, err := fileutil.Exists(existingPath)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestReadProjectConfig(t *testing.T) {
	projectDir, err := os.MkdirTemp(baseTempDir, "project-")
	require.NoError(t, err)
	defer fileutil.Cleanup(projectDir)

	configFile := filepath.Join(projectDir, "cifuzz.yaml")
	err = os.WriteFile(configFile, []byte("build-system: "), 0644)
	require.NoError(t, err)

	config, err := ReadProjectConfig(projectDir)
	require.NoError(t, err)

	require.Equal(t, BuildSystemOther, config.BuildSystem)
}

func TestReadProjectConfigCMake(t *testing.T) {
	projectDir, err := os.MkdirTemp(baseTempDir, "project-")
	require.NoError(t, err)
	defer fileutil.Cleanup(projectDir)

	configFile := filepath.Join(projectDir, "cifuzz.yaml")
	err = os.WriteFile(configFile, []byte("build-system: "), 0644)
	require.NoError(t, err)

	// Create a CMakeLists.txt in the project dir, which should cause
	// the build system to be detected as CMake
	err = os.WriteFile(filepath.Join(projectDir, "CMakeLists.txt"), []byte{}, 0644)
	require.NoError(t, err)

	config, err := ReadProjectConfig(projectDir)
	require.NoError(t, err)

	require.Equal(t, BuildSystemCMake, config.BuildSystem)
}

func TestDetermineBuildSystem_CMake(t *testing.T) {
	projectDir, err := os.MkdirTemp(baseTempDir, "project-")
	require.NoError(t, err)
	defer fileutil.Cleanup(projectDir)

	err = os.WriteFile(filepath.Join(projectDir, "CMakeLists.txt"), []byte{}, 0644)
	buildSystem, err := DetermineBuildSystem(projectDir)
	require.NoError(t, err)
	assert.Equal(t, BuildSystemCMake, buildSystem)
}

func TestDetermineBuildSystem_Maven(t *testing.T) {
	projectDir, err := os.MkdirTemp(baseTempDir, "project-")
	require.NoError(t, err)
	defer fileutil.Cleanup(projectDir)

	err = os.WriteFile(filepath.Join(projectDir, "pom.xml"), []byte{}, 0644)
	buildSystem, err := DetermineBuildSystem(projectDir)
	require.NoError(t, err)
	assert.Equal(t, BuildSystemMaven, buildSystem)
}

func TestDetermineBuildSystem_GradleGroovy(t *testing.T) {
	projectDir, err := os.MkdirTemp(baseTempDir, "project-")
	require.NoError(t, err)
	defer fileutil.Cleanup(projectDir)

	err = os.WriteFile(filepath.Join(projectDir, "build.gradle"), []byte{}, 0644)
	buildSystem, err := DetermineBuildSystem(projectDir)
	require.NoError(t, err)
	assert.Equal(t, BuildSystemGradle, buildSystem)
}

func TestDetermineBuildSystem_GradleKotlin(t *testing.T) {
	projectDir, err := os.MkdirTemp(baseTempDir, "project-")
	require.NoError(t, err)
	defer fileutil.Cleanup(projectDir)

	err = os.WriteFile(filepath.Join(projectDir, "build.gradle.kts"), []byte{}, 0644)
	buildSystem, err := DetermineBuildSystem(projectDir)
	require.NoError(t, err)
	assert.Equal(t, BuildSystemGradle, buildSystem)
}

func TestDetermineBuildSystem_Other(t *testing.T) {
	projectDir, err := os.MkdirTemp(baseTempDir, "project-")
	require.NoError(t, err)
	defer fileutil.Cleanup(projectDir)

	buildSystem, err := DetermineBuildSystem(projectDir)
	require.NoError(t, err)
	assert.Equal(t, BuildSystemOther, buildSystem)
}
