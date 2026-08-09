package signing

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/asc"
	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/cli/shared"
)

func captureOutput(t *testing.T, fn func()) (string, string) {
	t.Helper()

	oldStdout := os.Stdout
	oldStderr := os.Stderr

	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stderr pipe: %v", err)
	}

	os.Stdout = wOut
	os.Stderr = wErr

	outC := make(chan string)
	errC := make(chan string)

	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, rOut)
		_ = rOut.Close()
		outC <- buf.String()
	}()

	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, rErr)
		_ = rErr.Close()
		errC <- buf.String()
	}()

	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
		_ = wOut.Close()
		_ = wErr.Close()
	}()

	fn()

	_ = wOut.Close()
	_ = wErr.Close()

	stdout := <-outC
	stderr := <-errC

	os.Stdout = oldStdout
	os.Stderr = oldStderr

	return stdout, stderr
}

func TestSigningFetchValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "missing bundle-id",
			args:    []string{"signing", "fetch", "--profile-type", "IOS_APP_STORE"},
			wantErr: "Error: --bundle-id is required",
		},
		{
			name:    "missing profile-type",
			args:    []string{"signing", "fetch", "--bundle-id", "com.example.app"},
			wantErr: "Error: --profile-type is required",
		},
		{
			name:    "missing device for development profile",
			args:    []string{"signing", "fetch", "--bundle-id", "com.example.app", "--profile-type", "IOS_APP_DEVELOPMENT", "--create-missing"},
			wantErr: "Error: --device is required for development profiles",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cmd := SigningFetchCommand()
			cmd.FlagSet.SetOutput(io.Discard)

			stdout, stderr := captureOutput(t, func() {
				args := test.args
				if len(args) >= 2 && args[0] == "signing" && args[1] == "fetch" {
					args = args[2:]
				}
				if err := cmd.Parse(args); err != nil {
					t.Fatalf("parse error: %v", err)
				}
				err := cmd.Run(context.Background())
				if !errors.Is(err, flag.ErrHelp) {
					t.Fatalf("expected ErrHelp, got %v", err)
				}
			})

			if stdout != "" {
				t.Fatalf("expected empty stdout, got %q", stdout)
			}
			if !strings.Contains(stderr, test.wantErr) {
				t.Fatalf("expected error %q, got %q", test.wantErr, stderr)
			}
		})
	}
}

func TestSigningFetchWriteFiles_NoOverwrite(t *testing.T) {
	dir := t.TempDir()
	profilePath := filepath.Join(dir, "profile.mobileprovision")
	certPath := filepath.Join(dir, "cert.cer")

	profileContent := base64.StdEncoding.EncodeToString([]byte("profile"))
	certContent := base64.StdEncoding.EncodeToString([]byte("certificate"))

	profileData, err := decodeBase64Content("profile", profileContent)
	if err != nil {
		t.Fatalf("decode profile error: %v", err)
	}
	if err := shared.WriteProfileFile(profilePath, profileData); err != nil {
		t.Fatalf("writeProfileFile error: %v", err)
	}
	certData, err := decodeBase64Content("certificate", certContent)
	if err != nil {
		t.Fatalf("decode certificate error: %v", err)
	}
	if err := writeBinaryFile(certPath, certData); err != nil {
		t.Fatalf("writeBinaryFile error: %v", err)
	}

	if data, err := os.ReadFile(profilePath); err != nil {
		t.Fatalf("read profile error: %v", err)
	} else if string(data) != "profile" {
		t.Fatalf("unexpected profile content: %q", string(data))
	}

	if data, err := os.ReadFile(certPath); err != nil {
		t.Fatalf("read certificate error: %v", err)
	} else if string(data) != "certificate" {
		t.Fatalf("unexpected certificate content: %q", string(data))
	}

	if err := shared.WriteProfileFile(profilePath, profileData); err == nil {
		t.Fatal("expected error when overwriting profile file")
	} else if !errors.Is(err, os.ErrExist) {
		t.Fatalf("expected ErrExist, got %v", err)
	}

	if err := writeBinaryFile(certPath, certData); err == nil {
		t.Fatal("expected error when overwriting certificate file")
	} else if !errors.Is(err, os.ErrExist) {
		t.Fatalf("expected ErrExist, got %v", err)
	}
}

func TestFindActiveProfilesUseBundleIDRelationship(t *testing.T) {
	widgetProfileContent := base64.StdEncoding.EncodeToString([]byte("application-identifier=TEAM.com.example.signing.profile.widget"))
	mainProfileContent := base64.StdEncoding.EncodeToString([]byte("application-identifier=TEAM.com.example.signing.profile"))
	requestPaths := []string{}
	client := newSigningFetchTestClient(t, func(req *http.Request) *http.Response {
		requestPaths = append(requestPaths, req.URL.Path)

		switch req.URL.Path {
		case "/v1/profiles":
			return signingFetchJSONResponse(
				http.StatusOK,
				fmt.Sprintf(
					`{"data":[{"type":"profiles","id":"profile-widget","attributes":{"name":"Widget-stamped main profile","profileType":"IOS_APP_STORE","profileState":"ACTIVE","profileContent":%q}}]}`,
					widgetProfileContent,
				),
			)
		case "/v1/bundleIds/bundle-main/profiles":
			return signingFetchJSONResponse(
				http.StatusOK,
				fmt.Sprintf(
					`{"data":[{"type":"profiles","id":"profile-main","attributes":{"name":"Main App Store","profileType":"IOS_APP_STORE","profileState":"ACTIVE","profileContent":%q}}]}`,
					mainProfileContent,
				),
			)
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
			return signingFetchJSONResponse(http.StatusInternalServerError, `{}`)
		}
	})

	profiles, err := findActiveProfiles(
		context.Background(),
		client,
		"bundle-main",
		"IOS_APP_STORE",
	)
	if err != nil {
		t.Fatalf("findActiveProfiles() error: %v", err)
	}
	if len(profiles) != 1 || profiles[0].ID != "profile-main" {
		t.Fatalf("expected exact bundle profile, got %#v", profiles)
	}

	if len(requestPaths) != 1 || requestPaths[0] != "/v1/bundleIds/bundle-main/profiles" {
		t.Fatalf("expected Bundle ID scoped profile lookup, got %v", requestPaths)
	}
}

func TestResolveSigningAssetsUsesOnlyExistingProfileCertificates(t *testing.T) {
	profileContent := base64.StdEncoding.EncodeToString([]byte("profile"))
	profileCertificateContent := base64.StdEncoding.EncodeToString([]byte("profile-certificate"))
	requestPaths := []string{}
	client := newSigningFetchTestClient(t, func(req *http.Request) *http.Response {
		requestPaths = append(requestPaths, req.URL.Path)

		switch req.URL.Path {
		case "/v1/bundleIds/bundle-main/profiles":
			return signingFetchJSONResponse(
				http.StatusOK,
				fmt.Sprintf(
					`{"data":[{"type":"profiles","id":"profile-main","attributes":{"name":"Main App Store","profileType":"IOS_APP_STORE","profileState":"ACTIVE","profileContent":%q}}]}`,
					profileContent,
				),
			)
		case "/v1/profiles/profile-main/certificates":
			if req.URL.Query().Get("cursor") == "next" {
				return signingFetchJSONResponse(
					http.StatusOK,
					fmt.Sprintf(
						`{"data":[{"type":"certificates","id":"cert-profile-2","attributes":{"certificateType":"IOS_DISTRIBUTION","serialNumber":"PROFILE2","certificateContent":%q}}]}`,
						profileCertificateContent,
					),
				)
			}
			return signingFetchJSONResponse(
				http.StatusOK,
				fmt.Sprintf(
					`{"data":[{"type":"certificates","id":"cert-profile","attributes":{"certificateType":"IOS_DISTRIBUTION","serialNumber":"PROFILE","certificateContent":%q}}],"links":{"next":"https://api.appstoreconnect.apple.com/v1/profiles/profile-main/certificates?cursor=next"}}`,
					profileCertificateContent,
				),
			)
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
			return signingFetchJSONResponse(http.StatusInternalServerError, `{}`)
		}
	})

	profile, certificates, created, err := resolveSigningAssets(
		context.Background(),
		client,
		signingAssetsOptions{
			BundleIDResourceID: "bundle-main",
			BundleIdentifier:   "com.example.signing.profile",
			ProfileType:        "IOS_APP_STORE",
		},
	)
	if err != nil {
		t.Fatalf("resolveSigningAssets() error: %v", err)
	}
	if created {
		t.Fatal("expected existing profile, got created profile")
	}
	if profile.Data.ID != "profile-main" {
		t.Fatalf("expected profile-main, got %s", profile.Data.ID)
	}
	if got := strings.Join(extractIDs(certificates.Data), ","); got != "cert-profile,cert-profile-2" {
		t.Fatalf("expected only the paginated profile certificates, got %s", got)
	}
	if strings.Join(requestPaths, ",") != "/v1/bundleIds/bundle-main/profiles,/v1/profiles/profile-main/certificates,/v1/profiles/profile-main/certificates" {
		t.Fatalf("expected profile-scoped certificate lookup, got %v", requestPaths)
	}
}

func TestResolveSigningAssetsFiltersExistingProfileCertificatesByRequestedType(t *testing.T) {
	tests := []struct {
		name            string
		certificateType string
		wantID          string
		wantErr         string
	}{
		{
			name:            "matching associated certificate",
			certificateType: "ios_distribution",
			wantID:          "cert-ios",
		},
		{
			name:            "comma separated types include matching certificate",
			certificateType: "DEVELOPER_ID_APPLICATION, ios_distribution",
			wantID:          "cert-ios",
		},
		{
			name:            "no associated certificate matches",
			certificateType: "DEVELOPER_ID_APPLICATION",
			wantErr:         "profile profile-main has no associated certificates of type DEVELOPER_ID_APPLICATION",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newSigningFetchTestClient(t, func(req *http.Request) *http.Response {
				switch req.URL.Path {
				case "/v1/bundleIds/bundle-main/profiles":
					return signingFetchJSONResponse(
						http.StatusOK,
						`{"data":[{"type":"profiles","id":"profile-main","attributes":{"profileType":"IOS_APP_STORE","profileState":"ACTIVE"}}]}`,
					)
				case "/v1/profiles/profile-main/certificates":
					return signingFetchJSONResponse(
						http.StatusOK,
						`{"data":[{"type":"certificates","id":"cert-mac","attributes":{"certificateType":"MAC_APP_DISTRIBUTION"}},{"type":"certificates","id":"cert-ios","attributes":{"certificateType":"IOS_DISTRIBUTION"}}]}`,
					)
				default:
					t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
					return signingFetchJSONResponse(http.StatusInternalServerError, `{}`)
				}
			})

			_, certificates, _, err := resolveSigningAssets(
				context.Background(),
				client,
				signingAssetsOptions{
					BundleIDResourceID: "bundle-main",
					BundleIdentifier:   "com.example.signing.profile",
					ProfileType:        "IOS_APP_STORE",
					CertificateType:    tt.certificateType,
				},
			)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveSigningAssets() error: %v", err)
			}
			if got := extractIDs(certificates.Data); len(got) != 1 || got[0] != tt.wantID {
				t.Fatalf("expected only %s, got %v", tt.wantID, got)
			}
		})
	}
}

func TestResolveSigningAssetsRejectsUnknownCertificateTypesBeforeLookup(t *testing.T) {
	requests := 0
	client := newSigningFetchTestClient(t, func(req *http.Request) *http.Response {
		requests++
		t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
		return signingFetchJSONResponse(http.StatusInternalServerError, `{}`)
	})

	_, _, _, err := resolveSigningAssets(
		context.Background(),
		client,
		signingAssetsOptions{
			BundleIDResourceID: "bundle-main",
			BundleIdentifier:   "com.example.signing.profile",
			ProfileType:        "IOS_APP_STORE",
			CertificateType:    "IOS_DISTRIBUTION,BOGUS",
		},
	)
	if err == nil || !strings.Contains(err.Error(), "unsupported certificate type BOGUS") {
		t.Fatalf("resolveSigningAssets() error = %v, want unsupported certificate type", err)
	}
	if requests != 0 {
		t.Fatalf("expected validation before lookup, got %d requests", requests)
	}
}

func TestResolveSigningCertificateTypesUsesIOSCertificatesForTVOSProfiles(t *testing.T) {
	tests := []struct {
		profileType string
		want        string
	}{
		{profileType: "TVOS_APP_DEVELOPMENT", want: "IOS_DEVELOPMENT"},
		{profileType: "TVOS_APP_STORE", want: "IOS_DISTRIBUTION"},
		{profileType: "TVOS_APP_ADHOC", want: "IOS_DISTRIBUTION"},
		{profileType: "TVOS_APP_INHOUSE", want: "IOS_DISTRIBUTION"},
	}

	for _, tt := range tests {
		t.Run(tt.profileType, func(t *testing.T) {
			got, err := resolveSigningCertificateTypes(tt.profileType, "")
			if err != nil {
				t.Fatalf("resolveSigningCertificateTypes() error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("resolveSigningCertificateTypes() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveSigningAssetsChecksEveryActiveProfileForInferredCertificateType(t *testing.T) {
	requestPaths := []string{}
	client := newSigningFetchTestClient(t, func(req *http.Request) *http.Response {
		requestPaths = append(requestPaths, req.URL.Path)
		switch req.URL.Path {
		case "/v1/bundleIds/bundle-main/profiles":
			return signingFetchJSONResponse(http.StatusOK, `{"data":[{"type":"profiles","id":"profile-first","attributes":{"profileType":"IOS_APP_STORE","profileState":"ACTIVE"}},{"type":"profiles","id":"profile-second","attributes":{"profileType":"IOS_APP_STORE","profileState":"ACTIVE"}}]}`)
		case "/v1/profiles/profile-first/certificates":
			return signingFetchJSONResponse(http.StatusOK, `{"data":[{"type":"certificates","id":"cert-mac","attributes":{"certificateType":"MAC_APP_DISTRIBUTION"}}]}`)
		case "/v1/profiles/profile-second/certificates":
			return signingFetchJSONResponse(http.StatusOK, `{"data":[{"type":"certificates","id":"cert-ios","attributes":{"certificateType":"IOS_DISTRIBUTION"}}]}`)
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
			return signingFetchJSONResponse(http.StatusInternalServerError, `{}`)
		}
	})

	profile, certificates, created, err := resolveSigningAssets(
		context.Background(),
		client,
		signingAssetsOptions{
			BundleIDResourceID: "bundle-main",
			BundleIdentifier:   "com.example.signing.profile",
			ProfileType:        "IOS_APP_STORE",
		},
	)
	if err != nil {
		t.Fatalf("resolveSigningAssets() error: %v", err)
	}
	if created || profile.Data.ID != "profile-second" {
		t.Fatalf("expected matching existing profile, got created=%v profile=%#v", created, profile)
	}
	if got := extractIDs(certificates.Data); len(got) != 1 || got[0] != "cert-ios" {
		t.Fatalf("expected cert-ios, got %v", got)
	}
	wantPaths := "/v1/bundleIds/bundle-main/profiles,/v1/profiles/profile-first/certificates,/v1/profiles/profile-second/certificates"
	if strings.Join(requestPaths, ",") != wantPaths {
		t.Fatalf("unexpected lookup order: %v", requestPaths)
	}
}

func TestResolveSigningAssetsCreatesWhenActiveProfilesLackRequestedCertificate(t *testing.T) {
	requestPaths := []string{}
	client := newSigningFetchTestClient(t, func(req *http.Request) *http.Response {
		requestPaths = append(requestPaths, req.Method+" "+req.URL.Path)
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/bundleIds/bundle-main/profiles":
			return signingFetchJSONResponse(http.StatusOK, `{"data":[{"type":"profiles","id":"profile-first","attributes":{"profileType":"IOS_APP_STORE","profileState":"ACTIVE"}},{"type":"profiles","id":"profile-second","attributes":{"profileType":"IOS_APP_STORE","profileState":"ACTIVE"}}]}`)
		case req.Method == http.MethodGet && strings.HasPrefix(req.URL.Path, "/v1/profiles/"):
			return signingFetchJSONResponse(http.StatusOK, `{"data":[{"type":"certificates","id":"cert-mac","attributes":{"certificateType":"MAC_APP_DISTRIBUTION"}}]}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/certificates":
			return signingFetchJSONResponse(http.StatusOK, `{"data":[{"type":"certificates","id":"cert-ios","attributes":{"certificateType":"IOS_DISTRIBUTION"}}]}`)
		case req.Method == http.MethodPost && req.URL.Path == "/v1/profiles":
			return signingFetchJSONResponse(http.StatusCreated, `{"data":{"type":"profiles","id":"profile-created","attributes":{"profileType":"IOS_APP_STORE","profileState":"ACTIVE"}}}`)
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
			return signingFetchJSONResponse(http.StatusInternalServerError, `{}`)
		}
	})

	profile, certificates, created, err := resolveSigningAssets(
		context.Background(),
		client,
		signingAssetsOptions{
			BundleIDResourceID: "bundle-main",
			BundleIdentifier:   "com.example.signing.profile",
			ProfileType:        "IOS_APP_STORE",
			CertificateType:    "IOS_DISTRIBUTION",
			CreateMissing:      true,
		},
	)
	if err != nil {
		t.Fatalf("resolveSigningAssets() error: %v", err)
	}
	if !created || profile.Data.ID != "profile-created" {
		t.Fatalf("expected created profile, got created=%v profile=%#v", created, profile)
	}
	if got := extractIDs(certificates.Data); len(got) != 1 || got[0] != "cert-ios" {
		t.Fatalf("expected creation certificate cert-ios, got %v", got)
	}
	wantPaths := "GET /v1/bundleIds/bundle-main/profiles,GET /v1/profiles/profile-first/certificates,GET /v1/profiles/profile-second/certificates,GET /v1/certificates,POST /v1/profiles"
	if strings.Join(requestPaths, ",") != wantPaths {
		t.Fatalf("unexpected lookup and creation order: %v", requestPaths)
	}
}

func TestResolveSigningAssetsPreflightsBeforeCreatingProfile(t *testing.T) {
	certificateContent := base64.StdEncoding.EncodeToString([]byte("certificate"))
	profileContent := base64.StdEncoding.EncodeToString([]byte("profile"))
	events := []string{}
	client := newSigningFetchTestClient(t, func(req *http.Request) *http.Response {
		events = append(events, req.Method+" "+req.URL.Path)

		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/bundleIds/bundle-main/profiles":
			return signingFetchJSONResponse(http.StatusOK, `{"data":[]}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/certificates":
			return signingFetchJSONResponse(
				http.StatusOK,
				fmt.Sprintf(
					`{"data":[{"type":"certificates","id":"cert-1","attributes":{"certificateType":"IOS_DISTRIBUTION","serialNumber":"CERT1","certificateContent":%q}}]}`,
					certificateContent,
				),
			)
		case req.Method == http.MethodPost && req.URL.Path == "/v1/profiles":
			return signingFetchJSONResponse(
				http.StatusCreated,
				fmt.Sprintf(
					`{"data":{"type":"profiles","id":"profile-created","attributes":{"name":"Created Profile","profileType":"IOS_APP_STORE","profileState":"ACTIVE","profileContent":%q}}}`,
					profileContent,
				),
			)
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
			return signingFetchJSONResponse(http.StatusInternalServerError, `{}`)
		}
	})

	profile, certificates, created, err := resolveSigningAssets(
		context.Background(),
		client,
		signingAssetsOptions{
			BundleIDResourceID: "bundle-main",
			BundleIdentifier:   "com.example.signing.profile",
			ProfileType:        "IOS_APP_STORE",
			CreateMissing:      true,
			BeforeCreate: func() error {
				events = append(events, "repository preflight")
				return nil
			},
		},
	)
	if err != nil {
		t.Fatalf("resolveSigningAssets() error: %v", err)
	}
	if !created || profile.Data.ID != "profile-created" {
		t.Fatalf("expected newly created profile, got created=%v id=%s", created, profile.Data.ID)
	}
	if got := extractIDs(certificates.Data); len(got) != 1 || got[0] != "cert-1" {
		t.Fatalf("expected creation certificate, got %v", got)
	}
	wantEvents := []string{
		"GET /v1/bundleIds/bundle-main/profiles",
		"GET /v1/certificates",
		"repository preflight",
		"POST /v1/profiles",
	}
	if strings.Join(events, ",") != strings.Join(wantEvents, ",") {
		t.Fatalf("unexpected operation order: got %v, want %v", events, wantEvents)
	}
}

func TestResolveSigningAssetsRefreshesCreateTimeoutAfterPreflight(t *testing.T) {
	requestCtx, expireRequest := context.WithCancel(context.Background())
	events := []string{}
	client := newSigningFetchTestClient(t, func(req *http.Request) *http.Response {
		events = append(events, req.Method+" "+req.URL.Path)
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/bundleIds/bundle-main/profiles":
			return signingFetchJSONResponse(http.StatusOK, `{"data":[]}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/certificates":
			return signingFetchJSONResponse(http.StatusOK, `{"data":[{"type":"certificates","id":"cert-1","attributes":{"certificateType":"IOS_DISTRIBUTION"}}]}`)
		case req.Method == http.MethodPost && req.URL.Path == "/v1/profiles":
			return signingFetchJSONResponse(http.StatusCreated, `{"data":{"type":"profiles","id":"profile-created","attributes":{"profileType":"IOS_APP_STORE","profileState":"ACTIVE"}}}`)
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
			return signingFetchJSONResponse(http.StatusInternalServerError, `{}`)
		}
	})

	profile, _, created, err := resolveSigningAssets(
		requestCtx,
		client,
		signingAssetsOptions{
			BundleIDResourceID: "bundle-main",
			BundleIdentifier:   "com.example.signing.profile",
			ProfileType:        "IOS_APP_STORE",
			CreateMissing:      true,
			BeforeCreate: func() error {
				events = append(events, "repository preflight")
				expireRequest()
				return nil
			},
			CreateContext: func() (context.Context, context.CancelFunc) {
				events = append(events, "refresh request timeout")
				return context.WithCancel(context.Background())
			},
		},
	)
	if err != nil {
		t.Fatalf("resolveSigningAssets() error: %v", err)
	}
	if !created || profile.Data.ID != "profile-created" {
		t.Fatalf("expected created profile, got created=%v profile=%#v", created, profile)
	}
	wantEvents := []string{
		"GET /v1/bundleIds/bundle-main/profiles",
		"GET /v1/certificates",
		"repository preflight",
		"refresh request timeout",
		"POST /v1/profiles",
	}
	if strings.Join(events, ",") != strings.Join(wantEvents, ",") {
		t.Fatalf("unexpected operation order: got %v, want %v", events, wantEvents)
	}
}

type signingFetchRoundTripFunc func(*http.Request) *http.Response

func (fn signingFetchRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req), nil
}

func newSigningFetchTestClient(t *testing.T, fn signingFetchRoundTripFunc) *asc.Client {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("MarshalPKCS8PrivateKey() error: %v", err)
	}
	keyPath := filepath.Join(t.TempDir(), "key.p8")
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), 0o600); err != nil {
		t.Fatalf("write key: %v", err)
	}

	client, err := asc.NewClientWithHTTPClient("KEY123", "ISS456", keyPath, &http.Client{
		Transport: fn,
	})
	if err != nil {
		t.Fatalf("NewClientWithHTTPClient() error: %v", err)
	}
	return client
}

func signingFetchJSONResponse(status int, body string) *http.Response {
	return &http.Response{
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}
