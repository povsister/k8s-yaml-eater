package yamleater

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"reflect"

	"k8s.io/apimachinery/pkg/runtime"
	apischeme "k8s.io/apimachinery/pkg/runtime/schema"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
)

var (
	// ReadAhead defines the cache for reading YAML from source.
	// By default read 5 YAML documents ahead of decoding.
	// Change it before NewYamlEater.
	ReadAhead uint = 5
)

// for internal use
var (
	invalidJSON = []byte(`null`)
	emptyJSON   = []byte(`{}`)
)

type decodeResult struct {
	Obj runtime.Object
	Gvk *apischeme.GroupVersionKind
}

type yamlEater struct {
	in         io.Reader
	current    []byte
	currentObj *decodeResult

	yamlReader *k8syaml.YAMLReader
	errRead    error
	readChan   chan []byte

	yamlDecoder runtime.Decoder
	errDecode   error
}

// NewYamlEater returns a YamlEater obj with given data source.
// The read source should be one of: []byte content, io.Reader, io.ReadCloser or a string representing a file path.
func NewYamlEater(read interface{}) (*yamlEater, error) {
	in, err := newReader(read)
	if err != nil {
		return nil, err
	}

	eater := &yamlEater{in, nil, nil,
		nil, nil, make(chan []byte, ReadAhead), nil, nil}

	go eater.readYAML()

	return eater, nil
}

// Next returns the next full YAML documents in form of []byte, or an error.
// The index for "Next" is in sync with "NextObj". It returns io.EOF error if reached the end.
func (e *yamlEater) Next() ([]byte, error) {
	next, ok := <-e.readChan
	if !ok {
		e.current = nil
		return nil, e.errRead
	}
	e.current = next
	return next, nil
}

// Current returns the current full YAML documents.
// It returns an error if called before Next/NextObj or the last Next call fails.
func (e *yamlEater) Current() ([]byte, error) {
	if e.current == nil && e.errRead == nil {
		return nil, fmt.Errorf(`method Current() called before Next()`)
	}
	return e.current, e.errRead
}

// CurrentObj returns the current full decoded object.
// It returns an error if called before NextObj/Next or the last Next call fails.
func (e *yamlEater) CurrentObj() (runtime.Object, *apischeme.GroupVersionKind, error) {
	if e.currentObj == nil {
		if e.current == nil && e.errRead == nil {
			return nil, nil, fmt.Errorf(`method CurrentObj() called before NextObj()`)
		} else if e.current != nil {
			// didn't Decode the Object
			obj, gvk, err := e.yamlDecoder.Decode(e.current, nil, nil)
			e.currentObj = &decodeResult{obj, gvk}
			e.errDecode = err
		} else if e.errRead != nil {
			return nil, nil, e.errRead
		}
	}
	return e.currentObj.Obj, e.currentObj.Gvk, e.errDecode
}

// NextObj returns the next decoded object as well as the kind, group, and version from the serialized data, or an error.
// It will recognize all known typed resources registered in current API schema.
// The index for "NextObj" is in sync with "Next". It returns io.EOF error if reached the end.
func (e *yamlEater) NextObj() (runtime.Object, *apischeme.GroupVersionKind, error) {
	nextDoc, err := e.Next()
	if err != nil {
		e.currentObj = nil
		e.errDecode = err
		return nil, nil, err
	}
	obj, gvk, err := e.yamlDecoder.Decode(nextDoc, nil, nil)
	e.currentObj = &decodeResult{obj, gvk}
	e.errDecode = err
	return obj, gvk, err
}

// runs in separated goroutine
func (e *yamlEater) readYAML() {
	// init Reader and Decoder
	e.yamlReader = k8syaml.NewYAMLReader(bufio.NewReader(e.in))
	e.yamlDecoder = scheme.Codecs.UniversalDeserializer()

	for {
		read, err := e.yamlReader.Read()
		if err != nil {
			// err could be io.EOF
			e.errRead = err
			// if in is an io.ReadCloser, eg: fileDescriptor. Close it on err or io.EOF
			if readCloser, ok := e.in.(io.ReadCloser); ok {
				_ = readCloser.Close()
			}
			close(e.readChan)
			return
		}
		// validate the YAML by converting it to JSON
		jsonTest, err := k8syaml.ToJSON(read)
		if !bytes.Equal(jsonTest, invalidJSON) && !bytes.Equal(jsonTest, emptyJSON) {
			e.readChan <- read
		}
	}
}

func newReader(read interface{}) (io.Reader, error) {
	if read == nil {
		return nil, fmt.Errorf(`read can not be nil`)
	}
	v := reflect.ValueOf(read)

	switch v.Kind() {
	case reflect.Slice:
		elmV := v.Elem()
		// expect byte slice
		if elmV.Kind() != reflect.Uint8 {
			return nil, fmt.Errorf(`unexpect %s slice, only []byte allowed`, v.Type().String())
		}
		return bytes.NewReader(read.([]byte)), nil

	case reflect.String:
		// treat it as a file
		file, err := os.Open(read.(string))
		if err != nil {
			return nil, err
		}
		return file, nil

	case reflect.Ptr, reflect.Struct:
		if reader, ok := read.(io.Reader); !ok {
			return nil, fmt.Errorf(`read does not implement io.Reader`)
		} else {
			return reader, nil
		}

	default:
		return nil, fmt.Errorf(`can not use read as input, read must be []byte content, io.Reader, io.ReadCloser or a string representing a file path`)
	}
}
