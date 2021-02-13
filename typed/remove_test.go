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
	lhs typed.YAMLObject
	rhs typed.YAMLObject
	out typed.YAMLObject
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

var structGrabSchema = `types:
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

var removeCases = []removeTestCase{{
	//	name:         "simple pair",
	//	rootTypeName: "stringPair",
	//	schema:       typed.YAMLObject(simplePairSchema),
	//	triplets:     []removeTriplet{{
	//		//`{"key":"foo","value":{}}`,
	//		//`{"key":"foo","value":1}`,
	//		//`{"key":"foo","value":1}`,
	//		//}, {
	//		//	`{"key":"foo","value":{}}`,
	//		//	`{"key":"foo","value":1}`,
	//		//	`{"key":"foo","value":1}`,
	//		//}, {
	//		//	`{"key":"foo","value":1}`,
	//		//	`{"key":"foo","value":{}}`,
	//		//	`{"key":"foo","value":{}}`,
	//		//}, {
	//		//	`{"key":"foo","value":null}`,
	//		//	`{"key":"foo","value":{}}`,
	//		//	`{"key":"foo","value":{}}`,
	//		//}, {
	//		//	`{"key":"foo"}`,
	//		//	`{"value":true}`,
	//		//	`{"key":"foo","value":true}`,
	//	}},
	//}, {
	name:         "struct grab bag",
	rootTypeName: "myStruct",
	schema:       typed.YAMLObject(structGrabSchema),
	triplets: []removeTriplet{{
		//	`{"numeric":1}`,
		//	`{"numeric":3.14159}`,
		//	`{"numeric":3.14159}`,
		//}, {
		//	`{"numeric":3.14159}`,
		//	`{"numeric":1}`,
		//	`{"numeric":1}`,
		//}, {
		`{"string":"aoeu"}`,
		`{"bool":true}`,
		`{"string":"aoeu","bool":true}`,
		//}, {
		//	`{"setStr":["a","b","c"]}`,
		//	`{"setStr":["a","b"]}`,
		//	`{"setStr":["a","b","c"]}`,
		//}, {
		//	`{"setStr":["a","b"]}`,
		//	`{"setStr":["a","b","c"]}`,
		//	`{"setStr":["a","b","c"]}`,
		//}, {
		//	`{"setStr":["a","b","c"]}`,
		//	`{"setStr":[]}`,
		//	`{"setStr":["a","b","c"]}`,
		//}, {
		//	`{"setStr":[]}`,
		//	`{"setStr":["a","b","c"]}`,
		//	`{"setStr":["a","b","c"]}`,
		//}, {
		//	`{"setBool":[true]}`,
		//	`{"setBool":[false]}`,
		//	`{"setBool":[true,false]}`,
		//}, {
		//	`{"setNumeric":[1,2,3.14159]}`,
		//	`{"setNumeric":[1,2,3]}`,
		//	// KNOWN BUG: this order is wrong
		//	`{"setNumeric":[1,2,3.14159,3]}`,
	}},
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

			lhs, err := pt.FromYAML(triplet.lhs)
			if err != nil {
				t.Fatalf("unable to parser/validate lhs yaml: %v\n%v", err, triplet.lhs)
			}

			rhs, err := pt.FromYAML(triplet.rhs)
			if err != nil {
				t.Fatalf("unable to parser/validate rhs yaml: %v\n%v", err, triplet.rhs)
			}

			out, err := pt.FromYAML(triplet.out)
			if err != nil {
				t.Fatalf("unable to parser/validate out yaml: %v\n%v", err, triplet.out)
			}

			got, err := lhs.Merge(rhs)
			if err != nil {
				t.Errorf("got validation errors: %v", err)
			} else {
				if !value.Equals(got.AsValue(), out.AsValue()) {
					t.Errorf("Expected\n%v\nbut got\n%v\n",
						value.ToString(out.AsValue()), value.ToString(got.AsValue()),
					)
				}
			}

			set, err := lhs.ToFieldSet()
			if err != nil {
				t.Errorf("set validation errors: %v", err)
			}
			fmt.Printf("got %v\n", value.ToString(got.AsValue()))
			rmGot := got.RemoveItems(set)
			fmt.Printf("rmGot %v\n", value.ToString(rmGot.AsValue()))
			if !value.Equals(rmGot.AsValue(), rhs.AsValue()) {
				t.Errorf("Expected\n%v\nbut got\n%v\n",
					value.ToString(rhs.AsValue()), value.ToString(rmGot.AsValue()),
				)
			}
			exGot := got.ExtractItems(set)
			fmt.Printf("exGot %v\n", value.ToString(exGot.AsValue()))
			if !value.Equals(exGot.AsValue(), lhs.AsValue()) {
				t.Errorf("Expected\n%v\nbut got\n%v\n",
					value.ToString(lhs.AsValue()), value.ToString(exGot.AsValue()),
				)
			}
			fmt.Printf("got2 %v\n", value.ToString(got.AsValue()))
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
