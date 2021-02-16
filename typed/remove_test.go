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
	triplets     []removeTriplet
}

type removeTriplet struct {
	object typed.YAMLObject
	set    *fieldpath.Set
	out    typed.YAMLObject
	exOut  typed.YAMLObject
}

//var (
//	// Short names for readable test cases.
//	_NS  = fieldpath.NewSet
//	_P   = fieldpath.MakePathOrDie
//	_KBF = fieldpath.KeyByFields
//	_V   = value.NewValueInterface
//)

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
var nestedMapSchema = `types:
- name: nestedMap
  map:
    fields:
    - name: inner
      type:
        map:
          elementType:
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

var nestedStructSchema = `types:
- name: nestedStruct
  map:
    fields:
    - name: inner
      type:
        map:
          fields:
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

var nestedListSchema = `types:
- name: nestedList
  map:
    fields:
    - name: inner
      type:
        list:
          elementType:
            namedType: __untyped_atomic_
          elementRelationship: atomic
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
	triplets: []removeTriplet{{
		//`{"key":"foo","value":{}}`,
		//`{"key":"foo","value":1}`,
		//`{"key":"foo","value":1}`,
		//}, {
		//	`{"key":"foo","value":{}}`,
		//	`{"key":"foo","value":1}`,
		//	`{"key":"foo","value":1}`,
		//}, {
		//	`{"key":"foo","value":1}`,
		//	`{"key":"foo","value":{}}`,
		//	`{"key":"foo","value":{}}`,
		//}, {
		//	`{"key":"foo","value":null}`,
		//	`{"key":"foo","value":{}}`,
		//	`{"key":"foo","value":{}}`,
		//}, {
		`{"key":"foo","value":true}`,
		//`{"key":"foo"}`,
		_NS(_P("key")),
		`{"value":true}`,
		`{"key":"foo"}`,
		//},{
		//	`{"key":"foo", "value": {"a": "b"}}`,

	}},
}, {
	name:         "associative list",
	rootTypeName: "myRoot",
	schema:       typed.YAMLObject(associativeListSchema),
	triplets: []removeTriplet{{
		`{"list":[{"key":"a","id":1},{"key":"b","id":2},{"key":"c","id":3}]}`,
		_NS(_P("list", _KBF("key", "a", "id", 1))),
		`{"list":[{"key":"b","id":2},{"key":"c","id":3}]}`,
		`{"list":[{"key":"a","id":1}]}`,
	}},
	//}
	//}, {
	//name:         "struct grab bag",
	//rootTypeName: "myStruct",
	//schema:       typed.YAMLObject(structGrabBagSchema),
	//triplets: []removeTriplet{{
	//	//	`{"numeric":1}`,
	//	//	`{"numeric":3.14159}`,
	//	//	`{"numeric":3.14159}`,
	//	//}, {
	//	//	`{"numeric":3.14159}`,
	//	//	`{"numeric":1}`,
	//	//	`{"numeric":1}`,
	//	//}, {
	//	`{"string":"aoeu"}`,
	//	`{"bool":true}`,
	//	`{"string":"aoeu","bool":true}`,
	//	//}, {
	//	//	`{"setStr":["a","b","c"]}`,
	//	//	`{"setStr":["a","b"]}`,
	//	//	`{"setStr":["a","b","c"]}`,
	//	//}, {
	//	//	`{"setStr":["a","b"]}`,
	//	//	`{"setStr":["a","b","c"]}`,
	//	//	`{"setStr":["a","b","c"]}`,
	//	//}, {
	//	//	`{"setStr":["a","b","c"]}`,
	//	//	`{"setStr":[]}`,
	//	//	`{"setStr":["a","b","c"]}`,
	//	//}, {
	//	//	`{"setStr":[]}`,
	//	//	`{"setStr":["a","b","c"]}`,
	//	//	`{"setStr":["a","b","c"]}`,
	//	//}, {
	//	//	`{"setBool":[true]}`,
	//	//	`{"setBool":[false]}`,
	//	//	`{"setBool":[true,false]}`,
	//	//}, {
	//	//	`{"setNumeric":[1,2,3.14159]}`,
	//	//	`{"setNumeric":[1,2,3]}`,
	//	//	// KNOWN BUG: this order is wrong
	//	//	`{"setNumeric":[1,2,3.14159,3]}`,
	//}},
}}

func (tt removeTestCase) test(t *testing.T) {
	parser, err := typed.NewParser(tt.schema)
	if err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	for i, triplet := range tt.triplets {
		triplet := triplet
		t.Run(fmt.Sprintf("%v-valid-%v", tt.name, i), func(t *testing.T) {
			t.Parallel()
			pt := parser.Type(tt.rootTypeName)

			tv, err := pt.FromYAML(triplet.object)
			if err != nil {
				t.Fatalf("unable to parser/validate lhs yaml: %v\n%v", err, triplet.object)
			}
			fmt.Printf("tv:\n %v\n", value.ToString(tv.AsValue()))

			// remove
			fmt.Printf("STARTING REMOVE TEST: %v-valid-%v\n", tt.name, i)
			out, err := pt.FromYAML(triplet.out)
			if err != nil {
				t.Fatalf("unable to parser/validate out yaml: %v\n%v", err, triplet.out)
			}
			fmt.Printf("out:\n %v\n", value.ToString(out.AsValue()))

			set := triplet.set
			if err != nil {
				t.Errorf("set validation errors: %v", err)
			}
			got := tv.RemoveItems(set)
			fmt.Printf("got %v\n", value.ToString(got.AsValue()))
			if !value.Equals(got.AsValue(), out.AsValue()) {
				t.Errorf("Expected\n%v\nbut got\n%v\n",
					value.ToString(out.AsValue()), value.ToString(got.AsValue()),
				)
			}

			// extract
			fmt.Printf("STARTING EXTRACT TEST: %v-valid-%v\n", tt.name, i)
			exOut, err := pt.FromYAML(triplet.exOut)
			if err != nil {
				t.Fatalf("unable to parser/validate out yaml: %v\n%v", err, triplet.exOut)
			}
			//fmt.Printf("exOut:\n %v\n", value.ToString(exOut.AsValue()))
			exGot := tv.ExtractItems(set)
			fmt.Printf("exGot %v\n", value.ToString(exGot.AsValue()))
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
