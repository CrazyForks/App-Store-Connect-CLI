package asc

import (
	"fmt"
	"reflect"
)

// rowsFunc extracts headers and rows from a typed response value.
type rowsFunc func(data any) ([]string, [][]string, error)

// directRenderFunc renders the value using the provided render callback.
// Used for multi-table types that need to call render more than once.
type directRenderFunc func(data any, render func([]string, [][]string)) error

// outputRegistry maps concrete pointer types to their rows-extraction function.
var outputRegistry = map[reflect.Type]rowsFunc{}

// directRenderRegistry maps types that need direct render control (multi-table output).
var directRenderRegistry = map[reflect.Type]directRenderFunc{}

func typeForPtr[T any]() reflect.Type {
	return reflect.TypeFor[*T]()
}

func typeFor[T any]() reflect.Type {
	return reflect.TypeFor[T]()
}

func panicNilHelperFunction(kind string, t reflect.Type) {
	panic(fmt.Sprintf("output registry: nil %s for %s", kind, t))
}

func panicDuplicateRegistration(t reflect.Type) {
	panic(fmt.Sprintf("output registry: duplicate registration for %s", t))
}

func panicInvalidRegistryType() {
	panic("output registry: invalid nil registration type")
}

func panicSingleListAdapterStructRequirement(kind string, t reflect.Type) {
	panic(fmt.Sprintf("output registry: single/list adapter %s type must be a struct: %s", kind, t))
}

func panicSingleListAdapter(reason string) {
	panic("output registry: single/list adapter " + reason)
}

func panicSingleListAdapterDataTypeMismatch(source, target reflect.Type) {
	panic(fmt.Sprintf(
		"output registry: adapter Data type mismatch source=%s target=%s",
		source,
		target,
	))
}

func isRegistryTypeRegistered(t reflect.Type) bool {
	if _, exists := outputRegistry[t]; exists {
		return true
	}
	if _, exists := directRenderRegistry[t]; exists {
		return true
	}
	return false
}

func ensureRegistryTypeAvailable(t reflect.Type) {
	if t == nil {
		panicInvalidRegistryType()
	}
	if isRegistryTypeRegistered(t) {
		panicDuplicateRegistration(t)
	}
}

func ensureRegistryTypesAvailable(types ...reflect.Type) {
	seen := make(map[reflect.Type]struct{}, len(types))
	for _, t := range types {
		if _, exists := seen[t]; exists {
			panicDuplicateRegistration(t)
		}
		seen[t] = struct{}{}
		ensureRegistryTypeAvailable(t)
	}
}

// registerRows registers a rows function for the given pointer type.
// The function must accept a pointer and return (headers, rows).
func registerRows[T any](fn func(*T) ([]string, [][]string)) {
	t := typeForPtr[T]()
	if fn == nil {
		panicNilHelperFunction("rows function", t)
	}
	ensureRegistryTypeAvailable(t)
	outputRegistry[t] = func(data any) ([]string, [][]string, error) {
		h, r := fn(data.(*T))
		return h, r, nil
	}
}

// registerRowsErr registers a rows function that can return an error.
func registerRowsErr[T any](fn func(*T) ([]string, [][]string, error)) {
	t := typeForPtr[T]()
	if fn == nil {
		panicNilHelperFunction("rows function", t)
	}
	ensureRegistryTypeAvailable(t)
	outputRegistry[t] = func(data any) ([]string, [][]string, error) {
		return fn(data.(*T))
	}
}

func registerSingleLinkageRows[T any](extract func(*T) ResourceData) {
	t := typeForPtr[T]()
	if extract == nil {
		panicNilHelperFunction("linkage extractor", t)
	}
	registerRows(func(v *T) ([]string, [][]string) {
		return linkagesRows(&LinkagesResponse{Data: []ResourceData{extract(v)}})
	})
}

func registerIDStateRows[T any](extract func(*T) (string, string), rows func(string, string) ([]string, [][]string)) {
	t := typeForPtr[T]()
	if extract == nil {
		panicNilHelperFunction("id/state extractor", t)
	}
	if rows == nil {
		panicNilHelperFunction("id/state rows function", t)
	}
	registerRows(func(v *T) ([]string, [][]string) {
		id, state := extract(v)
		return rows(id, state)
	})
}

func registerIDBoolRows[T any](extract func(*T) (string, bool), rows func(string, bool) ([]string, [][]string)) {
	t := typeForPtr[T]()
	if extract == nil {
		panicNilHelperFunction("id/bool extractor", t)
	}
	if rows == nil {
		panicNilHelperFunction("id/bool rows function", t)
	}
	registerRows(func(v *T) ([]string, [][]string) {
		id, deleted := extract(v)
		return rows(id, deleted)
	})
}

func registerResponseDataRows[T any](rows func([]Resource[T]) ([]string, [][]string)) {
	t := typeForPtr[Response[T]]()
	if rows == nil {
		panicNilHelperFunction("response-data rows function", t)
	}
	registerRows(func(v *Response[T]) ([]string, [][]string) {
		return rows(v.Data)
	})
}

// registerSingleResourceRowsAdapter registers rows rendering for list renderers
// by adapting SingleResponse[T] into Response[T] with one item in Data.
func registerSingleResourceRowsAdapter[T any](rows func(*Response[T]) ([]string, [][]string)) {
	t := typeForPtr[SingleResponse[T]]()
	if rows == nil {
		panicNilHelperFunction("rows function", t)
	}
	registerRows(func(v *SingleResponse[T]) ([]string, [][]string) {
		return rows(&Response[T]{Data: []Resource[T]{v.Data}})
	})
}

// registerRowsWithSingleResourceAdapter registers both list and single handlers
// for row renderers that operate on Response[T].
func registerRowsWithSingleResourceAdapter[T any](rows func(*Response[T]) ([]string, [][]string)) {
	listType := typeForPtr[Response[T]]()
	singleType := typeForPtr[SingleResponse[T]]()
	if rows == nil {
		panicNilHelperFunction("rows function", listType)
	}
	ensureRegistryTypesAvailable(listType, singleType)
	registerRows(rows)
	registerSingleResourceRowsAdapter(rows)
}

// registerSingleToListRowsAdapter registers rows rendering by adapting a single
// response struct into a corresponding list response struct using shared field
// names. The source type must expose `Data` and may expose `Links`; the target
// type must expose `Data` as a slice and may expose `Links`.
func registerSingleToListRowsAdapter[T any, U any](rows func(*U) ([]string, [][]string)) {
	registerRows(singleToListRowsAdapter[T, U](rows))
}

func singleToListRowsAdapter[T any, U any](rows func(*U) ([]string, [][]string)) func(*T) ([]string, [][]string) {
	if rows == nil {
		panicNilHelperFunction("rows function", typeForPtr[U]())
	}

	sourceType := typeFor[T]()
	targetType := typeFor[U]()
	if sourceType.Kind() != reflect.Struct {
		panicSingleListAdapterStructRequirement("source", sourceType)
	}
	if targetType.Kind() != reflect.Struct {
		panicSingleListAdapterStructRequirement("target", targetType)
	}

	sourceDataField, sourceHasData := sourceType.FieldByName("Data")
	targetDataField, targetHasData := targetType.FieldByName("Data")
	if !sourceHasData || !targetHasData {
		panicSingleListAdapter("requires Data field on source and target")
	}
	if targetDataField.Type.Kind() != reflect.Slice {
		panicSingleListAdapter("target Data field must be a slice")
	}
	targetElemType := targetDataField.Type.Elem()
	if !sourceDataField.Type.AssignableTo(targetElemType) {
		panicSingleListAdapterDataTypeMismatch(sourceDataField.Type, targetElemType)
	}

	sourceLinksField, sourceHasLinks := sourceType.FieldByName("Links")
	targetLinksField, targetHasLinks := targetType.FieldByName("Links")
	copyLinks := sourceHasLinks &&
		targetHasLinks &&
		sourceLinksField.Type.AssignableTo(targetLinksField.Type)

	return func(v *T) ([]string, [][]string) {
		source := reflect.ValueOf(v).Elem()
		var target U
		targetValue := reflect.ValueOf(&target).Elem()

		sourceData := source.FieldByIndex(sourceDataField.Index)
		targetData := targetValue.FieldByIndex(targetDataField.Index)

		rowsSlice := reflect.MakeSlice(targetData.Type(), 1, 1)
		rowsSlice.Index(0).Set(sourceData)
		targetData.Set(rowsSlice)

		if copyLinks {
			sourceLinks := source.FieldByIndex(sourceLinksField.Index)
			targetLinks := targetValue.FieldByIndex(targetLinksField.Index)
			targetLinks.Set(sourceLinks)
		}

		return rows(&target)
	}
}

// registerRowsWithSingleToListAdapter registers both list and single handlers
// when list rendering expects a concrete list response type.
func registerRowsWithSingleToListAdapter[T any, U any](rows func(*U) ([]string, [][]string)) {
	listType := typeForPtr[U]()
	singleType := typeForPtr[T]()
	if rows == nil {
		panicNilHelperFunction("rows function", listType)
	}
	adapter := singleToListRowsAdapter[T, U](rows)
	ensureRegistryTypesAvailable(listType, singleType)
	registerRows(rows)
	registerRows(adapter)
}

// registerDirect registers a type that needs direct render control (multi-table output).
func registerDirect[T any](fn func(*T, func([]string, [][]string)) error) {
	t := typeForPtr[T]()
	if fn == nil {
		panicNilHelperFunction("direct render function", t)
	}
	ensureRegistryTypeAvailable(t)
	directRenderRegistry[t] = func(data any, render func([]string, [][]string)) error {
		return fn(data.(*T), render)
	}
}

// renderByRegistry looks up the rows function for the given value and renders
// using the provided render function (RenderTable or RenderMarkdown).
// Falls back to JSON output for unregistered types.
func renderByRegistry(data any, render func([]string, [][]string)) error {
	t := reflect.TypeOf(data)

	// Check direct render registry first (multi-table types).
	if fn, ok := directRenderRegistry[t]; ok {
		return fn(data, render)
	}

	// Standard single-table types.
	if fn, ok := outputRegistry[t]; ok {
		h, r, err := fn(data)
		if err != nil {
			return err
		}
		render(h, r)
		return nil
	}

	return PrintJSON(data)
}
