package notify

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
)

// The webhook path is itself the credential, so the secret sentinel lives in the
// path segments a Slack incoming webhook uses.
const slackWebhookSecretSentinel = "asc-red-sentinel-slack-webhook-3fd914"

func slackWebhookWithSecret() string {
	return "http://127.0.0.1:1/services/T00000000/B00000000/" + slackWebhookSecretSentinel
}

type failingSlackTransport struct {
	err error
}

func (t *failingSlackTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, t.err
}

func stubSlackTransport(t *testing.T, transportErr error) {
	t.Helper()
	original := slackHTTPClient
	t.Cleanup(func() {
		slackHTTPClient = original
	})
	slackHTTPClient = func() *http.Client {
		return &http.Client{Transport: &failingSlackTransport{err: transportErr}}
	}
}

func TestNotifySlackTransportFailuresNeverExposeWebhookSecret(t *testing.T) {
	tests := []struct {
		name         string
		transportErr error
		wantContext  string
	}{
		{name: "dns", transportErr: &net.DNSError{Err: "no such host", Name: "hooks.slack.com"}, wantContext: ""},
		{name: "tls", transportErr: errors.New("tls: failed to verify certificate"), wantContext: ""},
		{name: "connection refused", transportErr: errors.New("connect: connection refused"), wantContext: ""},
		{name: "proxy", transportErr: errors.New("proxyconnect tcp: bad proxy"), wantContext: ""},
		{name: "redirect", transportErr: errors.New("stopped after 10 redirects"), wantContext: ""},
		{name: "timeout", transportErr: context.DeadlineExceeded, wantContext: "timeout"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(slackWebhookEnvVar, "")
			t.Setenv(slackWebhookAllowLocalEnv, "1")
			t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))
			stubSlackTransport(t, test.transportErr)

			root := SlackCommand()
			root.FlagSet.SetOutput(io.Discard)

			var runErr error
			stderr := captureOutput(t, func() {
				if err := root.Parse([]string{
					"--webhook", slackWebhookWithSecret(),
					"--message", "Build uploaded",
				}); err != nil {
					t.Fatalf("parse error: %v", err)
				}
				runErr = root.Run(context.Background())
			})

			if runErr == nil {
				t.Fatal("expected a transport error")
			}
			if strings.Contains(runErr.Error(), slackWebhookSecretSentinel) {
				t.Fatalf("error leaked the webhook secret: %q", runErr.Error())
			}
			if strings.Contains(stderr, slackWebhookSecretSentinel) {
				t.Fatalf("stderr leaked the webhook secret: %q", stderr)
			}
			if !strings.Contains(runErr.Error(), "notify slack: failed to send") {
				t.Fatalf("error dropped the operation context: %q", runErr.Error())
			}
			if !errors.Is(runErr, test.transportErr) {
				t.Fatalf("errors.Is lost the wrapped transport error: %v", runErr)
			}
			if test.wantContext != "" && !strings.Contains(runErr.Error(), test.wantContext) {
				t.Fatalf("error dropped %q context: %q", test.wantContext, runErr.Error())
			}
		})
	}
}

func TestNotifySlackTransportErrorKeepsDNSErrorInspectable(t *testing.T) {
	dnsErr := &net.DNSError{Err: "no such host", Name: "hooks.slack.com", IsNotFound: true}

	t.Setenv(slackWebhookEnvVar, "")
	t.Setenv(slackWebhookAllowLocalEnv, "1")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "nonexistent.json"))
	stubSlackTransport(t, dnsErr)

	root := SlackCommand()
	root.FlagSet.SetOutput(io.Discard)

	var runErr error
	captureOutput(t, func() {
		if err := root.Parse([]string{
			"--webhook", slackWebhookWithSecret(),
			"--message", "Build uploaded",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		runErr = root.Run(context.Background())
	})

	var target *net.DNSError
	if !errors.As(runErr, &target) {
		t.Fatalf("errors.As could not recover the DNS error from %v", runErr)
	}
	if !target.IsNotFound {
		t.Fatal("errors.As recovered a DNS error without its classification")
	}
}
