package compare

import (
	"reflect"
	"unsafe"
)

// comparison holds the state of the Compare function, collecting errors
// and pointers that have already been compared.
type comparison struct {
	errs   *errorList
	visits map[visit]bool // track pointers already compared
}

type visit struct {
	got, want unsafe.Pointer
	typ       reflect.Type
}

// Compare compares the two given values, and if the comparison fails it returns
// an error that indicates where the two values differ. The ok return value reports
// whether the comparison passed or failed and is mainly useful in case when the
// err return value is unnecessary and discarded with _.
//
// The comparison algorithm is a copy of the one used by reflect.DeepEqual only
// split into multiple small functions.
func Compare(got, want interface{}) (err error, ok bool) {
	gotv := reflect.ValueOf(got)
	wantv := reflect.ValueOf(want)

	errlist := &errorList{}
	cmp := &comparison{
		errs:   errlist,
		visits: make(map[visit]bool),
	}

	p := path{rootnode{reflect.TypeOf(want)}}

	compare(gotv, wantv, cmp, p)
	if err = errlist.err(); err != nil {
		return err, false
	}
	return nil, true
}

func compare(got, want reflect.Value, cmp *comparison, p path) {
	if ok := compareValidity(got, want, cmp, p); !ok {
		return
	}
	if ok := compareType(got, want, cmp, p); !ok {
		return
	}
	if ok := checkVisited(got, want, cmp, p); !ok {
		return
	}

	switch got.Kind() {
	case reflect.Array:
		compareArray(got, want, cmp, p)
	case reflect.Slice:
		compareSlice(got, want, cmp, p)
	case reflect.Interface:
		compareInterface(got, want, cmp, p)
	case reflect.Ptr:
		comparePointer(got, want, cmp, p)
	case reflect.Struct:
		compareStruct(got, want, cmp, p)
	case reflect.Map:
		compareMap(got, want, cmp, p)
	case reflect.Func:
		compareFunc(got, want, cmp, p)
	default:
		compareInterfaceValue(got, want, cmp, p)
	}
}

// compareValidity compares the validity of the two values. The ok return value
// reports whether both of the values are valid effectively indicating that the
// comparison of the two values can continue.
func compareValidity(got, want reflect.Value, cmp *comparison, p path) (ok bool) {
	if got.IsValid() != want.IsValid() {
		cmp.errs.add(&validityError{got, want, p})
	}
	return got.IsValid() && want.IsValid()
}

// compareType compares the types of the two values. The ok return value reports
// whether the types are equal effectively indicating that the comparison of the
// two values can continue.
func compareType(got, want reflect.Value, cmp *comparison, p path) (ok bool) {
	if got.Type() != want.Type() {
		cmp.errs.add(&typeError{got, want, p})
		return false
	}
	return true
}

func hard(k reflect.Kind) bool {
	switch k {
	case reflect.Map, reflect.Slice, reflect.Ptr, reflect.Interface:
		return true
	}
	return false
}

// checkVisited checks whether the values, if they are addressable, have already
// been visited and if they haven't records a new visit into the visits map. The
// ok return value reports whether the comparison needs to continue or not.
func checkVisited(got, want reflect.Value, cmp *comparison, p path) (ok bool) {
	if got.CanAddr() && want.CanAddr() && hard(got.Kind()) {
		gotAddr := unsafe.Pointer(got.UnsafeAddr())
		wantAddr := unsafe.Pointer(want.UnsafeAddr())
		if uintptr(gotAddr) > uintptr(wantAddr) {
			gotAddr, wantAddr = wantAddr, gotAddr
		}

		typ := got.Type()
		v := visit{gotAddr, wantAddr, typ}
		if cmp.visits[v] {
			return false
		}
		cmp.visits[v] = true
	}
	return true
}

// compareSlice compares the address, length and contents of the two slice values.
func compareSlice(got, want reflect.Value, cmp *comparison, p path) {
	if got.Pointer() == want.Pointer() {
		return
	}
	if got.IsNil() != want.IsNil() {
		cmp.errs.add(&nilError{got, want, p})
		return
	}
	compareArray(got, want, cmp, p)
}

// compareArray compares the length and contents of the two array values.
func compareArray(got, want reflect.Value, cmp *comparison, p path) {
	if got.Len() != want.Len() {
		cmp.errs.add(&lenError{got, want, p})
		// TODO(mkopriva): might be good to compare the contents and
		// point out the "missing" or the "extra" elements...
		return
	}
	for i := 0; i < got.Len(); i++ {
		q := p.add(arrnode{i})
		ithGot := got.Index(i)
		ithWant := want.Index(i)
		compare(ithGot, ithWant, cmp, q)
	}
}

// compareInterface compares the underlying element values of the two interface values.
func compareInterface(got, want reflect.Value, cmp *comparison, p path) {
	if got.IsNil() != want.IsNil() {
		cmp.errs.add(&nilError{got, want, p})
		return
	}
	got = got.Elem()
	want = want.Elem()
	compare(got, want, cmp, p)
}

// comparePointer compares the values pointed to by the two given pointer values.
func comparePointer(got, want reflect.Value, cmp *comparison, p path) {
	if got.Pointer() == want.Pointer() {
		return
	}
	got = got.Elem()
	want = want.Elem()
	compare(got, want, cmp, p)
}

// compareStruct compares the corresponding fields of the two given struct values.
func compareStruct(got, want reflect.Value, cmp *comparison, p path) {
	for i, n := 0, got.NumField(); i < n; i++ {
		q := p.add(structnode{got.Type().Field(i).Name})
		fieldGot := got.Field(i)
		fieldWant := want.Field(i)
		compare(fieldGot, fieldWant, cmp, q)
	}
}

// compareMap compares the length and contents of the two given map values.
func compareMap(got, want reflect.Value, cmp *comparison, p path) {
	if got.Pointer() == want.Pointer() {
		return
	}
	if got.IsNil() != want.IsNil() {
		cmp.errs.add(&nilError{got, want, p})
		return
	}
	if got.Len() != want.Len() {
		cmp.errs.add(&lenError{got, want, p})
		// TODO(mkopriva): might be good to compare the contents and
		// point out the "missing" or the "extra" elements...
		return
	}

	for _, key := range want.MapKeys() {
		q := p.add(mapnode{key})
		valGot := got.MapIndex(key)
		valWant := want.MapIndex(key)

		if !valGot.IsValid() || !valWant.IsValid() {
			cmp.errs.add(&validityError{valGot, valWant, q})
			continue
		}
		compare(valGot, valWant, cmp, q)
	}
}

// compareFunc only checks whether the two given func values are nil.
func compareFunc(got, want reflect.Value, cmp *comparison, p path) {
	if !got.IsNil() || !want.IsNil() {
		cmp.errs.add(&funcError{got, want, p})
	}
}

// compareInterfaceValue compares the two given values as normal interface values.
func compareInterfaceValue(got, want reflect.Value, cmp *comparison, p path) {
	if g, w := valueInterface(got), valueInterface(want); g != w {
		cmp.errs.add(&valueError{g, w, p})
	}
}

func valueInterface(v reflect.Value) interface{} {
	switch v.Kind() {
	case reflect.Bool:
		return v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int()
	case reflect.Uint, reflect.Uintptr, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint()
	case reflect.Float32, reflect.Float64:
		return v.Float()
	case reflect.Complex64, reflect.Complex128:
		return v.Complex()
	case reflect.String:
		return v.String()
	case reflect.UnsafePointer:
		return v.Pointer()
	}
	return v.Interface()
}
