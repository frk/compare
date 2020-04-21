// The package compare facilitates the comparison of two Go values providing
// a detailed error message in case the comparison fails.
package compare

import (
	"reflect"
	"unsafe"
)

// Compare is a wrapper around DefaultConfig.Compare.
func Compare(got, want interface{}) error {
	return DefaultConfig.Compare(got, want)
}

// Config specifies the configuration for the value comparison.
type Config struct {
	// If IgnoreArrayOrder is set, the order of elements inside arrays and
	// slices is ignored. That is, two array/slice values are equal if they
	// have the same number of elements and each element in one array value
	// has an equivalent element in the other array value.
	IgnoreArrayOrder bool

	// The tag name to be checked by Compare for optional comparison rules.
	// If ObserveFieldTag is set, its value will be used as the name of the tag
	// to be checked, if it is empty then no tag will be checked.
	//
	// Currently the only optional rules are:
	// "-": The minus option omits a field from comparison.
	// "+": The plus option omits field *value* comparison, however, it does
	//      compare the fields' "zero-ness", that is, it checks whether both
	//      fields are zero or whether they are both non-zero.
	ObserveFieldTag string
}

// DefaultConfig is the default Config used by Compare.
var DefaultConfig Config

// comparison holds the state of the Compare function, collecting errors
// and pointers that have already been compared.
type comparison struct {
	errs   *errorList
	visits map[visit]bool // track pointers already compared
	zero   bool
}

func newComparison() *comparison {
	cmp := new(comparison)
	cmp.errs = new(errorList)
	cmp.visits = make(map[visit]bool)
	return cmp
}

type visit struct {
	got  unsafe.Pointer
	want unsafe.Pointer
	typ  reflect.Type
}

// Compare compares the two given values, and if the comparison fails it returns
// an error that indicates where the two values differ.
//
// The comparison algorithm is a copy of the one used by reflect.DeepEqual only
// split into multiple small functions.
func (conf Config) Compare(got, want interface{}) error {
	gotv := reflect.ValueOf(got)
	wantv := reflect.ValueOf(want)

	p := path{rootnode{reflect.TypeOf(want)}}
	cmp := newComparison()
	conf.compare(gotv, wantv, cmp, p)
	return cmp.errs.err()
}

func (conf Config) compare(got, want reflect.Value, cmp *comparison, p path) {
	if ok := conf.compareValidity(got, want, cmp, p); !ok {
		return
	}
	if ok := conf.compareType(got, want, cmp, p); !ok {
		return
	}
	if ok := conf.checkVisited(got, want, cmp, p); !ok {
		return
	}

	if cmp.zero {
		conf.compareZero(got, want, cmp, p)
		return
	}

	switch got.Kind() {
	case reflect.Array:
		conf.compareArray(got, want, cmp, p)
	case reflect.Slice:
		conf.compareSlice(got, want, cmp, p)
	case reflect.Interface:
		conf.compareInterface(got, want, cmp, p)
	case reflect.Ptr:
		conf.comparePointer(got, want, cmp, p)
	case reflect.Struct:
		conf.compareStruct(got, want, cmp, p)
	case reflect.Map:
		conf.compareMap(got, want, cmp, p)
	case reflect.Func:
		conf.compareFunc(got, want, cmp, p)
	case reflect.String:
		conf.compareString(got, want, cmp, p)
	case reflect.Chan:
		conf.compareChan(got, want, cmp, p)
	default:
		conf.compareInterfaceValue(got, want, cmp, p)
	}
}

func (conf Config) equals(got, want reflect.Value) bool {
	p := make(path, 0)
	cmp := newComparison()
	conf.compare(got, want, cmp, p)
	return len(cmp.errs.List) == 0
}

// compareValidity compares the validity of the two values. The ok return value
// reports whether both of the values are valid effectively indicating that the
// comparison of the two values can continue.
func (conf Config) compareValidity(got, want reflect.Value, cmp *comparison, p path) (ok bool) {
	if got.IsValid() != want.IsValid() {
		cmp.errs.add(&validityError{got, want, p})
	}
	return got.IsValid() && want.IsValid()
}

// compareType compares the types of the two values. The ok return value reports
// whether the types are equal effectively indicating that the comparison of the
// two values can continue.
func (conf Config) compareType(got, want reflect.Value, cmp *comparison, p path) (ok bool) {
	if got.Type() != want.Type() {
		cmp.errs.add(&typeError{got, want, p})
		return false
	}
	return true
}

func (conf Config) hard(k reflect.Kind) bool {
	switch k {
	case reflect.Map, reflect.Slice, reflect.Ptr, reflect.Interface:
		return true
	}
	return false
}

// checkVisited checks whether the values, if they are addressable, have already
// been visited and if they haven't records a new visit into the visits map. The
// ok return value reports whether the comparison needs to continue or not.
func (conf Config) checkVisited(got, want reflect.Value, cmp *comparison, p path) (ok bool) {
	if got.CanAddr() && want.CanAddr() && conf.hard(got.Kind()) {
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
func (conf Config) compareSlice(got, want reflect.Value, cmp *comparison, p path) {
	if got.Pointer() == want.Pointer() {
		return
	}
	if got.IsNil() != want.IsNil() {
		cmp.errs.add(&nilError{got, want, p})
		return
	}
	conf.compareArray(got, want, cmp, p)
}

// compareArray compares the length and contents of the two array values.
func (conf Config) compareArray(got, want reflect.Value, cmp *comparison, p path) {
	if got.Len() != want.Len() {
		cmp.errs.add(&lenError{got, want, p})
		// TODO(mkopriva): might be good to compare the contents and
		// point out the "missing" or the "extra" elements...
		return
	}

	if conf.IgnoreArrayOrder {
		conf.compareArrayIgnoreOrder(got, want, cmp, p)
		return
	}

	for i := 0; i < want.Len(); i++ {
		q := p.add(arrnode{i})
		ithGot := got.Index(i)
		ithWant := want.Index(i)
		conf.compare(ithGot, ithWant, cmp, q)
	}
}

func (conf Config) compareArrayIgnoreOrder(got, want reflect.Value, cmp *comparison, p path) {
	gotidx := make([]int, got.Len())
	for i := range gotidx {
		gotidx[i] = i
	}

	for i := 0; i < want.Len(); i++ {
		q := p.add(arrnode{i})
		ithWant := want.Index(i)

		var foundEqual bool
		for i, j := range gotidx {
			ithGot := got.Index(j)
			if conf.equals(ithGot, ithWant) {
				gotidx = append(gotidx[:i], gotidx[i+1:]...)
				foundEqual = true
				break
			}
		}
		if !foundEqual {
			gotNil := reflect.ValueOf((*interface{})(nil))
			cmp.errs.add(&nilError{gotNil, ithWant, q})
		}
	}
}

// compareInterface compares the underlying element values of the two interface values.
func (conf Config) compareInterface(got, want reflect.Value, cmp *comparison, p path) {
	if got.IsNil() != want.IsNil() {
		cmp.errs.add(&nilError{got, want, p})
		return
	}
	got = got.Elem()
	want = want.Elem()
	conf.compare(got, want, cmp, p)
}

// comparePointer compares the values pointed to by the two given pointer values.
func (conf Config) comparePointer(got, want reflect.Value, cmp *comparison, p path) {
	if got.Pointer() == want.Pointer() {
		return
	}
	got = got.Elem()
	want = want.Elem()
	conf.compare(got, want, cmp, p)
}

// compareStruct compares the corresponding fields of the two given struct values.
func (conf Config) compareStruct(got, want reflect.Value, cmp *comparison, p path) {
	if structIsTime(got) {
		// CanInterface is used here to determine whether or not
		// the value was obtained from an unexported field.
		if m := got.MethodByName("Equal"); m.CanInterface() {
			if !m.Call([]reflect.Value{want})[0].Bool() {
				cmp.errs.add(&valueError{got, want, p})
			}
			return
		}
	}

	for i, n := 0, want.NumField(); i < n; i++ {
		f := want.Type().Field(i)
		if len(conf.ObserveFieldTag) > 0 {
			if f.Tag.Get(conf.ObserveFieldTag) == "-" {
				continue
			}
			if f.Tag.Get(conf.ObserveFieldTag) == "+" {
				cmp.zero = true
			}
		}
		q := p.add(structnode{f.Name})
		fieldGot := got.Field(i)
		fieldWant := want.Field(i)
		conf.compare(fieldGot, fieldWant, cmp, q)
	}
}

// compareMap compares the length and contents of the two given map values.
func (conf Config) compareMap(got, want reflect.Value, cmp *comparison, p path) {
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
		conf.compare(valGot, valWant, cmp, q)
	}
}

// compareFunc only checks whether the two given func values are nil.
func (conf Config) compareFunc(got, want reflect.Value, cmp *comparison, p path) {
	if !got.IsNil() || !want.IsNil() {
		cmp.errs.add(&funcError{got, want, p})
	}
}

// compareString
func (conf Config) compareString(got, want reflect.Value, cmp *comparison, p path) {
	gots, wants := got.String(), want.String()
	if gots == wants {
		return
	}
	cmp.errs.add(newStringError(gots, wants, p))
}

// compareChan
func (conf Config) compareChan(got, want reflect.Value, cmp *comparison, p path) {
	if got.Len() != want.Len() {
		cmp.errs.add(&lenError{got, want, p})
		// TODO(mkopriva): might be good to compare the contents and
		// point out the "missing" or the "extra" elements...
		return
	}

	if length := want.Len(); length > 0 {
		for i := 1; i <= length; i++ {
			q := p.add(channode{i})
			ithGot, _ := got.Recv()
			ithWant, _ := want.Recv()
			conf.compare(ithGot, ithWant, cmp, q)
		}
	}
}

// compareInterfaceValue compares the two given values as normal interface{} values.
func (conf Config) compareInterfaceValue(got, want reflect.Value, cmp *comparison, p path) {
	if g, w := valueInterface(got), valueInterface(want); g != w {
		cmp.errs.add(&valueError{g, w, p})
	}
}

// compareZero checks whether the two given values are both zero or both non-zero values.
func (conf Config) compareZero(got, want reflect.Value, cmp *comparison, p path) {
	if g, w := got.IsZero(), want.IsZero(); g != w {
		cmp.errs.add(&zeroError{g, w, p})
	}
	cmp.zero = false
}

func structIsTime(v reflect.Value) bool {
	typ := v.Type()
	return typ.PkgPath() == "time" && typ.Name() == "Time"
}

func valueInterface(v reflect.Value) interface{} {
	switch v.Kind() {
	case reflect.Bool:
		return v.Bool()
	case reflect.Int:
		return int(v.Int())
	case reflect.Int8:
		return int8(v.Int())
	case reflect.Int16:
		return int16(v.Int())
	case reflect.Int32:
		return int32(v.Int())
	case reflect.Int64:
		return v.Int()
	case reflect.Uint, reflect.Uintptr, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint()
	case reflect.Float32:
		return float32(v.Float())
	case reflect.Float64:
		return float64(v.Float())
	case reflect.Complex64, reflect.Complex128:
		return v.Complex()
	case reflect.String:
		return v.String()
	case reflect.UnsafePointer:
		return v.Pointer()
	}
	return v.Interface()
}
