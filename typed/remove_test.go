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

var nestedTypesSchema = `types:
- name: type
  map:
    fields:
      - name: listOfLists
        type:
          namedType: listOfLists
      - name: listOfMaps
        type:
          namedType: listOfMaps
      - name: mapOfLists
        type:
          namedType: mapOfLists
      - name: mapOfMaps
        type:
          namedType: mapOfMaps
      - name: mapOfMapsRecursive
        type:
          namedType: mapOfMapsRecursive
      - name: struct
        type:
          namedType: struct
- name: struct
  map:
    fields:
    - name: name
      type:
        scalar: string
    - name: value
      type:
        scalar: number
- name: listOfLists
  list:
    elementType:
      map:
        fields:
        - name: name
          type:
            scalar: string
        - name: value
          type:
            namedType: list
    elementRelationship: associative
    keys:
    - name
- name: list
  list:
    elementType:
      scalar: string
    elementRelationship: associative
- name: listOfMaps
  list:
    elementType:
      map:
        fields:
        - name: name
          type:
            scalar: string
        - name: value
          type:
            namedType: map
    elementRelationship: associative
    keys:
    - name
- name: map
  map:
    elementType:
      scalar: string
    elementRelationship: associative
- name: mapOfLists
  map:
    elementType:
      namedType: list
    elementRelationship: associative
- name: mapOfMaps
  map:
    elementType:
      namedType: map
    elementRelationship: associative
- name: mapOfMapsRecursive
  map:
    elementType:
      namedType: mapOfMapsRecursive
    elementRelationship: associative
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
	}},
}, {
	name:         "nested types",
	rootTypeName: "type",
	schema:       typed.YAMLObject(nestedTypesSchema),
	quadruplets: []removeQuadruplet{{
		// path to root type
		`{"listOfLists": [{"name": "a", "value": ["b", "c"]}, {"name": "d"}]}`,
		_NS(
			_P("listOfLists"),
		),
		``,
		`{"listOfLists": [{"name": "a", "value": ["b", "c"]}, {"name": "d"}]}`,
	}, {
		// path to a top-level element
		`{"listOfLists": [{"name": "a", "value": ["b", "c"]}, {"name": "d"}]}`,
		_NS(_P("listOfLists", _KBF("name", "d"))),
		`{"listOfLists": [{"name": "a", "value": ["b", "c"]}]}`,
		`{"listOfLists": [{"name": "d"}]}`,
	}, {
		// same as previous with the other top-level element containing nested elements.
		`{"listOfLists": [{"name": "a", "value": ["b", "c"]}, {"name": "d"}]}`,
		_NS(
			_P("listOfLists", _KBF("name", "a")),
		),
		`{"listOfLists": [{"name": "d"}]}`,
		`{"listOfLists": [{"name": "a", "value": ["b", "c"]}]}`,
	}, {
		// just one path to leaf element
		`{"listOfLists": [{"name": "a", "value": ["b", "c"]}, {"name": "d"}]}`,
		_NS(
			_P("listOfLists", _KBF("name", "a"), "value", _V("b")),
		),
		`{"listOfLists": [{"name":"a", "value": ["c"]}, {"name": "d"}]}`,
		`unparseable`, // cannot extract leaf element without path to top-level element as well
	}, {
		// paths to leaf and top level element
		`{"listOfLists": [{"name": "a", "value": ["b", "c"]}, {"name": "d"}]}`,
		_NS(
			_P("listOfLists", _KBF("name", "a"), "name"),
			_P("listOfLists", _KBF("name", "a"), "value", _V("b")),
		),
		`unparseable`, // cannot remove a top-lvel list and a single element from the list within
		`{"listOfLists": [{"name": "a", "value": ["b"]}]}`,
	}, {
		// path to non-existant top-level element
		`{"listOfLists": [{"name": "a", "value": ["b", "c"]}, {"name": "d"}]}`,
		_NS(
			_P("listOfLists", _KBF("name", "x")),
		),
		`{"listOfLists": [{"name": "a", "value": ["b", "c"]}, {"name": "d"}]}`, // doesn't remove anything
		`{"listOfLists":null}`, // extract only the root type
	}, {
		// path with existant top-level but non-existant leaf element
		`{"listOfLists": [{"name": "a", "value": ["b", "c"]}, {"name": "d"}]}`,
		_NS(
			_P("listOfLists", _KBF("name", "a"), "value", _V("x")),
		),
		`{"listOfLists": [{"name":"a", "value": ["b","c"]}, {"name": "d"}]}`, // nothing removed since the path doesn't exist.
		`unparseable`, //`{"listOfLists":[{"value":null}]}`, // unparseable because there is no name (and value cannot be null)
	}, {
		// paths with existant top-level but non-existant leaf element
		`{"listOfLists": [{"name": "a", "value": ["b", "c"]}, {"name": "d"}]}`,
		_NS(
			_P("listOfLists", _KBF("name", "a"), "name"),
			_P("listOfLists", _KBF("name", "a"), "value", _V("x")),
		),
		`unparseable`, // unparseable because remove cannot operate on a top-level element and a leaf within
		`unparseable`, //`{"listOfLists":[{"name: "a","value": null}]}`, // unparseable because value cannot be null
	}, {
		// invalid path to just a leaf
		`{"listOfLists": [{"name": "a", "value": ["b", "c"]}, {"name": "d"}]}`,
		_NS(
			_P(_V("b")),
		),
		`{"listOfLists": [{"name": "a", "value": ["b", "c"]}, {"name": "d"}]}`,
		``,
	}, {
		// path to root type
		`{"listOfMaps": [{"name": "a", "value": {"b":"x", "c":"y"}}, {"name": "d", "value": {"e":"z"}}]}`,
		_NS(
			_P("listOfMaps"),
		),
		``,
		`{"listOfMaps": [{"name": "a", "value": {"b":"x", "c":"y"}}, {"name": "d", "value": {"e":"z"}}]}`,
	}, {
		// path to a top-level element
		`{"listOfMaps": [{"name": "a", "value": {"b":"x", "c":"y"}}, {"name": "d", "value": {"e":"z"}}]}`,
		_NS(
			_P("listOfMaps", _KBF("name", "a")),
		),
		`{"listOfMaps": [{"name": "d", "value": {"e":"z"}}]}`,
		`{"listOfMaps": [{"name": "a", "value": {"b":"x", "c":"y"}}]}`,
	}, {
		// just one path to leaf element
		`{"listOfMaps": [{"name": "a", "value": {"b":"x", "c":"y"}}, {"name": "d", "value": {"e":"z"}}]}`,
		_NS(
			_P("listOfMaps", _KBF("name", "a"), "value", "b"),
		),
		`{"listOfMaps": [{"name": "a", "value": {"c":"y"}}, {"name": "d", "value": {"e":"z"}}]}`,
		`unparseable`, // cannot extract leaf element without path to top-level element as well
	}, {
		// paths to leaf and top level element
		`{"listOfMaps": [{"name": "a", "value": {"b":"x", "c":"y"}}, {"name": "d", "value": {"e":"z"}}]}`,
		_NS(
			_P("listOfMaps", _KBF("name", "a"), "name"),
			_P("listOfMaps", _KBF("name", "a"), "value", "b"),
		),
		`unparseable`, // cannot remove a top-lvel list and a single element from the list within
		`{"listOfMaps": [{"name": "a", "value": {"b":"x"}}]}`,
	}, {
		// path to non-existant top-level element
		`{"listOfMaps": [{"name": "a", "value": {"b":"x", "c":"y"}}, {"name": "d", "value": {"e":"z"}}]}`,
		_NS(
			_P("listOfMaps", _KBF("name", "q")),
		),
		`{"listOfMaps": [{"name": "a", "value": {"b":"x", "c":"y"}}, {"name": "d", "value": {"e":"z"}}]}`, // doesn't remove anything
		`{"listOfMaps":null}`, // extract only the root type
	}, {
		// path with existant top-level but non-existant leaf element.
		`{"listOfMaps": [{"name": "a", "value": {"b":"x", "c":"y"}}, {"name": "d", "value": {"e":"z"}}]}`,
		_NS(
			_P("listOfMaps", _KBF("name", "a"), "value", "q"),
		),
		`{"listOfMaps": [{"name": "a", "value": {"b":"x", "c":"y"}}, {"name": "d", "value": {"e":"z"}}]}`, // doesn't remove anything
		`unparseable`, //`{"listOfMaps": [{"value": null}]}`, // unparseable because there is no name (and value cannot be null)
	}, {
		// paths with existant top-level but non-existant leaf element
		`{"listOfMaps": [{"name": "a", "value": {"b":"x", "c":"y"}}, {"name": "d", "value": {"e":"z"}}]}`,
		_NS(
			_P("listOfMaps", _KBF("name", "a"), "name"),
			_P("listOfMaps", _KBF("name", "a"), "value", "q"),
		),
		`unparseable`, // unparseable because remove cannot operate on a top-level element and a leaf within
		`{"listOfMaps": [{"name":"a", "value": null}]}`, // IS parseable because value can be null
	}, {
		// path to root type
		//`{"mapOfLists": {"b":["a","c"]}}`,
		`{"mapOfLists": {"b":["a","c"], "d":["e", "f"]}}`,
		_NS(
			_P("mapOfLists"),
		),
		``,
		`{"mapOfLists": {"b":["a","c"], "d":["e", "f"]}}`,
	}, {
		// path to a top-level element
		`{"mapOfLists": {"b":["a","c"], "d":["e", "f"]}}`,
		_NS(
			_P("mapOfLists", "b"),
		),
		`{"mapOfLists": {"d":["e", "f"]}}`,
		`{"mapOfLists": {"b":["a","c"]}}`,
	}, {
		// just one path to leaf element
		`{"mapOfLists": {"b":["a","c"], "d":["e", "f"]}}`,
		_NS(
			_P("mapOfLists", "b", _V("a")),
		),
		`{"mapOfLists":{"b":["c"],"d":["e", "f"]}}`,
		`{"mapOfLists":{"b":["a"]}}`,
	}, {
		// path to non-existant top-level element
		`{"mapOfLists": {"b":["a","c"], "d":["e", "f"]}}`,
		_NS(
			_P("mapOfLists", "q"),
		),
		`{"mapOfLists": {"b":["a","c"], "d":["e", "f"]}}`,
		`{"mapOfLists":null}`,
	}, {
		// path with existant top-level but non-existant leaf element
		`{"mapOfLists": {"b":["a","c"], "d":["e", "f"]}}`,
		_NS(
			_P("mapOfLists", "b", _V("q")),
		),
		`{"mapOfLists": {"b":["a","c"], "d":["e", "f"]}}`,
		`{"mapOfLists":{"b":null}}`,
	}, {
		// path with existant top-level but non-existant leaf element
		`{"mapOfLists": {"b":null, "d":["e", "f"]}}`,
		_NS(
			_P("mapOfLists", "b"),
		),
		`{"mapOfLists": {"d":["e", "f"]}}`,
		`{"mapOfLists":{"b":null}}`, // same output as previous case, but can be differentiated by input fieldpath.Set
	}, {
		// invalid path to just a leaf
		`{"mapOfLists": {"b":["a","c"], "d":["e", "f"]}}`,
		_NS(
			_P(_V("a")),
		),
		`{"mapOfLists": {"b":["a","c"], "d":["e", "f"]}}`,
		``,
	}, {
		// path to root type
		`{"mapOfMaps": {"b":{"a":"x","c":"z"}, "d":{"e":"y", "f":"w"}}}`,
		_NS(
			_P("mapOfMaps"),
		),
		``,
		`{"mapOfMaps": {"b":{"a":"x","c":"z"}, "d":{"e":"y", "f":"w"}}}`,
	}, {
		// path to a top-level element
		`{"mapOfMaps": {"b":{"a":"x","c":"z"}, "d":{"e":"y", "f":"w"}}}`,
		_NS(
			_P("mapOfMaps", "b"),
		),
		`{"mapOfMaps": {"d":{"e":"y", "f":"w"}}}`,
		`{"mapOfMaps": {"b":{"a":"x","c":"z"}}}`,
	}, {
		// just one path to leaf element
		`{"mapOfMaps": {"b":{"a":"x","c":"z"}, "d":{"e":"y", "f":"w"}}}`,
		_NS(
			_P("mapOfMaps", "b", "a"),
		),
		`{"mapOfMaps": {"b":{"c":"z"},"d":{"e":"y", "f":"w"}}}`,
		`{"mapOfMaps": {"b":{"a":"x"}}}`,
	}, {
		// path to non-existant top-level element
		`{"mapOfMaps": {"b":{"a":"x","c":"z"}, "d":{"e":"y", "f":"w"}}}`,
		_NS(
			_P("mapOfMaps", "q"),
		),
		`{"mapOfMaps": {"b":{"a":"x","c":"z"}, "d":{"e":"y", "f":"w"}}}`,
		`{"mapOfMaps": null}`,
	}, {
		// path with existant top-level but non-existant leaf element
		`{"mapOfMaps": {"b":{"a":"x","c":"z"}, "d":{"e":"y", "f":"w"}}}`,
		_NS(
			_P("mapOfMaps", "b", "q"),
		),
		`{"mapOfMaps": {"b":{"a":"x","c":"z"}, "d":{"e":"y", "f":"w"}}}`,
		`{"mapOfMaps": {"b":null}}`,
	}, {
		// top-level element with null leaf elements
		`{"mapOfMaps": {"b":null, "d":{"e":"y", "f":"w"}}}`,
		_NS(
			_P("mapOfMaps", "b"),
		),
		`{"mapOfMaps": {"d":{"e":"y", "f":"w"}}}`,
		`{"mapOfMaps": {"b":null}}`, // same output as previous case, but can be differentiated by input fieldpath.Set
	}, {
		// invalid path to just a leaf
		`{"mapOfMaps": {"b":{"a":"x","c":"z"}, "d":{"e":"y", "f":"w"}}}`,
		_NS(
			_P("a"),
		),
		`{"mapOfMaps": {"b":{"a":"x","c":"z"}, "d":{"e":"y", "f":"w"}}}`,
		``,
	}, {
		// root element
		`{"mapOfMapsRecursive": {"a":{"b":{"c":null}}}}`,
		_NS(
			_P("mapOfMapsRecursive"),
		),
		``,
		`{"mapOfMapsRecursive": {"a":{"b":{"c":null}}}}`,
	}, {
		// top-level map
		`{"mapOfMapsRecursive": {"a":{"b":{"c":null}}}}`,
		_NS(
			_P("mapOfMapsRecursive", "a"),
		),
		`{"mapOfMapsRecursive"}`,
		`{"mapOfMapsRecursive": {"a":{"b":{"c":null}}}}`,
	}, {
		// second-level map
		`{"mapOfMapsRecursive": {"a":{"b":{"c":null}}}}`,
		_NS(
			_P("mapOfMapsRecursive", "a", "b"),
		),
		`{"mapOfMapsRecursive":{"a":null}}`,
		`{"mapOfMapsRecursive": {"a":{"b":{"c":null}}}}`,
	}, {
		// third-level map
		`{"mapOfMapsRecursive": {"a":{"b":{"c":null}}}}`,
		_NS(
			_P("mapOfMapsRecursive", "a", "b", "c"),
		),
		`{"mapOfMapsRecursive":{"a":{"b":null}}}`,
		`{"mapOfMapsRecursive": {"a":{"b":{"c":null}}}}`,
	}},
}}

func (tt removeTestCase) test(t *testing.T) {
	parser, err := typed.NewParser(tt.schema)
	if err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	for i, quadruplet := range tt.quadruplets {
		quadruplet := quadruplet
		tc := fmt.Sprintf("%v-valid-%v", tt.name, i)
		t.Run(tc, func(t *testing.T) {
			//fmt.Printf("tc = %+v\n", tc)
			// uncomment once done testing
			//t.Parallel()
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
			fmt.Println("test RemoveItems")
			if quadruplet.removeOutput != "unparseable" {
				rmOut, err := pt.FromYAML(quadruplet.removeOutput)
				if err != nil {
					t.Fatalf("unable to parser/validate removeOutput yaml: %v\n%v", err, quadruplet.removeOutput)
				}

				rmGot := tv.RemoveItems(set)
				if !value.Equals(rmGot.AsValue(), rmOut.AsValue()) {
					t.Errorf("RemoveItems expected\n%v\nbut got\n%v\n",
						value.ToString(rmOut.AsValue()), value.ToString(rmGot.AsValue()),
					)
				}
			}

			// test ExtractItems
			fmt.Println("---------------------")
			if quadruplet.extractOutput != "unparseable" {
				fmt.Println("test ExtractItems")
				exOut, err := pt.FromYAML(quadruplet.extractOutput)
				if err != nil {
					t.Fatalf("unable to parser/validate extractOutput yaml: %v\n%v", err, quadruplet.extractOutput)
				}
				exGot := tv.ExtractItems(set)
				//// DEBUGGING
				//setBytes, err := set.ToJSON()
				//if err != nil {
				//	t.Fatalf("setBytes err %v", err)
				//}
				//fmt.Printf("setBytes = %+v\n", string(setBytes))

				exOutFS, err := exOut.ToFieldSet()
				if err != nil {
					t.Fatalf("exOutFS err %v", err)
				}
				exOutBytes, err := exOutFS.ToJSON()
				if err != nil {
					t.Fatalf("exOutBytes err %v", err)
				}
				fmt.Printf("outBytes = %+v\n", string(exOutBytes))
				// broken because you cant go from json to yaml string
				//exOutYAML, err := k8syaml.JSONToYAML(exOutBytes)
				//if err != nil {
				//	t.Fatalf("exOutYAML err %v", err)
				//}
				//exOut2, err := pt.FromYAML(typed.YAMLObject(string(exOutYAML)))
				//if err != nil {
				//	t.Fatalf("exOut2 eer %v", err)
				//}
				//fmt.Printf("value.ToString(exOut2.AsValue() = %+v\n", value.ToString(exOut2.AsValue()))

				////
				if !value.Equals(exGot.AsValue(), exOut.AsValue()) {
					t.Errorf("ExtractItems expected\n%v\nbut got\n%v\n",
						value.ToString(exOut.AsValue()), value.ToString(exGot.AsValue()),
					)
				}

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
