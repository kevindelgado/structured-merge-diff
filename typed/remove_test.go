/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package typed_test

import (
	"fmt"
	"testing"

	"sigs.k8s.io/structured-merge-diff/v4/fieldpath"
	"sigs.k8s.io/structured-merge-diff/v4/typed"
	"sigs.k8s.io/structured-merge-diff/v4/value"
)

type removeTestCase struct {
	name         string
	rootTypeName string
	schema       typed.YAMLObject
	quadruplets  []removeQuadruplet
}

type removeQuadruplet struct {
	object        typed.YAMLObject
	set           *fieldpath.Set
	removeOutput  typed.YAMLObject
	extractOutput typed.YAMLObject
}

var simplePairSchema = `types:
- name: stringPair
  map:
    fields:
    - name: key
      type:
        scalar: string
    - name: value
      type:
        namedType: __untyped_atomic_
- name: __untyped_atomic_
  scalar: untyped
  list:
    elementType:
      namedType: __untyped_atomic_
    elementRelationship: atomic
  map:
    elementType:
      namedType: __untyped_atomic_
    elementRelationship: atomic
`

var structGrabBagSchema = `types:
- name: myStruct
  map:
    fields:
    - name: numeric
      type:
        scalar: numeric
    - name: string
      type:
        scalar: string
    - name: bool
      type:
        scalar: boolean
    - name: setStr
      type:
        list:
          elementType:
            scalar: string
          elementRelationship: associative
    - name: setBool
      type:
        list:
          elementType:
            scalar: boolean
          elementRelationship: associative
    - name: setNumeric
      type:
        list:
          elementType:
            scalar: numeric
          elementRelationship: associative
`

var associativeListSchema = `types:
- name: myRoot
  map:
    fields:
    - name: list
      type:
        namedType: myList
    - name: atomicList
      type:
        namedType: mySequence
- name: myList
  list:
    elementType:
      namedType: myElement
    elementRelationship: associative
    keys:
    - key
    - id
- name: mySequence
  list:
    elementType:
      scalar: string
    elementRelationship: atomic
- name: myElement
  map:
    fields:
    - name: key
      type:
        scalar: string
    - name: id
      type:
        scalar: numeric
    - name: value
      type:
        namedType: myValue
    - name: bv
      type:
        scalar: boolean
    - name: nv
      type:
        scalar: numeric
- name: myValue
  map:
    elementType:
      scalar: string
`

var removeCases = []removeTestCase{{
	name:         "simple pair",
	rootTypeName: "stringPair",
	schema:       typed.YAMLObject(simplePairSchema),
	quadruplets: []removeQuadruplet{{
		`{"key":"foo"}`,
		_NS(_P("key")),
		``,
		`{"key":"foo"}`,
	}, {
		`{"key":"foo"}`,
		_NS(),
		`{"key":"foo"}`,
		``,
	}, {
		`{"key":"foo","value":true}`,
		_NS(_P("key")),
		`{"value":true}`,
		`{"key":"foo"}`,
	}, {
		`{"key":"foo","value":{"a": "b"}}`,
		_NS(_P("value")),
		`{"key":"foo"}`,
		`{"value":{"a": "b"}}`,
	}},
}, {
	name:         "struct grab bag",
	rootTypeName: "myStruct",
	schema:       typed.YAMLObject(structGrabBagSchema),
	quadruplets: []removeQuadruplet{{
		`{"setBool":[false]}`,
		_NS(_P("setBool", _V(false))),
		// is this the right remove output?
		`{"setBool":null}`,
		`{"setBool":[false]}`,
	}, {
		`{"setBool":[false]}`,
		_NS(_P("setBool", _V(true))),
		`{"setBool":[false]}`,
		`{"setBool":null}`,
	}, {
		`{"setBool":[true,false]}`,
		_NS(_P("setBool", _V(true))),
		`{"setBool":[false]}`,
		`{"setBool":[true]}`,
	}, {
		`{"setBool":[true,false]}`,
		_NS(_P("setBool")),
		``,
		`{"setBool":[true,false]}`,
	}, {
		`{"setNumeric":[1,2,3,4.5]}`,
		_NS(_P("setNumeric", _V(1)), _P("setNumeric", _V(4.5))),
		`{"setNumeric":[2,3]}`,
		`{"setNumeric":[1,4.5]}`,
	}, {
		`{"setStr":["a","b","c"]}`,
		_NS(_P("setStr", _V("a"))),
		`{"setStr":["b","c"]}`,
		`{"setStr":["a"]}`,
	}},
}, {
	name:         "associative list",
	rootTypeName: "myRoot",
	schema:       typed.YAMLObject(associativeListSchema),
	quadruplets: []removeQuadruplet{{
		`{"list":[{"key":"a","id":1},{"key":"a","id":2},{"key":"b","id":1}]}`,
		_NS(_P("list", _KBF("key", "a", "id", 1))),
		`{"list":[{"key":"a","id":2},{"key":"b","id":1}]}`,
		`{"list":[{"key":"a","id":1}]}`,
	}, {
		`{"atomicList":["a", "a", "a"]}`,
		_NS(_P("atomicList")),
		``,
		`{"atomicList":["a", "a", "a"]}`,
		//}, {
		//	// questions about the rest of this list
		//	`{"list":[{"key":"a","id":1,"value":{"a":"a"}}]}`,
		//	_NS(_P("list", _KBF("key", "a", "id", "1", "value", "a"))),
		//	`{"list":[{"key":"a","id":1,"value":{"a":"a"}}]}`,
		//	`{"list":null}`,
		//}, {
		//	`{"list":[{"key":"a","id":1,"value":{"a":"a"}}]}`,
		//	_NS(_P("list", _KBF("key", "a", "id", "1"))),
		//	`{"list":null}`,
		//	`{"list":[{"key":"a","id":1,"value":{"a":"a"}}]}`,
	}},
}}

func (tt removeTestCase) test(t *testing.T) {
	parser, err := typed.NewParser(tt.schema)
	if err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	for i, quadruplet := range tt.quadruplets {
		quadruplet := quadruplet
		t.Run(fmt.Sprintf("%v-valid-%v", tt.name, i), func(t *testing.T) {
			t.Parallel()
			pt := parser.Type(tt.rootTypeName)

			tv, err := pt.FromYAML(quadruplet.object)
			if err != nil {
				t.Fatalf("unable to parser/validate object yaml: %v\n%v", err, quadruplet.object)
			}

			set := quadruplet.set
			if err != nil {
				t.Errorf("set validation errors: %v", err)
			}

			// test RemoveItems
			rmOut, err := pt.FromYAML(quadruplet.removeOutput)
			if err != nil {
				t.Fatalf("unable to parser/validate removeOutput yaml: %v\n%v", err, quadruplet.removeOutput)
			}

			rmGot := tv.RemoveItems(set)
			if !value.Equals(rmGot.AsValue(), rmOut.AsValue()) {
				t.Errorf("Expected\n%v\nbut got\n%v\n",
					value.ToString(rmOut.AsValue()), value.ToString(rmGot.AsValue()),
				)
			}

			// test ExtractItems
			exOut, err := pt.FromYAML(quadruplet.extractOutput)
			if err != nil {
				t.Fatalf("unable to parser/validate extractOutput yaml: %v\n%v", err, quadruplet.extractOutput)
			}
			exGot := tv.ExtractItems(set)
			if !value.Equals(exGot.AsValue(), exOut.AsValue()) {
				t.Errorf("Expected\n%v\nbut got\n%v\n",
					value.ToString(exOut.AsValue()), value.ToString(exGot.AsValue()),
				)
			}
		})
	}
}

func TestRemove(t *testing.T) {
	for _, tt := range removeCases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.test(t)
		})
	}
}
