package asc

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestOutputRegistryNotEmpty(t *testing.T) {
	if len(outputRegistry) == 0 {
		t.Fatal("output registry is empty; init() may not have run")
	}
}

func TestOutputRegistryAllHandlersNonNil(t *testing.T) {
	for typ, fn := range outputRegistry {
		if fn == nil {
			t.Errorf("nil handler registered for type %s", typ)
		}
	}
}

func TestOutputRegistryExpectedTypeCount(t *testing.T) {
	// Total registered types across both registries should be ~471.
	total := len(outputRegistry) + len(directRenderRegistry)
	const minExpected = 460
	if total < minExpected {
		t.Errorf("expected at least %d registered types, got %d (rows: %d, direct: %d)",
			minExpected, total, len(outputRegistry), len(directRenderRegistry))
	}
}

func TestDirectRenderRegistryAllHandlersNonNil(t *testing.T) {
	for typ, fn := range directRenderRegistry {
		if fn == nil {
			t.Errorf("nil handler registered for type %s", typ)
		}
	}
}

func TestRenderByRegistryFallbackToJSON(t *testing.T) {
	// Unregistered type should fall back to JSON without error.
	type unregistered struct {
		Value string `json:"value"`
	}
	output := captureStdout(t, func() error {
		return renderByRegistry(&unregistered{Value: "test"}, RenderTable)
	})
	if output == "" {
		t.Fatal("expected JSON fallback output")
	}
	if !strings.Contains(output, "test") {
		t.Fatalf("expected JSON output to contain 'test', got: %s", output)
	}
}

func TestRenderByRegistryNilFallsBackToJSON(t *testing.T) {
	output := captureStdout(t, func() error {
		return renderByRegistry(nil, RenderTable)
	})
	if strings.TrimSpace(output) != "null" {
		t.Fatalf("expected JSON null fallback output, got: %q", output)
	}
}

func TestRenderByRegistryUsesRowsRegistryRenderer(t *testing.T) {
	type registered struct {
		Value string
	}

	key := reflect.TypeOf(&registered{})
	cleanupRegistryTypes(t, key)

	registerRows(func(v *registered) ([]string, [][]string) {
		return []string{"value"}, [][]string{{v.Value}}
	})

	var gotHeaders []string
	var gotRows [][]string
	err := renderByRegistry(&registered{Value: "from-registry"}, func(headers []string, rows [][]string) {
		gotHeaders = headers
		gotRows = rows
	})
	if err != nil {
		t.Fatalf("renderByRegistry returned error: %v", err)
	}
	assertSingleRowEquals(t, gotHeaders, gotRows, []string{"value"}, []string{"from-registry"})
}

func TestRenderByRegistryPropagatesRowsRegistryErrors(t *testing.T) {
	type registered struct{}

	key := reflect.TypeOf(&registered{})
	cleanupRegistryTypes(t, key)

	wantErr := errors.New("rows renderer failed")
	registerRowsErr(func(*registered) ([]string, [][]string, error) {
		return nil, nil, wantErr
	})

	renderCalls := 0
	err := renderByRegistry(&registered{}, func([]string, [][]string) {
		renderCalls++
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("renderByRegistry error = %v, want %v", err, wantErr)
	}
	if renderCalls != 0 {
		t.Fatalf("expected render callback not to run on rows error, got %d calls", renderCalls)
	}
}

func TestRenderByRegistryUsesDirectRenderer(t *testing.T) {
	type registered struct {
		Value string
	}

	key := reflect.TypeOf(&registered{})
	cleanupRegistryTypes(t, key)

	registerDirect(func(v *registered, render func([]string, [][]string)) error {
		render([]string{"value"}, [][]string{{v.Value}})
		return nil
	})

	var gotHeaders []string
	var gotRows [][]string
	err := renderByRegistry(&registered{Value: "from-direct"}, func(headers []string, rows [][]string) {
		gotHeaders = headers
		gotRows = rows
	})
	if err != nil {
		t.Fatalf("renderByRegistry returned error: %v", err)
	}
	assertSingleRowEquals(t, gotHeaders, gotRows, []string{"value"}, []string{"from-direct"})
}

func TestRenderByRegistryPropagatesDirectRendererErrors(t *testing.T) {
	type registered struct{}

	key := reflect.TypeOf(&registered{})
	cleanupRegistryTypes(t, key)

	wantErr := errors.New("direct renderer failed")
	registerDirect(func(*registered, func([]string, [][]string)) error {
		return wantErr
	})

	renderCalls := 0
	err := renderByRegistry(&registered{}, func([]string, [][]string) {
		renderCalls++
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("renderByRegistry error = %v, want %v", err, wantErr)
	}
	if renderCalls != 0 {
		t.Fatalf("expected render callback not to run on direct error, got %d calls", renderCalls)
	}
}

func TestRenderByRegistryPrefersDirectRegistryWhenBothHandlersExist(t *testing.T) {
	type registered struct{}

	key := reflect.TypeOf(&registered{})
	cleanupRegistryTypes(t, key)

	rowsHandlerCalled := seedRowsAndDirectHandlersForTest(
		key,
		renderWithRows([]string{"source"}, [][]string{{"direct"}}),
	)

	var gotHeaders []string
	var gotRows [][]string
	err := renderByRegistry(&registered{}, func(headers []string, rows [][]string) {
		gotHeaders = headers
		gotRows = rows
	})
	if err != nil {
		t.Fatalf("renderByRegistry returned error: %v", err)
	}
	if rowsHandlerCalled() {
		t.Fatal("expected direct handler precedence over rows handler")
	}
	assertSingleRowEquals(t, gotHeaders, gotRows, []string{"source"}, []string{"direct"})
}

func TestRenderByRegistryPrefersDirectRegistryErrorWhenBothHandlersExist(t *testing.T) {
	type registered struct{}

	key := reflect.TypeOf(&registered{})
	cleanupRegistryTypes(t, key)

	wantErr := errors.New("direct failed")
	rowsHandlerCalled := seedRowsAndDirectHandlersForTest(key, func(any, func([]string, [][]string)) error {
		return wantErr
	})

	renderCalls := 0
	err := renderByRegistry(&registered{}, func([]string, [][]string) {
		renderCalls++
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("renderByRegistry error = %v, want %v", err, wantErr)
	}
	if rowsHandlerCalled() {
		t.Fatal("expected rows handler not to run when direct handler exists")
	}
	if renderCalls != 0 {
		t.Fatalf("expected render callback not to run on direct error, got %d calls", renderCalls)
	}
}

func TestOutputRegistrySingleLinkageHelperRegistration(t *testing.T) {
	handler := requireOutputHandler(
		t,
		reflect.TypeOf(&AppStoreVersionSubmissionLinkageResponse{}),
		"AppStoreVersionSubmissionLinkageResponse",
	)

	headers, rows, err := handler(&AppStoreVersionSubmissionLinkageResponse{
		Data: ResourceData{
			Type: ResourceType("appStoreVersionSubmissions"),
			ID:   "submission-123",
		},
	})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	assertRowContains(t, headers, rows, 2, "submission-123")
}

func TestOutputRegistrySingleLinkageHelperPanicsOnNilExtractor(t *testing.T) {
	type linkage struct{}

	key := reflect.TypeOf(&linkage{})
	cleanupRegistryTypes(t, key)

	expectNilRegistryPanic(t, "linkage extractor", func() {
		registerSingleLinkageRows[linkage](nil)
	})

	assertRegistryTypeAbsent(t, key)
}

func TestOutputRegistrySingleLinkageHelperNilExtractorPanicsBeforeConflictChecks(t *testing.T) {
	type linkage struct{}

	preregisterRowsForConflict[linkage](t, "id")

	expectNilRegistryPanic(t, "linkage extractor", func() {
		registerSingleLinkageRows[linkage](nil)
	})
}

func TestOutputRegistryIDStateHelperRegistration(t *testing.T) {
	handler := requireOutputHandler(
		t,
		reflect.TypeOf(&BackgroundAssetVersionAppStoreReleaseResponse{}),
		"BackgroundAssetVersionAppStoreReleaseResponse",
	)

	headers, rows, err := handler(&BackgroundAssetVersionAppStoreReleaseResponse{
		Data: Resource[BackgroundAssetVersionAppStoreReleaseAttributes]{
			ID:         "release-1",
			Attributes: BackgroundAssetVersionAppStoreReleaseAttributes{State: "READY_FOR_SALE"},
		},
	})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	assertRowContains(t, headers, rows, 2, "release-1", "READY_FOR_SALE")
}

func TestOutputRegistryIDStateHelperPanicsOnNilExtractor(t *testing.T) {
	type state struct{}

	key := reflect.TypeOf(&state{})
	cleanupRegistryTypes(t, key)

	expectNilRegistryPanic(t, "id/state extractor", func() {
		registerIDStateRows[state](nil, func(id, value string) ([]string, [][]string) {
			return []string{"id", "state"}, [][]string{{id, value}}
		})
	})

	assertRegistryTypeAbsent(t, key)
}

func TestOutputRegistryIDStateHelperNilExtractorPanicsBeforeConflictChecks(t *testing.T) {
	type state struct{}

	preregisterRowsForConflict[state](t, "id", "state")

	expectNilRegistryPanic(t, "id/state extractor", func() {
		registerIDStateRows[state](nil, func(id, value string) ([]string, [][]string) {
			return []string{"id", "state"}, [][]string{{id, value}}
		})
	})
}

func TestOutputRegistryIDStateHelperPanicsOnNilRows(t *testing.T) {
	type state struct{}

	key := reflect.TypeOf(&state{})
	cleanupRegistryTypes(t, key)

	expectNilRegistryPanic(t, "id/state rows function", func() {
		registerIDStateRows[state](func(*state) (string, string) {
			return "id", "value"
		}, nil)
	})

	assertRegistryTypeAbsent(t, key)
}

func TestOutputRegistryIDStateHelperNilRowsPanicsBeforeConflictChecks(t *testing.T) {
	type state struct{}

	preregisterRowsForConflict[state](t, "id", "state")

	expectNilRegistryPanic(t, "id/state rows function", func() {
		registerIDStateRows[state](func(*state) (string, string) {
			return "id", "value"
		}, nil)
	})
}

func TestOutputRegistryIDBoolHelperRegistration(t *testing.T) {
	handler := requireOutputHandler(
		t,
		reflect.TypeOf(&AlternativeDistributionDomainDeleteResult{}),
		"AlternativeDistributionDomainDeleteResult",
	)

	headers, rows, err := handler(&AlternativeDistributionDomainDeleteResult{
		ID:      "domain-1",
		Deleted: true,
	})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	assertRowContains(t, headers, rows, 2, "domain-1", "true")
}

func TestOutputRegistryIDBoolHelperPanicsOnNilRows(t *testing.T) {
	type idBool struct{}

	key := reflect.TypeOf(&idBool{})
	cleanupRegistryTypes(t, key)

	expectNilRegistryPanic(t, "id/bool rows function", func() {
		registerIDBoolRows[idBool](func(*idBool) (string, bool) {
			return "id", true
		}, nil)
	})

	assertRegistryTypeAbsent(t, key)
}

func TestOutputRegistryIDBoolHelperPanicsOnNilExtractor(t *testing.T) {
	type idBool struct{}

	key := reflect.TypeOf(&idBool{})
	cleanupRegistryTypes(t, key)

	expectNilRegistryPanic(t, "id/bool extractor", func() {
		registerIDBoolRows[idBool](nil, func(id string, deleted bool) ([]string, [][]string) {
			return []string{"id", "deleted"}, [][]string{{id, fmt.Sprintf("%t", deleted)}}
		})
	})

	assertRegistryTypeAbsent(t, key)
}

func TestOutputRegistryIDBoolHelperNilExtractorPanicsBeforeConflictChecks(t *testing.T) {
	type idBool struct{}

	preregisterRowsForConflict[idBool](t, "id", "deleted")

	expectNilRegistryPanic(t, "id/bool extractor", func() {
		registerIDBoolRows[idBool](nil, func(id string, deleted bool) ([]string, [][]string) {
			return []string{"id", "deleted"}, [][]string{{id, fmt.Sprintf("%t", deleted)}}
		})
	})
}

func TestOutputRegistryResponseDataHelperRegistration(t *testing.T) {
	handler := requireOutputHandler(
		t,
		reflect.TypeOf(&Response[BetaGroupMetricAttributes]{}),
		"Response[BetaGroupMetricAttributes]",
	)

	headers, rows, err := handler(&Response[BetaGroupMetricAttributes]{
		Data: []Resource[BetaGroupMetricAttributes]{
			{
				ID:         "metric-1",
				Attributes: BetaGroupMetricAttributes{"installs": 12},
			},
		},
	})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	assertRowContains(t, headers, rows, 2, "metric-1", "installs=12")
}

func TestOutputRegistryResponseDataHelperPanicsOnNilRows(t *testing.T) {
	type attrs struct{}

	key := reflect.TypeOf(&Response[attrs]{})
	cleanupRegistryTypes(t, key)

	expectNilRegistryPanic(t, "response-data rows function", func() {
		registerResponseDataRows[attrs](nil)
	})

	assertRegistryTypeAbsent(t, key)
}

func TestOutputRegistryResponseDataHelperNilRowsPanicsBeforeConflictChecks(t *testing.T) {
	type attrs struct{}

	preregisterRowsForConflict[Response[attrs]](t, "id")

	expectNilRegistryPanic(t, "response-data rows function", func() {
		registerResponseDataRows[attrs](nil)
	})
}

func TestOutputRegistrySingleResourceHelperRegistration(t *testing.T) {
	type helperAttrs struct {
		Name string `json:"name"`
	}

	registerSingleResourceRowsAdapter(func(v *Response[helperAttrs]) ([]string, [][]string) {
		if len(v.Data) == 0 {
			return []string{"ID", "Name"}, nil
		}
		return []string{"ID", "Name"}, [][]string{{v.Data[0].ID, v.Data[0].Attributes.Name}}
	})

	key := reflect.TypeOf(&SingleResponse[helperAttrs]{})
	cleanupRegistryTypes(t, key)

	handler := requireOutputHandler(t, key, "SingleResponse helper")

	headers, rows, err := handler(&SingleResponse[helperAttrs]{
		Data: Resource[helperAttrs]{
			ID:         "helper-id",
			Attributes: helperAttrs{Name: "helper-name"},
		},
	})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	assertSingleRowEquals(t, headers, rows, []string{"ID", "Name"}, []string{"helper-id", "helper-name"})
}

func TestOutputRegistrySingleResourceHelperPanicsOnNilRowsFunction(t *testing.T) {
	type helperAttrs struct {
		Name string `json:"name"`
	}

	singleKey := reflect.TypeOf(&SingleResponse[helperAttrs]{})
	cleanupRegistryTypes(t, singleKey)

	expectNilRowsFunctionPanic(t, func() {
		registerSingleResourceRowsAdapter[helperAttrs](nil)
	})

	assertRegistryTypeAbsent(t, singleKey)
}

func TestOutputRegistrySingleResourceHelperNilRowsPanicsBeforeConflictChecks(t *testing.T) {
	type helperAttrs struct {
		Name string `json:"name"`
	}

	preregisterRowsForConflict[SingleResponse[helperAttrs]](t, "ID")

	expectNilRowsFunctionPanic(t, func() {
		registerSingleResourceRowsAdapter[helperAttrs](nil)
	})
}

func TestOutputRegistryRowsWithSingleResourceHelperRegistration(t *testing.T) {
	type attrs struct {
		Name string `json:"name"`
	}

	registerRowsWithSingleResourceAdapter(func(v *Response[attrs]) ([]string, [][]string) {
		if len(v.Data) == 0 {
			return []string{"ID", "Name"}, nil
		}
		return []string{"ID", "Name"}, [][]string{{v.Data[0].ID, v.Data[0].Attributes.Name}}
	})

	listKey := reflect.TypeOf(&Response[attrs]{})
	singleKey := reflect.TypeOf(&SingleResponse[attrs]{})
	cleanupRegistryTypes(t, listKey, singleKey)

	listHandler := requireOutputHandler(t, listKey, "list handler from rows+single-resource helper")
	singleHandler := requireOutputHandler(t, singleKey, "single handler from rows+single-resource helper")

	listHeaders, listRows, err := listHandler(&Response[attrs]{
		Data: []Resource[attrs]{{ID: "list-id", Attributes: attrs{Name: "list-name"}}},
	})
	if err != nil {
		t.Fatalf("list handler returned error: %v", err)
	}
	assertSingleRowEquals(t, listHeaders, listRows, []string{"ID", "Name"}, []string{"list-id", "list-name"})

	singleHeaders, singleRows, err := singleHandler(&SingleResponse[attrs]{
		Data: Resource[attrs]{ID: "single-id", Attributes: attrs{Name: "single-name"}},
	})
	if err != nil {
		t.Fatalf("single handler returned error: %v", err)
	}
	assertSingleRowEquals(t, singleHeaders, singleRows, []string{"ID", "Name"}, []string{"single-id", "single-name"})
}

func TestOutputRegistryRowsWithSingleResourceHelperNoPartialRegistrationOnPanic(t *testing.T) {
	type attrs struct {
		Name string `json:"name"`
	}

	listKey := reflect.TypeOf(&Response[attrs]{})
	preregisterRowsForConflict[SingleResponse[attrs]](t, "ID")
	cleanupRegistryTypes(t, listKey)

	expectPanic(t, "expected conflict panic when single handler is already registered", func() {
		registerRowsWithSingleResourceAdapter(func(v *Response[attrs]) ([]string, [][]string) {
			return []string{"ID"}, nil
		})
	})

	assertRegistryTypeAbsent(t, listKey)
}

func TestOutputRegistryRowsWithSingleResourceHelperNoPartialRegistrationWhenListRegistered(t *testing.T) {
	type attrs struct {
		Name string `json:"name"`
	}

	preregisterRowsForConflict[Response[attrs]](t, "ID")
	singleKey := reflect.TypeOf(&SingleResponse[attrs]{})
	cleanupRegistryTypes(t, singleKey)

	expectPanic(t, "expected conflict panic when list handler is already registered", func() {
		registerRowsWithSingleResourceAdapter(func(v *Response[attrs]) ([]string, [][]string) {
			return []string{"ID"}, nil
		})
	})

	assertRegistryTypeAbsent(t, singleKey)
}

func TestOutputRegistryRowsWithSingleResourceHelperNoPartialRegistrationWhenSingleDirectRegistered(t *testing.T) {
	type attrs struct {
		Name string `json:"name"`
	}

	listKey := reflect.TypeOf(&Response[attrs]{})
	singleKey := reflect.TypeOf(&SingleResponse[attrs]{})
	cleanupRegistryTypes(t, listKey, singleKey)

	preregisterDirectForConflict[SingleResponse[attrs]](t)

	expectPanic(t, "expected conflict panic when single direct handler is already registered", func() {
		registerRowsWithSingleResourceAdapter(func(v *Response[attrs]) ([]string, [][]string) {
			return []string{"ID"}, nil
		})
	})

	assertRegistryTypeAbsent(t, listKey)
}

func TestOutputRegistryRowsWithSingleResourceHelperNoPartialRegistrationWhenListDirectRegistered(t *testing.T) {
	type attrs struct {
		Name string `json:"name"`
	}

	listKey := reflect.TypeOf(&Response[attrs]{})
	singleKey := reflect.TypeOf(&SingleResponse[attrs]{})
	cleanupRegistryTypes(t, listKey, singleKey)

	preregisterDirectForConflict[Response[attrs]](t)

	expectPanic(t, "expected conflict panic when list direct handler is already registered", func() {
		registerRowsWithSingleResourceAdapter(func(v *Response[attrs]) ([]string, [][]string) {
			return []string{"ID"}, nil
		})
	})

	assertRegistryTypeAbsent(t, singleKey)
}

func TestOutputRegistryRowsWithSingleResourceHelperNoPartialRegistrationWhenRowsNil(t *testing.T) {
	type attrs struct {
		Name string `json:"name"`
	}

	listKey := reflect.TypeOf(&Response[attrs]{})
	singleKey := reflect.TypeOf(&SingleResponse[attrs]{})
	cleanupRegistryTypes(t, listKey, singleKey)

	expectNilRowsFunctionPanic(t, func() {
		registerRowsWithSingleResourceAdapter[attrs](nil)
	})

	assertRegistryTypesAbsent(t, listKey, singleKey)
}

func TestOutputRegistryRowsWithSingleResourceHelperNilRowsPanicsBeforeConflictChecks(t *testing.T) {
	type attrs struct {
		Name string `json:"name"`
	}

	preregisterRowsForConflict[Response[attrs]](t, "ID")
	singleKey := reflect.TypeOf(&SingleResponse[attrs]{})
	cleanupRegistryTypes(t, singleKey)

	expectNilRowsFunctionPanic(t, func() {
		registerRowsWithSingleResourceAdapter[attrs](nil)
	})

	assertRegistryTypeAbsent(t, singleKey)
}

func TestOutputRegistrySingleToListHelperRegistration(t *testing.T) {
	type single struct {
		Data string
	}
	type list struct {
		Data []string
	}

	registerSingleToListRowsAdapter[single, list](func(v *list) ([]string, [][]string) {
		if len(v.Data) == 0 {
			return []string{"value"}, nil
		}
		return []string{"value"}, [][]string{{v.Data[0]}}
	})

	key := reflect.TypeOf(&single{})
	cleanupRegistryTypes(t, key)

	handler := requireOutputHandler(t, key, "single-to-list helper")

	headers, rows, err := handler(&single{Data: "converted"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	assertSingleRowEquals(t, headers, rows, []string{"value"}, []string{"converted"})
}

func TestOutputRegistryRowsWithSingleToListHelperRegistration(t *testing.T) {
	type single struct {
		Data string
	}
	type list struct {
		Data []string
	}

	registerRowsWithSingleToListAdapter[single, list](func(v *list) ([]string, [][]string) {
		if len(v.Data) == 0 {
			return []string{"value"}, nil
		}
		return []string{"value"}, [][]string{{v.Data[0]}}
	})

	singleKey := reflect.TypeOf(&single{})
	listKey := reflect.TypeOf(&list{})
	cleanupRegistryTypes(t, singleKey, listKey)

	singleHandler := requireOutputHandler(t, singleKey, "single handler from rows+single-to-list helper")
	listHandler := requireOutputHandler(t, listKey, "list handler from rows+single-to-list helper")

	singleHeaders, singleRows, err := singleHandler(&single{Data: "single-value"})
	if err != nil {
		t.Fatalf("single handler returned error: %v", err)
	}
	assertSingleRowEquals(t, singleHeaders, singleRows, []string{"value"}, []string{"single-value"})

	listHeaders, listRows, err := listHandler(&list{Data: []string{"list-value"}})
	if err != nil {
		t.Fatalf("list handler returned error: %v", err)
	}
	assertSingleRowEquals(t, listHeaders, listRows, []string{"value"}, []string{"list-value"})
}

func TestOutputRegistryRowsWithSingleToListHelperNoPartialRegistrationOnPanic(t *testing.T) {
	type single struct {
		Data string
	}
	type list struct {
		Data []string
	}

	preregisterRowsForConflict[single](t, "value")
	listKey := reflect.TypeOf(&list{})
	cleanupRegistryTypes(t, listKey)

	expectPanic(t, "expected conflict panic when single handler is already registered", func() {
		registerRowsWithSingleToListAdapter[single, list](func(v *list) ([]string, [][]string) {
			return []string{"value"}, nil
		})
	})

	assertRegistryTypeAbsent(t, listKey)
}

func TestOutputRegistryRowsWithSingleToListHelperNoPartialRegistrationWhenListRegistered(t *testing.T) {
	type single struct {
		Data string
	}
	type list struct {
		Data []string
	}

	singleKey := reflect.TypeOf(&single{})
	preregisterRowsForConflict[list](t, "value")
	cleanupRegistryTypes(t, singleKey)

	expectPanic(t, "expected conflict panic when list handler is already registered", func() {
		registerRowsWithSingleToListAdapter[single, list](func(v *list) ([]string, [][]string) {
			return []string{"value"}, nil
		})
	})

	assertRegistryTypeAbsent(t, singleKey)
}

func TestOutputRegistryRowsWithSingleToListHelperNoPartialRegistrationWhenSingleDirectRegistered(t *testing.T) {
	type single struct {
		Data string
	}
	type list struct {
		Data []string
	}

	singleKey := reflect.TypeOf(&single{})
	listKey := reflect.TypeOf(&list{})
	cleanupRegistryTypes(t, singleKey, listKey)

	preregisterDirectForConflict[single](t)

	expectPanic(t, "expected conflict panic when single direct handler is already registered", func() {
		registerRowsWithSingleToListAdapter[single, list](func(v *list) ([]string, [][]string) {
			return []string{"value"}, nil
		})
	})

	assertRegistryTypeAbsent(t, listKey)
}

func TestOutputRegistryRowsWithSingleToListHelperNoPartialRegistrationWhenListDirectRegistered(t *testing.T) {
	type single struct {
		Data string
	}
	type list struct {
		Data []string
	}

	singleKey := reflect.TypeOf(&single{})
	listKey := reflect.TypeOf(&list{})
	cleanupRegistryTypes(t, singleKey, listKey)

	preregisterDirectForConflict[list](t)

	expectPanic(t, "expected conflict panic when list direct handler is already registered", func() {
		registerRowsWithSingleToListAdapter[single, list](func(v *list) ([]string, [][]string) {
			return []string{"value"}, nil
		})
	})

	assertRegistryTypeAbsent(t, singleKey)
}

func TestOutputRegistryRowsWithSingleToListHelperNoPartialRegistrationWhenAdapterPanics(t *testing.T) {
	type single struct {
		Value string
	}
	type list struct {
		Data []string
	}

	singleKey := reflect.TypeOf(&single{})
	listKey := reflect.TypeOf(&list{})
	cleanupRegistryTypes(t, singleKey, listKey)

	expectPanic(t, "expected adapter panic for missing Data field", func() {
		registerRowsWithSingleToListAdapter[single, list](func(v *list) ([]string, [][]string) {
			return []string{"value"}, nil
		})
	})

	assertRegistryTypesAbsent(t, singleKey, listKey)
}

func TestOutputRegistrySingleToListHelperCopiesLinks(t *testing.T) {
	type single struct {
		Data  ResourceData
		Links Links
	}
	type list struct {
		Data  []ResourceData
		Links Links
	}

	registerSingleToListRowsAdapter[single, list](func(v *list) ([]string, [][]string) {
		if len(v.Data) == 0 {
			return []string{"id", "self"}, nil
		}
		return []string{"id", "self"}, [][]string{{v.Data[0].ID, v.Links.Self}}
	})

	key := reflect.TypeOf(&single{})
	cleanupRegistryTypes(t, key)

	handler := requireOutputHandler(t, key, "single-to-list links helper")

	headers, rows, err := handler(&single{
		Data: ResourceData{ID: "item-1", Type: ResourceType("items")},
		Links: Links{
			Self: "https://example.test/items/1",
		},
	})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	assertSingleRowEquals(t, headers, rows, []string{"id", "self"}, []string{"item-1", "https://example.test/items/1"})
}

func TestOutputRegistrySingleToListHelperWorksWhenTargetHasNoLinks(t *testing.T) {
	type single struct {
		Data  string
		Links Links
	}
	type list struct {
		Data []string
	}

	registerSingleToListRowsAdapter[single, list](func(v *list) ([]string, [][]string) {
		if len(v.Data) == 0 {
			return []string{"value"}, nil
		}
		return []string{"value"}, [][]string{{v.Data[0]}}
	})

	key := reflect.TypeOf(&single{})
	cleanupRegistryTypes(t, key)

	handler := requireOutputHandler(t, key, "single-to-list no-target-links helper")

	headers, rows, err := handler(&single{
		Data:  "converted",
		Links: Links{Self: "https://example.test/unused"},
	})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	assertSingleRowEquals(t, headers, rows, []string{"value"}, []string{"converted"})
}

func TestOutputRegistrySingleToListHelperLeavesTargetLinksZeroWhenSourceHasNoLinks(t *testing.T) {
	type single struct {
		Data string
	}
	type list struct {
		Data  []string
		Links Links
	}

	registerSingleToListRowsAdapter[single, list](func(v *list) ([]string, [][]string) {
		if len(v.Data) == 0 {
			return []string{"value", "self"}, nil
		}
		return []string{"value", "self"}, [][]string{{v.Data[0], v.Links.Self}}
	})

	key := reflect.TypeOf(&single{})
	cleanupRegistryTypes(t, key)

	handler := requireOutputHandler(t, key, "single-to-list missing-source-links helper")

	headers, rows, err := handler(&single{Data: "converted"})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	assertSingleRowEquals(t, headers, rows, []string{"value", "self"}, []string{"converted", ""})
}

func TestOutputRegistrySingleToListHelperPanicsWithoutDataField(t *testing.T) {
	type single struct {
		Value string
	}
	type list struct {
		Data []string
	}

	expectPanic(t, "expected panic when source Data field is missing", func() {
		registerSingleToListRowsAdapter[single, list](func(v *list) ([]string, [][]string) {
			return []string{"value"}, [][]string{{v.Data[0]}}
		})
	})
}

func TestOutputRegistrySingleToListHelperPanicsWhenSourceIsNotStruct(t *testing.T) {
	type single string
	type list struct {
		Data []string
	}

	expectPanicContains(t, "source type must be a struct", func() {
		registerSingleToListRowsAdapter[single, list](func(v *list) ([]string, [][]string) {
			return nil, nil
		})
	})
}

func TestOutputRegistrySingleToListHelperPanicsWhenTargetIsNotStruct(t *testing.T) {
	type single struct {
		Data string
	}
	type list []string

	expectPanicContains(t, "target type must be a struct", func() {
		registerSingleToListRowsAdapter[single, list](func(v *list) ([]string, [][]string) {
			return nil, nil
		})
	})
}

func TestOutputRegistrySingleToListHelperPanicsWhenTargetDataIsNotSlice(t *testing.T) {
	type single struct {
		Data string
	}
	type list struct {
		Data string
	}

	expectPanicContains(t, "target Data field must be a slice", func() {
		registerSingleToListRowsAdapter[single, list](func(v *list) ([]string, [][]string) {
			return []string{"value"}, [][]string{{v.Data}}
		})
	})
}

func TestOutputRegistrySingleToListHelperPanicsOnDataTypeMismatch(t *testing.T) {
	type single struct {
		Data int
	}
	type list struct {
		Data []string
	}

	expectPanicContains(t, "Data type mismatch source=int target=string", func() {
		registerSingleToListRowsAdapter[single, list](func(v *list) ([]string, [][]string) {
			return []string{"value"}, nil
		})
	})
}

func TestOutputRegistrySingleToListHelperPanicsOnNilRowsFunction(t *testing.T) {
	type single struct {
		Data string
	}
	type list struct {
		Data []string
	}

	singleKey := reflect.TypeOf(&single{})
	cleanupRegistryTypes(t, singleKey)

	expectNilRowsFunctionPanic(t, func() {
		registerSingleToListRowsAdapter[single, list](nil)
	})

	assertRegistryTypeAbsent(t, singleKey)
}

func TestOutputRegistrySingleToListHelperAdapterValidationPanicsBeforeConflictChecks(t *testing.T) {
	type single struct {
		Value string
	}
	type list struct {
		Data []string
	}

	preregisterRowsForConflict[single](t, "value")

	expectPanicContains(t, "requires Data field", func() {
		registerSingleToListRowsAdapter[single, list](func(v *list) ([]string, [][]string) {
			return []string{"value"}, nil
		})
	})
}

func TestOutputRegistrySingleToListHelperNilRowsPanicsBeforeConflictChecks(t *testing.T) {
	type single struct {
		Data string
	}
	type list struct {
		Data []string
	}

	preregisterRowsForConflict[single](t, "value")

	expectNilRowsFunctionPanic(t, func() {
		registerSingleToListRowsAdapter[single, list](nil)
	})
}

func TestOutputRegistryRegisterRowsPanicsOnDuplicate(t *testing.T) {
	type duplicate struct{}
	preregisterRowsForConflict[duplicate](t, "value")

	expectDuplicateRegistrationPanic(t, func() {
		registerRowsForConflict[duplicate]("value")
	})
}

func TestOutputRegistryRegisterRowsPanicsOnNilFunction(t *testing.T) {
	type nilRows struct{}
	key := reflect.TypeOf(&nilRows{})
	cleanupRegistryTypes(t, key)

	expectNilRowsFunctionPanic(t, func() {
		registerRows[nilRows](nil)
	})

	assertRegistryTypeAbsent(t, key)
}

func TestOutputRegistryRegisterRowsNilFunctionPanicIncludesType(t *testing.T) {
	type nilRows struct{}
	key := reflect.TypeOf(&nilRows{})
	cleanupRegistryTypes(t, key)

	expectPanicContains(t, key.String(), func() {
		registerRows[nilRows](nil)
	})
}

func TestOutputRegistryRegisterRowsNilFunctionPanicsBeforeConflictChecks(t *testing.T) {
	type nilRows struct{}
	preregisterRowsForConflict[nilRows](t, "value")

	expectNilRowsFunctionPanic(t, func() {
		registerRows[nilRows](nil)
	})
}

func TestOutputRegistryRegisterRowsPanicsWhenDirectRegistered(t *testing.T) {
	type conflict struct{}
	preregisterDirectForConflict[conflict](t)

	expectDuplicateRegistrationPanic(t, func() {
		registerRowsForConflict[conflict]("value")
	})
}

func TestOutputRegistryRegisterRowsErrPanicsWhenDirectRegistered(t *testing.T) {
	type conflict struct{}
	preregisterDirectForConflict[conflict](t)

	expectDuplicateRegistrationPanic(t, func() {
		registerRowsErrForConflict[conflict]()
	})
}

func TestOutputRegistryRegisterRowsErrPanicsWhenRowsRegistered(t *testing.T) {
	type conflict struct{}
	preregisterRowsForConflict[conflict](t, "value")

	expectDuplicateRegistrationPanic(t, func() {
		registerRowsErrForConflict[conflict]()
	})
}

func TestOutputRegistryRegisterRowsErrPanicsOnDuplicate(t *testing.T) {
	type duplicate struct{}
	preregisterRowsErrForConflict[duplicate](t)

	expectDuplicateRegistrationPanic(t, func() {
		registerRowsErrForConflict[duplicate]()
	})
}

func TestOutputRegistryRegisterRowsErrPanicsOnNilFunction(t *testing.T) {
	type nilRowsErr struct{}
	key := reflect.TypeOf(&nilRowsErr{})
	cleanupRegistryTypes(t, key)

	expectNilRowsFunctionPanic(t, func() {
		registerRowsErr[nilRowsErr](nil)
	})

	assertRegistryTypeAbsent(t, key)
}

func TestOutputRegistryRegisterRowsErrNilFunctionPanicsBeforeConflictChecks(t *testing.T) {
	type nilRowsErr struct{}
	preregisterRowsErrForConflict[nilRowsErr](t)

	expectNilRowsFunctionPanic(t, func() {
		registerRowsErr[nilRowsErr](nil)
	})
}

func TestOutputRegistryRegisterDirectPanicsWhenRowsRegistered(t *testing.T) {
	type conflict struct{}
	preregisterRowsForConflict[conflict](t, "value")

	expectDuplicateRegistrationPanic(t, func() {
		registerDirectForConflict[conflict]()
	})
}

func TestOutputRegistryRegisterDirectPanicsWhenRowsErrRegistered(t *testing.T) {
	type conflict struct{}
	preregisterRowsErrForConflict[conflict](t)

	expectDuplicateRegistrationPanic(t, func() {
		registerDirectForConflict[conflict]()
	})
}

func TestOutputRegistryRegisterDirectPanicsOnDuplicate(t *testing.T) {
	type duplicate struct{}
	preregisterDirectForConflict[duplicate](t)

	expectDuplicateRegistrationPanic(t, func() {
		registerDirectForConflict[duplicate]()
	})
}

func TestOutputRegistryRegisterDirectPanicsOnNilFunction(t *testing.T) {
	type nilDirect struct{}
	key := reflect.TypeOf(&nilDirect{})
	cleanupRegistryTypes(t, key)

	expectNilDirectRenderFunctionPanic(t, func() {
		registerDirect[nilDirect](nil)
	})

	assertRegistryTypeAbsent(t, key)
}

func TestOutputRegistryRegisterDirectNilFunctionPanicsBeforeConflictChecks(t *testing.T) {
	type nilDirect struct{}
	preregisterDirectForConflict[nilDirect](t)

	expectNilDirectRenderFunctionPanic(t, func() {
		registerDirect[nilDirect](nil)
	})
}

func TestOutputRegistryRowsWithSingleToListHelperNoPartialRegistrationWhenRowsNil(t *testing.T) {
	type single struct {
		Data string
	}
	type list struct {
		Data []string
	}

	singleKey := reflect.TypeOf(&single{})
	listKey := reflect.TypeOf(&list{})
	cleanupRegistryTypes(t, singleKey, listKey)

	expectNilRowsFunctionPanic(t, func() {
		registerRowsWithSingleToListAdapter[single, list](nil)
	})

	assertRegistryTypesAbsent(t, singleKey, listKey)
}

func TestOutputRegistryRowsWithSingleToListHelperNilRowsPanicsBeforeConflictChecks(t *testing.T) {
	type single struct {
		Data string
	}
	type list struct {
		Data []string
	}

	singleKey := reflect.TypeOf(&single{})
	preregisterRowsForConflict[list](t, "value")
	cleanupRegistryTypes(t, singleKey)

	expectNilRowsFunctionPanic(t, func() {
		registerRowsWithSingleToListAdapter[single, list](nil)
	})

	assertRegistryTypeAbsent(t, singleKey)
}

func TestOutputRegistryRowsWithSingleToListHelperAdapterValidationPanicsBeforeConflictChecks(t *testing.T) {
	type single struct {
		Value string
	}
	type list struct {
		Data []string
	}

	singleKey := reflect.TypeOf(&single{})
	preregisterRowsForConflict[list](t, "value")
	cleanupRegistryTypes(t, singleKey)

	expectPanicContains(t, "requires Data field", func() {
		registerRowsWithSingleToListAdapter[single, list](func(v *list) ([]string, [][]string) {
			return []string{"value"}, nil
		})
	})

	assertRegistryTypeAbsent(t, singleKey)
}

func TestEnsureRegistryTypesAvailablePanicsOnDuplicateTypes(t *testing.T) {
	type duplicate struct{}
	key := typeKey[duplicate]()
	cleanupRegistryTypes(t, key)

	expectDuplicateRegistrationPanic(t, func() {
		ensureRegistryTypesAvailable(key, key)
	})

	assertRegistryTypeAbsent(t, key)
}

func TestEnsureRegistryTypeAvailablePanicsOnNilType(t *testing.T) {
	expectPanicContains(t, "invalid nil registration type", func() {
		ensureRegistryTypeAvailable(nil)
	})
}

func TestEnsureRegistryTypesAvailableDuplicatePanicIncludesType(t *testing.T) {
	type duplicate struct{}
	key := typeKey[duplicate]()
	cleanupRegistryTypes(t, key)

	expectPanicContains(t, key.String(), func() {
		ensureRegistryTypesAvailable(key, key)
	})
}

func TestEnsureRegistryTypesAvailablePanicsOnNilType(t *testing.T) {
	expectPanicContains(t, "invalid nil registration type", func() {
		ensureRegistryTypesAvailable(nil)
	})
}

func TestEnsureRegistryTypesAvailablePanicsWhenTypeAlreadyRegistered(t *testing.T) {
	type existing struct{}
	key := preregisterRowsForConflict[existing](t, "value")

	expectDuplicateRegistrationPanic(t, func() {
		ensureRegistryTypesAvailable(key)
	})
}

func TestEnsureRegistryTypesAvailablePanicsWhenDirectTypeAlreadyRegistered(t *testing.T) {
	type existing struct{}
	key := preregisterDirectForConflict[existing](t)

	expectDuplicateRegistrationPanic(t, func() {
		ensureRegistryTypesAvailable(key)
	})
}

func TestEnsureRegistryTypesAvailableAllowsEmptyInput(t *testing.T) {
	ensureRegistryTypesAvailable()
}

func TestIsRegistryTypeRegistered(t *testing.T) {
	type outputRegistered struct{}
	type directRegistered struct{}
	type missing struct{}

	outputKey := preregisterRowsForConflict[outputRegistered](t, "value")
	directKey := preregisterDirectForConflict[directRegistered](t)
	missingKey := typeKey[missing]()
	cleanupRegistryTypes(t, missingKey)

	if !isRegistryTypeRegistered(outputKey) {
		t.Fatalf("expected output-registered type %v to be present", outputKey)
	}
	if !isRegistryTypeRegistered(directKey) {
		t.Fatalf("expected direct-registered type %v to be present", directKey)
	}
	if isRegistryTypeRegistered(missingKey) {
		t.Fatalf("expected unregistered type %v to be absent", missingKey)
	}
}

func TestIsRegistryTypeRegisteredWithNilType(t *testing.T) {
	if isRegistryTypeRegistered(nil) {
		t.Fatal("expected nil type to be treated as unregistered")
	}
}

func expectPanic(t *testing.T, message string, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			t.Fatal(message)
		}
	}()
	fn()
}

func expectPanicContains(t *testing.T, want string, fn func()) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected panic containing %q", want)
		}
		got := fmt.Sprint(r)
		if !strings.Contains(got, want) {
			t.Fatalf("panic %q does not contain %q", r, want)
		}
	}()
	fn()
}

func expectDuplicateRegistrationPanic(t *testing.T, fn func()) {
	t.Helper()
	expectPanicContains(t, "duplicate registration", fn)
}

func expectNilRegistryPanic(t *testing.T, kind string, fn func()) {
	t.Helper()
	expectPanicContains(t, "nil "+kind, fn)
}

func expectNilRowsFunctionPanic(t *testing.T, fn func()) {
	t.Helper()
	expectNilRegistryPanic(t, "rows function", fn)
}

func expectNilDirectRenderFunctionPanic(t *testing.T, fn func()) {
	t.Helper()
	expectNilRegistryPanic(t, "direct render function", fn)
}

func renderWithRows(headers []string, rows [][]string) directRenderFunc {
	return func(data any, render func([]string, [][]string)) error {
		render(headers, rows)
		return nil
	}
}

func seedRowsAndDirectHandlersForTest(t reflect.Type, directFn directRenderFunc) func() bool {
	rowsHandlerCalled := false
	outputRegistry[t] = func(any) ([]string, [][]string, error) {
		rowsHandlerCalled = true
		return []string{"source"}, [][]string{{"rows"}}, nil
	}
	directRenderRegistry[t] = directFn

	return func() bool {
		return rowsHandlerCalled
	}
}

func assertRowContains(t *testing.T, headers []string, rows [][]string, minColumns int, expected ...string) {
	t.Helper()
	if len(headers) == 0 || len(rows) == 0 {
		t.Fatalf("expected non-empty headers/rows, got headers=%v rows=%v", headers, rows)
	}
	if len(rows[0]) < minColumns {
		t.Fatalf("expected at least %d columns in row, got row=%v", minColumns, rows[0])
	}
	joined := strings.Join(rows[0], " ")
	for _, want := range expected {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected row to contain %q, got row=%v", want, rows[0])
		}
	}
}

func assertSingleRowEquals(t *testing.T, headers []string, rows [][]string, wantHeaders []string, wantRow []string) {
	t.Helper()
	if !reflect.DeepEqual(headers, wantHeaders) {
		t.Fatalf("unexpected headers: got=%v want=%v", headers, wantHeaders)
	}
	if len(rows) != 1 {
		t.Fatalf("expected exactly 1 row, got %d (%v)", len(rows), rows)
	}
	if !reflect.DeepEqual(rows[0], wantRow) {
		t.Fatalf("unexpected row: got=%v want=%v", rows[0], wantRow)
	}
}

func cleanupRegistryTypes(t *testing.T, types ...reflect.Type) {
	t.Helper()
	t.Cleanup(func() {
		for _, typ := range types {
			delete(outputRegistry, typ)
			delete(directRenderRegistry, typ)
		}
	})
}

func assertRegistryTypeAbsent(t *testing.T, typ reflect.Type) {
	t.Helper()
	if _, exists := outputRegistry[typ]; exists {
		t.Fatalf("registry type %v should be absent from output registry", typ)
	}
	if _, exists := directRenderRegistry[typ]; exists {
		t.Fatalf("registry type %v should be absent from direct render registry", typ)
	}
}

func assertRegistryTypesAbsent(t *testing.T, types ...reflect.Type) {
	t.Helper()
	for _, typ := range types {
		assertRegistryTypeAbsent(t, typ)
	}
}

func requireOutputHandler(t *testing.T, typ reflect.Type, label string) rowsFunc {
	t.Helper()
	handler, ok := outputRegistry[typ]
	if !ok || handler == nil {
		t.Fatalf("expected %s handler for type %v", label, typ)
	}
	return handler
}

func registerRowsForConflict[T any](headers ...string) {
	if len(headers) == 0 {
		headers = []string{"value"}
	}

	registerRows(func(*T) ([]string, [][]string) {
		return headers, nil
	})
}

func preregisterRowsForConflict[T any](t *testing.T, headers ...string) reflect.Type {
	t.Helper()

	return preregisterConflictType[T](t, func() {
		registerRowsForConflict[T](headers...)
	})
}

func registerRowsErrForConflict[T any]() {
	registerRowsErr(func(*T) ([]string, [][]string, error) {
		return nil, nil, nil
	})
}

func preregisterRowsErrForConflict[T any](t *testing.T) reflect.Type {
	t.Helper()

	return preregisterConflictType[T](t, registerRowsErrForConflict[T])
}

func registerDirectForConflict[T any]() {
	registerDirect(func(*T, func([]string, [][]string)) error {
		return nil
	})
}

func preregisterDirectForConflict[T any](t *testing.T) reflect.Type {
	t.Helper()

	return preregisterConflictType[T](t, registerDirectForConflict[T])
}

func preregisterConflictType[T any](t *testing.T, register func()) reflect.Type {
	t.Helper()

	key := typeKey[T]()
	cleanupRegistryTypes(t, key)
	register()
	return key
}

func typeKey[T any]() reflect.Type {
	return reflect.TypeFor[*T]()
}
