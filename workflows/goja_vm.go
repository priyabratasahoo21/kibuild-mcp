package workflows

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dop251/goja"
)

// VMConfig holds execution parameters for the sandbox JS VM.
type VMConfig struct {
	ProjectPath string
	Timeout     time.Duration
}

// RunScript executes the given JS script in the Goja VM sandbox.
func RunScript(script string, config VMConfig) (goja.Value, error) {
	vm := goja.New()

	// Hard timeout configuration
	timeLimit := config.Timeout
	if timeLimit <= 0 {
		timeLimit = 3 * time.Second
	}

	// Channel to signal execution completion
	done := make(chan struct{})
	defer close(done)

	go func() {
		select {
		case <-done:
			return
		case <-time.After(timeLimit):
			vm.Interrupt("execution timeout exceeded")
		}
	}()

	// Expose safe filesystem API namespace
	fsAPI := vm.NewObject()

	// fs.read(path)
	err := fsAPI.Set("read", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 {
			panic(vm.ToValue("read requires 1 path argument"))
		}
		targetPath := call.Argument(0).String()
		resolved, err := resolveWorkspacePath(config.ProjectPath, targetPath)
		if err != nil {
			panic(vm.ToValue(err.Error()))
		}
		data, err := ioutil.ReadFile(resolved)
		if err != nil {
			panic(vm.ToValue(err.Error()))
		}
		return vm.ToValue(string(data))
	})
	if err != nil {
		return nil, err
	}

	// fs.write(path, content)
	err = fsAPI.Set("write", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.ToValue("write requires 2 arguments (path, content)"))
		}
		targetPath := call.Argument(0).String()
		content := call.Argument(1).String()
		resolved, err := resolveWorkspacePath(config.ProjectPath, targetPath)
		if err != nil {
			panic(vm.ToValue(err.Error()))
		}

		err = os.MkdirAll(filepath.Dir(resolved), 0755)
		if err != nil {
			panic(vm.ToValue(err.Error()))
		}

		err = ioutil.WriteFile(resolved, []byte(content), 0644)
		if err != nil {
			panic(vm.ToValue(err.Error()))
		}
		return goja.Undefined()
	})
	if err != nil {
		return nil, err
	}

	// fs.list(path)
	err = fsAPI.Set("list", func(call goja.FunctionCall) goja.Value {
		targetPath := "."
		if len(call.Arguments) >= 1 {
			targetPath = call.Argument(0).String()
		}
		resolved, err := resolveWorkspacePath(config.ProjectPath, targetPath)
		if err != nil {
			panic(vm.ToValue(err.Error()))
		}

		entries, err := ioutil.ReadDir(resolved)
		if err != nil {
			panic(vm.ToValue(err.Error()))
		}

		listResult := make([]map[string]interface{}, 0, len(entries))
		for _, entry := range entries {
			listResult = append(listResult, map[string]interface{}{
				"name":  entry.Name(),
				"isDir": entry.IsDir(),
				"size":  entry.Size(),
			})
		}
		return vm.ToValue(listResult)
	})
	if err != nil {
		return nil, err
	}

	// fs.grep(pattern, path, caseInsensitive)
	err = fsAPI.Set("grep", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 2 {
			panic(vm.ToValue("grep requires at least 2 arguments: (pattern, path, [caseInsensitive])"))
		}
		pattern := call.Argument(0).String()
		targetPath := call.Argument(1).String()
		caseInsensitive := false
		if len(call.Arguments) >= 3 {
			caseInsensitive = call.Argument(2).ToBoolean()
		}

		resolved, err := resolveWorkspacePath(config.ProjectPath, targetPath)
		if err != nil {
			panic(vm.ToValue(err.Error()))
		}

		var matches []map[string]interface{}

		err = filepath.Walk(resolved, func(walkPath string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if info.IsDir() {
				return nil
			}

			data, err := ioutil.ReadFile(walkPath)
			if err != nil {
				return nil
			}

			lines := strings.Split(string(data), "\n")
			for lineNum, lineContent := range lines {
				match := false
				if caseInsensitive {
					match = strings.Contains(strings.ToLower(lineContent), strings.ToLower(pattern))
				} else {
					match = strings.Contains(lineContent, pattern)
				}

				if match {
					relPath, _ := filepath.Rel(config.ProjectPath, walkPath)
					matches = append(matches, map[string]interface{}{
						"file":        relPath,
						"lineNumber":  lineNum + 1,
						"lineContent": strings.TrimSpace(lineContent),
					})
				}
			}
			return nil
		})

		if err != nil {
			panic(vm.ToValue(err.Error()))
		}

		return vm.ToValue(matches)
	})
	if err != nil {
		return nil, err
	}

	// Expose standard namespace to the engine
	err = vm.Set("fs", fsAPI)
	if err != nil {
		return nil, err
	}

	return vm.RunString(script)
}

func resolveWorkspacePath(projectPath, targetPath string) (string, error) {
	projectClean := filepath.Clean(projectPath)
	resolvedProj, err := filepath.EvalSymlinks(projectClean)
	if err != nil {
		resolvedProj = projectClean
	}

	var absolute string
	if filepath.IsAbs(targetPath) {
		absolute = filepath.Clean(targetPath)
	} else {
		absolute = filepath.Clean(filepath.Join(resolvedProj, targetPath))
	}

	resolvedAbsolute, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		resolvedAbsolute = absolute
	}

	rel, err := filepath.Rel(resolvedProj, resolvedAbsolute)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", errors.New("access denied: path out of workspace scope")
	}
	return resolvedAbsolute, nil
}
