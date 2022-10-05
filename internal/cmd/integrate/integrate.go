package integrate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	copy2 "github.com/otiai10/copy"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"code-intelligence.com/cifuzz/internal/cmdutils"
	"code-intelligence.com/cifuzz/internal/config"
	"code-intelligence.com/cifuzz/pkg/log"
	"code-intelligence.com/cifuzz/pkg/runfiles"
	"code-intelligence.com/cifuzz/util/fileutil"
	"code-intelligence.com/cifuzz/util/stringutil"
)

type integrateCmd struct {
	*cobra.Command

	tools []string
}

func supportedTools() []string {
	return []string{"git", "cmake", "vscode"}
}

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "integrate <git|cmake|vscode>",
		Short: "Add integrations for the following tools: Git, CMake, VS Code",
		Long: `Add integrations for Git, CMake and VS Code:

Add files generated by cifuzz to your .gitignore:

    cifuzz integrate git

Add CMake presets to your CMakeUserPresets.json. Those presets simplify
the execution of regression tests from the command line and provide
integration with IDEs such as CLion and VS Code:

    cifuzz integrate cmake

Provide integration for coverage runs from within VS Code by adding
tasks to your tasks.json:

    cifuzz integrate vscode

Missing files are generated automatically.
`,
		ValidArgs: supportedTools(),
		Args:      cobra.MatchAll(cobra.RangeArgs(1, len(supportedTools())), cobra.OnlyValidArgs),
		RunE: func(c *cobra.Command, args []string) error {
			cmd := integrateCmd{
				Command: c,
				tools:   args,
			}

			return cmd.run()
		},
	}

	return cmd
}

func (c *integrateCmd) run() error {
	var err error

	projectDir, err := config.FindConfigDir()
	if errors.Is(err, os.ErrNotExist) {
		// The project directory doesn't exist, this is an expected
		// error, so we print it and return a silent error to avoid
		// printing a stack trace
		log.Error(err, fmt.Sprintf("%s\nUse 'cifuzz init' to set up a project for use with cifuzz.", err.Error()))
		return cmdutils.ErrSilent
	}
	if err != nil {
		return err
	}

	for _, tool := range c.tools {
		switch tool {
		case "git":
			err = setupGitIgnore(projectDir)
			if err != nil {
				return err
			}
		case "cmake":
			err = setupCMakePresets(projectDir, runfiles.Finder)
			if err != nil {
				return err
			}
		case "vscode":
			err = setupVSCodeTasks(projectDir, runfiles.Finder)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func setupGitIgnore(projectDir string) error {
	// Files to ignore for all build systems
	filesToIgnore := []string{
		"/.cifuzz-corpus/",
		"/.cifuzz-findings/",
	}

	buildSystem, err := config.DetermineBuildSystem(projectDir)
	if err != nil {
		return err
	}
	if buildSystem == config.BuildSystemCMake {
		filesToIgnore = append(filesToIgnore,
			"/.cifuzz-build/",
			"/CMakeUserPresets.json",
		)
	}

	gitIgnorePath := filepath.Join(projectDir, ".gitignore")
	hasGitIgnore, err := fileutil.Exists(gitIgnorePath)
	if err != nil {
		return err
	}

	if !hasGitIgnore {
		err = os.WriteFile(gitIgnorePath, []byte(strings.Join(filesToIgnore, "\n")), 0644)
		if err != nil {
			return errors.WithStack(err)
		}
	} else {
		bytes, err := os.ReadFile(gitIgnorePath)
		if err != nil {
			return errors.WithStack(err)
		}
		existingFilesToIgnore := stringutil.NonEmpty(strings.Split(string(bytes), "\n"))

		gitIgnore, err := os.OpenFile(gitIgnorePath, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return errors.WithStack(err)
		}
		defer gitIgnore.Close()

		for _, fileToIgnore := range filesToIgnore {
			if !stringutil.Contains(existingFilesToIgnore, fileToIgnore) {
				_, err = gitIgnore.WriteString(fileToIgnore + "\n")
				if err != nil {
					return errors.WithStack(err)
				}
			}
		}
	}
	log.Printf(`
Added files generated by cifuzz to .gitignore.`)

	return nil
}

func setupVSCodeTasks(projectDir string, finder runfiles.RunfilesFinder) error {
	tasksSrcPath, err := finder.VSCodeTasksPath()
	if err != nil {
		return err
	}
	tasksDestPath := filepath.Join(projectDir, ".vscode", "tasks.json")
	hasTasks, err := fileutil.Exists(tasksDestPath)
	if err != nil {
		return err
	}

	if !hasTasks {
		// Situation: The user doesn't have a tasks.json file set up and
		// may thus be unaware of this functionality. Create one and tell
		// them about it.
		err = copy2.Copy(tasksSrcPath, tasksDestPath)
		if err != nil {
			return errors.WithStack(err)
		}
		log.Printf(`
tasks.json has been created in .vscode to provide easy access to command
line workflows. It enables you to launch coverage runs from within
VS Code. You can use the Coverage Gutters extension to visualize the
generated coverage report. To learn more about tasks in VS Code, visit:

	https://code.visualstudio.com/docs/editor/tasks

You can download the Coverage Gutters extension from:

	https://marketplace.visualstudio.com/items?itemName=ryanluker.vscode-coverage-gutters`)
	} else {
		// Situation: The user does have a tasks.json file set up, so we
		// assume them to know about the benefits. We suggest to the user
		// that they add our task to the existing tasks.json.
		presetsSrc, err := os.ReadFile(tasksSrcPath)
		if err != nil {
			return errors.WithStack(err)
		}

		log.Printf(`
Add the following task to your tasks.json to provide easy access to
cifuzz coverage runs from within VS Code. You can use the Coverage
Gutters extension to visualize the generated coverage report.
%s

You can download the Coverage Gutters extension from:

	https://marketplace.visualstudio.com/items?itemName=ryanluker.vscode-coverage-gutters
`, presetsSrc)
	}

	return nil
}

func setupCMakePresets(projectDir string, finder runfiles.RunfilesFinder) error {
	presetsSrcPath, err := finder.CMakePresetsPath()
	if err != nil {
		return err
	}
	presetsDestPath := filepath.Join(projectDir, "CMakeUserPresets.json")
	hasPresets, err := fileutil.Exists(presetsDestPath)
	if err != nil {
		return err
	}

	if !hasPresets {
		// Situation: The user doesn't have a CMake user preset set up and
		// may thus be unaware of this functionality. Create one and tell
		// them about it.
		err = copy2.Copy(presetsSrcPath, presetsDestPath)
		if err != nil {
			return errors.WithStack(err)
		}
		log.Printf(`
CMakeUserPresets.json has been created. Those presets simplify
the execution of regression tests from the command line and provide
integration with IDEs such as CLion and VS Code.
This file should not be checked in to version control systems.
To learn more about CMake presets, visit:

    https://github.com/microsoft/vscode-cmake-tools/blob/main/docs/cmake-presets.md
    https://www.jetbrains.com/help/clion/cmake-presets.html`)
	} else {
		// Situation: The user does have a CMake user preset set up, so we
		// assume them to know about the benefits. We suggest to the user
		// that they add our preset to the existing CMakeUserPresets.json.
		presetsSrc, err := os.ReadFile(presetsSrcPath)
		if err != nil {
			return errors.WithStack(err)
		}

		log.Printf(`
Add the following presets to your CMakeUserPresets.json. Those presets
simplify the execution of regression tests from the command line and
provide integration with IDEs such as CLion and VS Code:

%s`, presetsSrc)
	}

	return nil
}
