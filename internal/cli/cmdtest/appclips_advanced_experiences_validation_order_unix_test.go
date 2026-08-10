//go:build darwin || linux

package cmdtest

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

func TestAppClipsAdvancedExperiencesCreateMissingSelectorDoesNotReadConfig(t *testing.T) {
	originalAppID, hadAppID := os.LookupEnv("ASC_APP_ID")
	if err := os.Unsetenv("ASC_APP_ID"); err != nil {
		t.Fatalf("unset ASC_APP_ID: %v", err)
	}
	t.Cleanup(func() {
		if hadAppID {
			if err := os.Setenv("ASC_APP_ID", originalAppID); err != nil {
				t.Errorf("restore ASC_APP_ID: %v", err)
			}
			return
		}
		if err := os.Unsetenv("ASC_APP_ID"); err != nil {
			t.Errorf("clear ASC_APP_ID: %v", err)
		}
	})

	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := unix.Mkfifo(configPath, 0o600); err != nil {
		t.Fatalf("create config sentinel: %v", err)
	}
	t.Setenv("ASC_CONFIG_PATH", configPath)

	configOpened := make(chan struct{})
	writerResult := make(chan error, 1)
	go func() {
		file, err := os.OpenFile(configPath, os.O_WRONLY, 0o600)
		if err != nil {
			writerResult <- err
			return
		}
		close(configOpened)
		if _, err := file.WriteString("{}"); err != nil {
			_ = file.Close()
			writerResult <- err
			return
		}
		writerResult <- file.Close()
	}()
	t.Cleanup(func() {
		select {
		case <-configOpened:
		case err := <-writerResult:
			if err != nil {
				t.Errorf("config sentinel writer: %v", err)
			}
			return
		default:
			fd, err := unix.Open(configPath, unix.O_RDONLY|unix.O_NONBLOCK, 0)
			if err != nil {
				t.Errorf("open config sentinel reader: %v", err)
				return
			}
			defer unix.Close(fd)
		}
		if err := <-writerResult; err != nil {
			t.Errorf("config sentinel writer: %v", err)
		}
	})

	assertAppClipAdvancedExperienceCreateUsageBeforeClient(
		t,
		nil,
		"Error: --app-clip-id or --bundle-id is required\n",
	)
	select {
	case <-configOpened:
		t.Fatal("configuration was read before selector validation")
	default:
	}
}
