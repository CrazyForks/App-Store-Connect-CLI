package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/rootfs"
)

type pathSource string

const (
	pathSourceFlag        pathSource = "flag"
	pathSourceDeliverfile pathSource = "deliverfile"
	pathSourceDefault     pathSource = "default"
)

type importInputOptions struct {
	WorkDir         string
	FastlaneDir     string
	MetadataDir     string
	ScreenshotsDir  string
	SkipScreenshots bool
}

type importInputs struct {
	DeliverfilePath   string
	DeliverfileConfig DeliverfileConfig
	MetadataDir       string
	ScreenshotsDir    string
	MetadataSource    pathSource
	ScreenshotsSource pathSource
}

func resolveImportInputs(opts importInputOptions) (importInputs, []SkippedItem, error) {
	workDir := strings.TrimSpace(opts.WorkDir)
	if workDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return importInputs{}, nil, fmt.Errorf("resolve import paths: %w", err)
		}
		workDir = cwd
	}

	if opts.FastlaneDir != "" {
		if err := ensureDirExists(opts.FastlaneDir); err != nil {
			return importInputs{}, nil, err
		}
	}

	deliverfilePath, err := discoverDeliverfilePath(workDir, opts.FastlaneDir)
	if err != nil {
		return importInputs{}, nil, err
	}

	var config DeliverfileConfig
	if deliverfilePath != "" {
		config, err = parseDeliverfile(deliverfilePath)
		if err != nil {
			return importInputs{}, nil, err
		}
	}

	inputs := importInputs{
		DeliverfilePath:   deliverfilePath,
		DeliverfileConfig: config,
	}

	metadataDir, metadataSource, err := resolveImportPath(workDir, opts.FastlaneDir, deliverfilePath, opts.MetadataDir, config.MetadataPath, "metadata")
	if err != nil {
		return importInputs{}, nil, err
	}
	screenshotsDir, screenshotsSource, err := resolveImportPath(workDir, opts.FastlaneDir, deliverfilePath, opts.ScreenshotsDir, config.ScreenshotsPath, "screenshots")
	if err != nil {
		return importInputs{}, nil, err
	}
	skipScreenshots := opts.SkipScreenshots || config.SkipScreenshots

	skipped := []SkippedItem{}
	metadataDir, skipped, err = validateResolvedDir(metadataDir, metadataSource, "metadata", skipped)
	if err != nil {
		return importInputs{}, nil, err
	}
	if !skipScreenshots {
		screenshotsDir, skipped, err = validateResolvedDir(screenshotsDir, screenshotsSource, "screenshots", skipped)
		if err != nil {
			return importInputs{}, nil, err
		}
	}

	inputs.MetadataDir = metadataDir
	inputs.ScreenshotsDir = screenshotsDir
	inputs.MetadataSource = metadataSource
	inputs.ScreenshotsSource = screenshotsSource

	return inputs, skipped, nil
}

func resolveImportPath(workDir, fastlaneDir, deliverfilePath, explicitPath, deliverfilePathValue, defaultDir string) (string, pathSource, error) {
	if strings.TrimSpace(explicitPath) != "" {
		return explicitPath, pathSourceFlag, nil
	}
	if strings.TrimSpace(fastlaneDir) != "" {
		return filepath.Join(fastlaneDir, defaultDir), pathSourceFlag, nil
	}
	if strings.TrimSpace(deliverfilePathValue) != "" {
		base := workDir
		if deliverfilePath != "" {
			base = filepath.Dir(deliverfilePath)
		}
		resolved, err := containDeliverfilePath(workDir, base, deliverfilePathValue)
		if err != nil {
			return "", pathSourceDeliverfile, err
		}
		return resolved, pathSourceDeliverfile, nil
	}
	base := workDir
	if deliverfilePath != "" {
		base = filepath.Dir(deliverfilePath)
	}
	return filepath.Join(base, defaultDir), pathSourceDefault, nil
}

// containDeliverfilePath resolves a repository-controlled Deliverfile path
// value against the Deliverfile's own directory and requires the result to stay
// inside the working directory, because the Deliverfile ships with the checkout
// and must not select files outside it.
func containDeliverfilePath(workDir, base, value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if err := rootfs.ValidateRelativeAllowingTraversal(trimmed); err != nil {
		return "", fmt.Errorf("deliverfile path %q: %w", value, err)
	}
	root, err := rootfs.New(workDir)
	if err != nil {
		return "", err
	}
	// The Deliverfile directory may itself be relative to the process working
	// directory; make it absolute so joining below cannot re-resolve it against
	// the trusted root and duplicate the leading components.
	absoluteBase, err := filepath.Abs(base)
	if err != nil {
		return "", err
	}
	resolved, err := root.Resolve(filepath.Clean(filepath.Join(absoluteBase, trimmed)))
	if err != nil {
		return "", fmt.Errorf("deliverfile path %q must stay inside %s: %w", value, root.Path(), err)
	}
	return resolved, nil
}

func validateResolvedDir(path string, source pathSource, label string, skipped []SkippedItem) (string, []SkippedItem, error) {
	if strings.TrimSpace(path) == "" {
		return "", skipped, nil
	}
	if err := ensureDirExists(path); err != nil {
		if source == pathSourceDefault {
			skipped = append(skipped, SkippedItem{
				Path:   path,
				Reason: fmt.Sprintf("default %s directory not found", label),
			})
			return "", skipped, nil
		}
		return "", skipped, err
	}
	return path, skipped, nil
}

func discoverDeliverfilePath(workDir, fastlaneDir string) (string, error) {
	if strings.TrimSpace(fastlaneDir) != "" {
		path := filepath.Join(fastlaneDir, "Deliverfile")
		if exists, err := fileExists(path); err != nil {
			return "", err
		} else if exists {
			return path, nil
		}
		return "", nil
	}

	candidates := []string{
		filepath.Join(workDir, "Deliverfile"),
		filepath.Join(workDir, "fastlane", "Deliverfile"),
	}
	for _, path := range candidates {
		if exists, err := fileExists(path); err != nil {
			return "", err
		} else if exists {
			return path, nil
		}
	}
	return "", nil
}

func ensureDirExists(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("expected directory: %s", path)
	}
	return nil
}

func fileExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return info.Mode().IsRegular(), nil
}
