package codec

import (
	"fmt"
	"math"
	"reflect"
	"sync"
	"time"
)

// Marshal returns the OPCUA encoding of v.
func Marshal(v any) ([]byte, error) {
	e := newEncodeState()
	defer encodeStatePool.Put(e)

	err := e.marshal(v)
	if err != nil {
		return nil, err
	}
	buf := append([]byte(nil), e.Bytes()...)

	return buf, nil
}

// Encoder is the interface implemented by types that
// can marshal themselves into valid OPCUA.
type Encoder interface {
	EncodeOPCUA(s *Stream) error
}

// An UnsupportedTypeError is returned by [Marshal] when attempting
// to encode an unsupported value type.
type UnsupportedTypeError struct {
	Type reflect.Type
}

func (e *UnsupportedTypeError) Error() string {
	return "opcua: unsupported type: " + e.Type.String()
}

// An UnsupportedValueError is returned by [Marshal] when attempting
// to encode an unsupported value.
type UnsupportedValueError struct {
	Value reflect.Value
	Str   string
}

func (e *UnsupportedValueError) Error() string {
	return "opcua: unsupported value: " + e.Str
}

// A MarshalerError represents an error from calling a
// [Encoder.EncodeOPCUA] method.
type MarshalerError struct {
	Type       reflect.Type
	Err        error
	sourceFunc string
}

func (e *MarshalerError) Error() string {
	srcFunc := e.sourceFunc
	if srcFunc == "" {
		srcFunc = "EncodeOPCUA"
	}
	return "opcua: error calling " + srcFunc +
		" for type " + e.Type.String() +
		": " + e.Err.Error()
}

type encodeState struct {
	Stream

	ptrLevel uint
	ptrSeen  map[any]struct{}
}

const startDetectingCyclesAfter = 1000

var encodeStatePool sync.Pool

func newEncodeState() *encodeState {
	if v := encodeStatePool.Get(); v != nil {
		e := v.(*encodeState)
		e.Reset()
		if len(e.ptrSeen) > 0 {
			panic("ptrEncoder.encode should have emptied ptrSeen via defers")
		}
		e.ptrLevel = 0
		return e
	}
	return &encodeState{
		Stream:  Stream{buf: make([]byte, 0, 256)},
		ptrSeen: make(map[any]struct{}),
	}
}

// codecError is an error wrapper type for internal use only.
// Panics with errors are wrapped in codecError so that the top-level recover
// can distinguish intentional panics from this package.
type codecError struct{ error }

func (e *encodeState) marshal(v any) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if je, ok := r.(codecError); ok {
				err = je.error
			} else {
				panic(r)
			}
		}
	}()
	e.reflectValue(reflect.ValueOf(v))
	return nil
}

// error aborts the encoding by panicking with err wrapped in jsonError.
func (e *encodeState) error(err error) {
	panic(codecError{err})
}

func (e *encodeState) reflectValue(v reflect.Value) {
	valueEncoder(v)(e, v)
}

type encoderFunc func(e *encodeState, v reflect.Value)

var encoderCache sync.Map // map[reflect.Type]encoderFunc

func valueEncoder(v reflect.Value) encoderFunc {
	if isTime(v) {
		return timeEncoder
	}

	return typeEncoder(v.Type())
}

func typeEncoder(t reflect.Type) encoderFunc {
	if fi, ok := encoderCache.Load(t); ok {
		return fi.(encoderFunc)
	}

	// To deal with recursive types, populate the map with an
	// indirect func before we build it. This type waits on the
	// real func (f) to be ready and then calls it. This indirect
	// func is only used for recursive types.
	var (
		wg sync.WaitGroup
		f  encoderFunc
	)
	wg.Add(1)
	fi, loaded := encoderCache.LoadOrStore(t, encoderFunc(func(e *encodeState, v reflect.Value) {
		wg.Wait()
		f(e, v)
	}))
	if loaded {
		return fi.(encoderFunc)
	}

	// Compute the real encoder and replace the indirect func with it.
	f = newTypeEncoder(t, true)
	wg.Done()
	encoderCache.Store(t, f)
	return f
}

var encoderType = reflect.TypeFor[Encoder]()

// newTypeEncoder constructs an encoderFunc for a type.
// The returned encoder only checks CanAddr when allowAddr is true.
func newTypeEncoder(t reflect.Type, allowAddr bool) encoderFunc {
	kind := t.Kind()
	// If we have a non-pointer value whose type implements
	// Marshaler with a value receiver, then we're better off taking
	// the address of the value - otherwise we end up with an
	// allocation as we cast the value to an interface.
	if kind != reflect.Pointer && allowAddr && reflect.PointerTo(t).Implements(encoderType) {
		return newCondAddrEncoder(addrMarshalerEncoder, newTypeEncoder(t, false))
	}
	if t.Implements(encoderType) {
		return marshalerEncoder
	}

	switch kind {
	case reflect.Bool:
		return boolEncoder
	case reflect.Int8:
		return int8Encoder
	case reflect.Uint8:
		return uint8Encoder
	case reflect.Int16:
		return int16Encoder
	case reflect.Uint16:
		return uint16Encoder
	case reflect.Int32:
		return int32Encoder
	case reflect.Uint32:
		return uint32Encoder
	case reflect.Int64:
		return int64Encoder
	case reflect.Uint64:
		return uint64Encoder
	case reflect.Float32:
		return float32Encoder
	case reflect.Float64:
		return float64Encoder
	case reflect.String:
		return stringEncoder
	case reflect.Interface:
		return interfaceEncoder
	case reflect.Ptr:
		return newPtrEncoder(t)
	case reflect.Struct:
		return newStructEncoder(t)
	case reflect.Array:
		return newArrayEncoder(t)
	case reflect.Slice:
		return newSliceEncoder(t)
	default:
		return unsupportedTypeEncoder
	}
}

func isTime(val reflect.Value) bool {
	return val.CanConvert(timeType)
}

func marshalerEncoder(e *encodeState, v reflect.Value) {
	if v.Kind() == reflect.Pointer && v.IsNil() {
		return
	}
	m, ok := v.Interface().(Encoder)
	if !ok {
		return
	}
	err := m.EncodeOPCUA(&e.Stream)
	if err != nil {
		e.error(&MarshalerError{v.Type(), err, "EncodeOPCUA"})
	}
}

func addrMarshalerEncoder(e *encodeState, v reflect.Value) {
	va := v.Addr()
	if va.IsNil() {
		e.WriteUint32(NULL)
		return
	}
	m := va.Interface().(Encoder)
	err := m.EncodeOPCUA(&e.Stream)
	if err != nil {
		e.error(&MarshalerError{v.Type(), err, "EncodeOPCUA"})
	}
}

func boolEncoder(e *encodeState, v reflect.Value) {
	val := v.Bool()
	if val {
		e.WriteByte(1)
	} else {
		e.WriteByte(0)
	}
}

func int8Encoder(e *encodeState, v reflect.Value) {
	val := int8(v.Int())
	e.WriteByte(byte(val))
}

func uint8Encoder(e *encodeState, v reflect.Value) {
	val := uint8(v.Uint())
	e.WriteByte(val)
}

func int16Encoder(e *encodeState, v reflect.Value) {
	val := uint16(v.Int())
	e.WriteUint16(val)
}

func uint16Encoder(e *encodeState, v reflect.Value) {
	val := uint16(v.Uint())
	e.WriteUint16(val)
}

func int32Encoder(e *encodeState, v reflect.Value) {
	val := uint32(v.Int())
	e.WriteUint32(val)
}

func uint32Encoder(e *encodeState, v reflect.Value) {
	val := uint32(v.Uint())
	e.WriteUint32(val)
}

func int64Encoder(e *encodeState, v reflect.Value) {
	val := uint64(v.Int())
	e.WriteUint64(val)
}

func uint64Encoder(e *encodeState, v reflect.Value) {
	val := v.Uint()
	e.WriteUint64(val)
}

func float32Encoder(e *encodeState, v reflect.Value) {
	if math.IsNaN(v.Float()) {
		e.WriteUint32(F32QNAN)
	} else {
		val := math.Float32bits((float32)(v.Float()))
		e.WriteUint32(val)
	}
}

func float64Encoder(e *encodeState, v reflect.Value) {
	if math.IsNaN(v.Float()) {
		e.WriteUint64(F64QNAN)
	} else {
		val := math.Float64bits(v.Float())
		e.WriteUint64(val)
	}
}

func stringEncoder(e *encodeState, v reflect.Value) {
	s := v.String()
	if s == "" {
		e.WriteUint32(NULL)
		return
	}

	l := len(s)
	e.WriteUint32(uint32(l))
	e.Write([]byte(s))
}

var timeType = reflect.TypeOf(time.Time{})

func timeEncoder(e *encodeState, v reflect.Value) {
	var ts uint64
	val := v.Convert(timeType).Interface().(time.Time)
	if !v.IsZero() {
		// encode time in "100 nanosecond intervals since January 1, 1601"
		ts = uint64(val.UTC().UnixNano()/100 + 116444736000000000)
	}
	e.WriteUint64(ts)
}

func interfaceEncoder(e *encodeState, v reflect.Value) {
	if v.IsNil() {
		return
	}
	e.reflectValue(v.Elem())
}

func unsupportedTypeEncoder(e *encodeState, v reflect.Value) {
	e.error(&UnsupportedTypeError{v.Type()})
}

type structEncoder struct {
	fields structFields
}

type structFields struct {
	list []field
}

func (se structEncoder) encode(e *encodeState, v reflect.Value) {
FieldLoop:
	for i := range se.fields.list {
		f := &se.fields.list[i]

		// Find the nested struct field by following f.index.
		fv := v
		for _, i := range f.index {
			if fv.Kind() == reflect.Pointer {
				if fv.IsNil() {
					continue FieldLoop
				}
				fv = fv.Elem()
			}
			fv = fv.Field(i)
		}

		f.encoder(e, fv)
	}
}

func newStructEncoder(t reflect.Type) encoderFunc {
	se := structEncoder{fields: cachedTypeFields(t)}
	return se.encode
}

func encodeByteSlice(e *encodeState, v reflect.Value) {
	if v.IsNil() {
		e.WriteUint32(NULL)
		return
	}

	n := v.Len()
	e.WriteUint32(uint32(n))

	b := make([]byte, n)
	reflect.Copy(reflect.ValueOf(b), v)
	e.Write(b)
}

// sliceEncoder just wraps an arrayEncoder, checking to make sure the value isn't nil.
type sliceEncoder struct {
	arrayEnc encoderFunc
}

func (se sliceEncoder) encode(e *encodeState, v reflect.Value) {
	if v.IsNil() {
		e.WriteUint32(NULL)
		return
	}

	if v.Len() > math.MaxInt32 {
		panic("array too large")
	}

	if e.ptrLevel++; e.ptrLevel > startDetectingCyclesAfter {
		// We're a large number of nested ptrEncoder.encode calls deep;
		// start checking if we've run into a pointer cycle.
		// Here we use a struct to memorize the pointer to the first element of the slice
		// and its length.
		ptr := struct {
			ptr interface{} // always an unsafe.Pointer, but avoids a dependency on package unsafe
			len int
		}{v.UnsafePointer(), v.Len()}
		if _, ok := e.ptrSeen[ptr]; ok {
			e.error(&UnsupportedValueError{v, fmt.Sprintf("encountered a cycle via %s", v.Type())})
		}
		e.ptrSeen[ptr] = struct{}{}
		defer delete(e.ptrSeen, ptr)
	}
	se.arrayEnc(e, v)
	e.ptrLevel--
}

func newSliceEncoder(t reflect.Type) encoderFunc {
	// Byte slices get special treatment; arrays don't.
	if t.Elem().Kind() == reflect.Uint8 {
		p := reflect.PointerTo(t.Elem())
		if !p.Implements(encoderType) {
			return encodeByteSlice
		}
	}
	enc := sliceEncoder{newArrayEncoder(t)}
	return enc.encode
}

type arrayEncoder struct {
	elemEnc encoderFunc
}

func (ae arrayEncoder) encode(e *encodeState, v reflect.Value) {
	n := v.Len()
	e.WriteUint32(uint32(n))

	// fast path for []byte
	if v.Type().Elem().Kind() == reflect.Uint8 {
		b := make([]byte, n)
		reflect.Copy(reflect.ValueOf(b), v)
		e.Write(b)
		return
	}

	// loop over elements
	// we write all the elements, also the zero values
	for i := 0; i < n; i++ {
		ae.elemEnc(e, v.Index(i))
	}
}

func newArrayEncoder(t reflect.Type) encoderFunc {
	enc := arrayEncoder{typeEncoder(t.Elem())}
	return enc.encode
}

type ptrEncoder struct {
	elemEnc encoderFunc
}

func (pe ptrEncoder) encode(e *encodeState, v reflect.Value) {
	if v.IsNil() {
		return
	}
	if e.ptrLevel++; e.ptrLevel > startDetectingCyclesAfter {
		// We're a large number of nested ptrEncoder.encode calls deep;
		// start checking if we've run into a pointer cycle.
		ptr := v.Interface()
		if _, ok := e.ptrSeen[ptr]; ok {
			e.error(&UnsupportedValueError{v, fmt.Sprintf("encountered a cycle via %s", v.Type())})
		}
		e.ptrSeen[ptr] = struct{}{}
		defer delete(e.ptrSeen, ptr)
	}
	pe.elemEnc(e, v.Elem())
	e.ptrLevel--
}

func newPtrEncoder(t reflect.Type) encoderFunc {
	enc := ptrEncoder{typeEncoder(t.Elem())}
	return enc.encode
}

type condAddrEncoder struct {
	canAddrEnc, elseEnc encoderFunc
}

func (ce condAddrEncoder) encode(e *encodeState, v reflect.Value) {
	if v.CanAddr() {
		ce.canAddrEnc(e, v)
	} else {
		ce.elseEnc(e, v)
	}
}

// newCondAddrEncoder returns an encoder that checks whether its value
// CanAddr and delegates to canAddrEnc if so, else to elseEnc.
func newCondAddrEncoder(canAddrEnc, elseEnc encoderFunc) encoderFunc {
	enc := condAddrEncoder{canAddrEnc: canAddrEnc, elseEnc: elseEnc}
	return enc.encode
}

// A field represents a single field found in a struct.
type field struct {
	name      string
	nameBytes []byte // []byte(name)

	index []int
	typ   reflect.Type

	encoder encoderFunc
}

// typeFields returns a list of fields that the encoder should recognize for a given type.
// The algorithm is a depth-first search of the set of structures, including any reachable anonymous structures,
// until the structure is traversed completely.
func typeFields(t reflect.Type) structFields {
	var fields []field

	// Visit the fields of the given type.
	visitField := func(f reflect.StructField) {
		t := f.Type
		// return marshalerEncoder directly, if it implements Marshaler.
		if t.Implements(encoderType) {
			fields = append(fields, field{name: f.Name, nameBytes: []byte(f.Name), index: f.Index, typ: t, encoder: marshalerEncoder})
			return
		}

		// time.Time is special because it has embedded structs that use timeEncoder.
		if t.AssignableTo(timeType) || (t.Kind() == reflect.Pointer && t.Elem().AssignableTo(timeType)) {
			fields = append(fields, field{name: f.Name, nameBytes: []byte(f.Name), index: f.Index, typ: t, encoder: timeEncoder})
			return
		}
		if t.ConvertibleTo(timeType) {
			converted := reflect.New(t).Elem().Convert(timeType)
			if _, ok := converted.Interface().(time.Time); ok {
				fields = append(fields, field{name: f.Name, nameBytes: []byte(f.Name), index: f.Index, typ: t, encoder: timeEncoder})
				return
			}
		}

		// Check for anonymous fields (embedded structs).
		if f.Anonymous {
			if t.Kind() == reflect.Pointer {
				t = f.Type.Elem()
			}
			fields = append(fields, typeFields(t).list...)
		}

		fields = append(fields, field{name: f.Name, nameBytes: []byte(f.Name), index: f.Index, typ: f.Type, encoder: typeEncoder(f.Type)})
	}

	// Process all fields in the root struct.
	for i := 0; i < t.NumField(); i++ {
		visitField(t.Field(i))
	}
	return structFields{list: fields}
}

var fieldCache sync.Map // map[reflect.Type]structFields

// cachedTypeFields is like typeFields but uses a cache to avoid repeated work.
func cachedTypeFields(t reflect.Type) structFields {
	if f, ok := fieldCache.Load(t); ok {
		return f.(structFields)
	}
	f, _ := fieldCache.LoadOrStore(t, typeFields(t))
	return f.(structFields)
}
