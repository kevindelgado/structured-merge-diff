/*
Copyright 2019 The Kubernetes Authors.

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

package merge_test

import (
	"testing"

	"sigs.k8s.io/structured-merge-diff/v4/fieldpath"
	. "sigs.k8s.io/structured-merge-diff/v4/internal/fixture"
)

func TestExtractApplyAssociativeList(t *testing.T) {
	tests := map[string]TestCase{
		// extract applying a subset of fields does NOT
		// remove fields that the manager previously applied
		// because the set to extract contains the fields previously
		// applied
		"extract_apply_doesnt_remove_one": {
			Ops: []Operation{
				Apply{
					Manager:    "apply-one",
					APIVersion: "v1",
					Object: `
						list:
						- name: a
						- name: b
					`,
				},
				Apply{
					Manager:    "apply-two",
					APIVersion: "v2",
					Object: `
						list:
						- name: c
					`,
				},
				ExtractApply{
					Apply{
						Manager:    "apply-one",
						APIVersion: "v3",
						Object: `
						list:
						- name: a
					`,
					},
				},
			},
			Object: `
				list:
				- name: a
				- name: b
				- name: c
			`,
			APIVersion: "v1",
			Managed: fieldpath.ManagedFields{
				"apply-one": fieldpath.NewVersionedSet(
					_NS(
						_P("list", _KBF("name", "a")),
						_P("list", _KBF("name", "a"), "name"),
						_P("list", _KBF("name", "b")),
						_P("list", _KBF("name", "b"), "name"),
					),
					"v3",
					false,
				),
				"apply-two": fieldpath.NewVersionedSet(
					_NS(
						_P("list", _KBF("name", "c")),
						_P("list", _KBF("name", "c"), "name"),
					),
					"v2",
					false,
				),
			},
		},
		// extract applying an empty list just retains whatever
		// object already exists and whatever managed fields were
		// already owned.
		"extract_apply_empty_list": {
			Ops: []Operation{
				Apply{
					Manager:    "apply-one",
					APIVersion: "v1",
					Object: `
						list:
						- name: a
						- name: b
						- name: c
					`,
				},
				Apply{
					Manager:    "apply-two",
					APIVersion: "v2",
					Object: `
						list:
						- name: c
						- name: d
					`,
				},
				ExtractApply{
					Apply{
						Manager:    "apply-one",
						APIVersion: "v3",
						Object: `
						list:
					`,
					},
				},
			},
			Object: `
				list:
				- name: a
				- name: b
				- name: c
				- name: d
			`,
			APIVersion: "v3",
			Managed: fieldpath.ManagedFields{
				"apply-one": fieldpath.NewVersionedSet(
					_NS(
						_P("list", _KBF("name", "a")),
						_P("list", _KBF("name", "a"), "name"),
						_P("list", _KBF("name", "b")),
						_P("list", _KBF("name", "b"), "name"),
						_P("list", _KBF("name", "c")),
						_P("list", _KBF("name", "c"), "name"),
					),
					"v3",
					false,
				),
				"apply-two": fieldpath.NewVersionedSet(
					_NS(
						_P("list", _KBF("name", "c")),
						_P("list", _KBF("name", "d")),
						_P("list", _KBF("name", "c"), "name"),
						_P("list", _KBF("name", "d"), "name"),
					),
					"v2",
					false,
				),
			},
		},
		// after an update removes a field previously applied by apply-one,
		// apply-one only owns fields that continue to exist, it can add back
		// a field (name: a), and will own the path to it, but will not
		// re-add any fields that are not in the apply object.
		"extract_apply_missing_field": {
			Ops: []Operation{
				Apply{
					Manager:    "apply-one",
					APIVersion: "v1",
					Object: `
						list:
						- name: a
						- name: b
						- name: c
					`,
				},
				Update{
					Manager:    "controller",
					APIVersion: "v2",
					Object: `
						list:
						- name: c
						- name: d
					`,
				},
				ExtractApply{
					Apply{
						Manager:    "apply-one",
						APIVersion: "v3",
						Object: `
						list:
						- name: a
						- name: c
					`,
					},
				},
			},
			Object: `
				list:
				- name: a
				- name: c
				- name: d
			`,
			APIVersion: "v1",
			Managed: fieldpath.ManagedFields{
				"apply-one": fieldpath.NewVersionedSet(
					_NS(
						_P("list", _KBF("name", "a")),
						_P("list", _KBF("name", "a"), "name"),
						_P("list", _KBF("name", "c")),
						_P("list", _KBF("name", "c"), "name"),
					),
					"v3",
					true,
				),
				"controller": fieldpath.NewVersionedSet(
					_NS(
						_P("list", _KBF("name", "d")),
						_P("list", _KBF("name", "d"), "name"),
					),
					"v2",
					false,
				),
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if err := test.Test(associativeListParser); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestExtractApplyNestedType(t *testing.T) {
	tests := map[string]TestCase{
		// apply-two adds a value (d) to the list with name:b
		// when apply-one extract applies the list with the
		// value it had previously owned not in the apply object
		// (value c), the apply extract does not remove it because
		// the apply object is merged with the extracted value based
		// on what the manager previously owned (from the first apply-one).
		// Nothing is deleted.
		"extract_doesnt_remove_field": {
			Ops: []Operation{
				Apply{
					Manager: "apply-one",
					Object: `
						listOfLists:
						- name: a
						- name: b
						  value:
						  - c
					`,
					APIVersion: "v1",
				},
				Apply{
					Manager: "apply-two",
					Object: `
						listOfLists:
						- name: b
						  value:
						  - d
					`,
					APIVersion: "v2",
				},
				ExtractApply{
					Apply{
						Manager: "apply-one",
						Object: `
						listOfLists:
						- name: a
					`,
						APIVersion: "v3",
					},
				},
			},
			Object: `
				listOfLists:
				- name: a
				- name: b
				  value:
				  - c
				  - d
			`,
			APIVersion: "v1",
			Managed: fieldpath.ManagedFields{
				"apply-one": fieldpath.NewVersionedSet(
					_NS(
						_P("listOfLists", _KBF("name", "a")),
						_P("listOfLists", _KBF("name", "b")),
						_P("listOfLists", _KBF("name", "b"), "name"),
						_P("listOfLists", _KBF("name", "b"), "value", _V("c")),
						_P("listOfLists", _KBF("name", "a"), "name"),
					),
					"v3",
					false,
				),
				"apply-two": fieldpath.NewVersionedSet(
					_NS(
						_P("listOfLists", _KBF("name", "b")),
						_P("listOfLists", _KBF("name", "b"), "name"),
						_P("listOfLists", _KBF("name", "b"), "value", _V("d")),
					),
					"v2",
					false,
				),
			},
		},
		// similar to the last one, but apply-two removes value before apply-one
		// does an extract apply. Because apply-one never owned that field,
		// it doesn't do anything with it.
		"extract_ignores_removed_field": {
			Ops: []Operation{
				Apply{
					Manager: "apply-one",
					Object: `
						listOfLists:
						- name: a
						- name: b
						  value:
						  - c
					`,
					APIVersion: "v1",
				},
				Apply{
					Manager: "apply-two",
					Object: `
						listOfLists:
						- name: b
						  value:
						  - d
					`,
					APIVersion: "v2",
				},
				Apply{
					Manager: "apply-two",
					Object: `
						listOfLists:
						- name: b
					`,
					APIVersion: "v2",
				},
				ExtractApply{
					Apply{
						Manager: "apply-one",
						Object: `
						listOfLists:
						- name: a
					`,
						APIVersion: "v3",
					},
				},
			},
			Object: `
				listOfLists:
				- name: a
				- name: b
				  value:
				  - c
			`,
			APIVersion: "v1",
			Managed: fieldpath.ManagedFields{
				"apply-one": fieldpath.NewVersionedSet(
					_NS(
						_P("listOfLists", _KBF("name", "a")),
						_P("listOfLists", _KBF("name", "b")),
						_P("listOfLists", _KBF("name", "b"), "name"),
						_P("listOfLists", _KBF("name", "b"), "value", _V("c")),
						_P("listOfLists", _KBF("name", "a"), "name"),
					),
					"v3",
					false,
				),
				"apply-two": fieldpath.NewVersionedSet(
					_NS(
						_P("listOfLists", _KBF("name", "b")),
						_P("listOfLists", _KBF("name", "b"), "name"),
					),
					"v2",
					false,
				),
			},
		},
		// TODO: I thought apply-two would have removed [name=a].value[=e]
		// but it did not, why?
		// What actually is happening is that the apply-two force apply
		// on [name=a] doesn't remove any of the values,
		// [name=a].value[=e] still exists after the apply-two ForceApply
		// and after the apply-one ExtractApply
		"extract_other_mgr_removes_field": {
			Ops: []Operation{
				Apply{
					Manager: "apply-one",
					Object: `
						listOfLists:
						- name: a
						  value:
						  - e
						- name: b
						  value:
						  - c
					`,
					APIVersion: "v1",
				},
				Apply{
					Manager: "apply-two",
					Object: `
						listOfLists:
						- name: b
						  value:
						  - d
					`,
					APIVersion: "v2",
				},
				ForceApply{
					Manager: "apply-two",
					Object: `
						listOfLists:
						- name: a
						- name: b
						  value:
						  - d
					`,
					APIVersion: "v2",
				},
				ExtractApply{
					Apply{
						Manager: "apply-one",
						Object: `
						listOfLists:
						- name: a
						  value:
						  - f
					`,
						APIVersion: "v3",
					},
				},
			},
			Object: `
				listOfLists:
				- name: a
				  value:
				  - e
				  - f
				- name: b
				  value:
				  - c
				  - d
			`,
			APIVersion: "v1",
			Managed: fieldpath.ManagedFields{
				"apply-one": fieldpath.NewVersionedSet(
					_NS(
						_P("listOfLists", _KBF("name", "a")),
						_P("listOfLists", _KBF("name", "b")),
						_P("listOfLists", _KBF("name", "b"), "name"),
						_P("listOfLists", _KBF("name", "b"), "value", _V("c")),
						_P("listOfLists", _KBF("name", "a"), "name"),
						_P("listOfLists", _KBF("name", "a"), "value", _V("e")),
						_P("listOfLists", _KBF("name", "a"), "value", _V("f")),
					),
					"v3",
					false,
				),
				"apply-two": fieldpath.NewVersionedSet(
					_NS(
						_P("listOfLists", _KBF("name", "a")),
						_P("listOfLists", _KBF("name", "a"), "name"),
						_P("listOfLists", _KBF("name", "b")),
						_P("listOfLists", _KBF("name", "b"), "name"),
						_P("listOfLists", _KBF("name", "b"), "value", _V("d")),
					),
					"v2",
					false,
				),
			},
		},
		// when an update overwrites a field, an extract apply
		// without that field retains ownership of what exists
		// in the ownership of that field but does not restore it.
		// i.e. apply-one continues to manage name=b and name=b.name
		// but does not restore name=b.value=c
		"extract_retains_ownership_but_doesnt_restore_field": {
			Ops: []Operation{
				Apply{
					Manager: "apply-one",
					Object: `
						listOfLists:
						- name: a
						- name: b
						  value:
						  - c
					`,
					APIVersion: "v1",
				},
				Update{
					Manager: "controller",
					Object: `
						listOfLists:
						- name: b
						  value:
						  - d
					`,
					APIVersion: "v2",
				},
				ExtractApply{
					Apply{
						Manager: "apply-one",
						Object: `
						listOfLists:
						- name: a
					`,
						APIVersion: "v3",
					},
				},
			},
			Object: `
				listOfLists:
				- name: a
				- name: b
				  value:
				  - d
			`,
			APIVersion: "v1",
			Managed: fieldpath.ManagedFields{
				"apply-one": fieldpath.NewVersionedSet(
					_NS(
						_P("listOfLists", _KBF("name", "a")),
						_P("listOfLists", _KBF("name", "a"), "name"),
						_P("listOfLists", _KBF("name", "b")),
						_P("listOfLists", _KBF("name", "b"), "name"),
					),
					"v3",
					false,
				),
				"controller": fieldpath.NewVersionedSet(
					_NS(
						_P("listOfLists", _KBF("name", "b"), "value", _V("d")),
					),
					"v2",
					false,
				),
			},
		},
		// this just shows that values written by an Updater do not get overwritten by ExtractApply
		"extract_doesnt_remove_with_dangling_subitem": {
			Ops: []Operation{
				Apply{
					Manager: "apply-one",
					Object: `
						listOfLists:
						- name: a
						- name: b
						  value:
						  - c
					`,
					APIVersion: "v1",
				},
				Apply{
					Manager: "apply-two",
					Object: `
						listOfLists:
						- name: b
						  value:
						  - d
					`,
					APIVersion: "v2",
				},
				Update{
					Manager: "controller",
					Object: `
						listOfLists:
						- name: a
						- name: b
						  value:
						  - c
						  - d
						  - e
					`,
					APIVersion: "v2",
				},
				ExtractApply{
					Apply{
						Manager: "apply-one",
						Object: `
						listOfLists:
						- name: a
					`,
						APIVersion: "v3",
					},
				},
			},
			Object: `
				listOfLists:
				- name: a
				- name: b
				  value:
				  - c
				  - d
				  - e
			`,
			APIVersion: "v1",
			Managed: fieldpath.ManagedFields{
				"apply-one": fieldpath.NewVersionedSet(
					_NS(
						_P("listOfLists", _KBF("name", "a")),
						_P("listOfLists", _KBF("name", "a"), "name"),
						_P("listOfLists", _KBF("name", "b")),
						_P("listOfLists", _KBF("name", "b"), "name"),
						_P("listOfLists", _KBF("name", "b"), "value", _V("c")),
					),
					"v3",
					false,
				),
				"apply-two": fieldpath.NewVersionedSet(
					_NS(
						_P("listOfLists", _KBF("name", "b")),
						_P("listOfLists", _KBF("name", "b"), "name"),
						_P("listOfLists", _KBF("name", "b"), "value", _V("d")),
					),
					"v2",
					false,
				),
				"controller": fieldpath.NewVersionedSet(
					_NS(
						_P("listOfLists", _KBF("name", "b"), "value", _V("e")),
					),
					"v2",
					false,
				),
			},
		},
		// this shows that apply-one only manages it's initial object
		// and none of the more recently added fields (name[a]value[b] or name[b]value[d])
		"adding_extra_subitems_remain_but_not_managed_by_extract_apply": {
			Ops: []Operation{
				Apply{
					Manager: "apply-one",
					Object: `
						listOfLists:
						- name: a
						- name: b
						  value:
						  - c
					`,
					APIVersion: "v1",
				},
				Apply{
					Manager: "apply-two",
					Object: `
						listOfLists:
						- name: a
						  value:
						  - b
					`,
					APIVersion: "v2",
				},
				Update{
					Manager: "controller",
					Object: `
						listOfLists:
						- name: a
						  value:
						  - b
						- name: b
						  value:
						  - c
						  - d
					`,
					APIVersion: "v2",
				},
				ExtractApply{
					Apply{
						Manager: "apply-one",
						Object: `
						listOfLists:
						- name: a
					`,
						APIVersion: "v3",
					},
				},
			},
			Object: `
				listOfLists:
				- name: a
				  value:
				  - b
				- name: b
				  value:
				  - c
				  - d
			`,
			APIVersion: "v1",
			Managed: fieldpath.ManagedFields{
				"apply-one": fieldpath.NewVersionedSet(
					_NS(
						_P("listOfLists", _KBF("name", "a")),
						_P("listOfLists", _KBF("name", "a"), "name"),
						_P("listOfLists", _KBF("name", "b")),
						_P("listOfLists", _KBF("name", "b"), "name"),
						_P("listOfLists", _KBF("name", "b"), "value", _V("c")),
					),
					"v3",
					true,
				),
				"apply-two": fieldpath.NewVersionedSet(
					_NS(
						_P("listOfLists", _KBF("name", "a")),
						_P("listOfLists", _KBF("name", "a"), "name"),
						_P("listOfLists", _KBF("name", "a"), "value", _V("b")),
					),
					"v2",
					false,
				),
				"controller": fieldpath.NewVersionedSet(
					_NS(
						_P("listOfLists", _KBF("name", "b"), "value", _V("d")),
					),
					"v2",
					false,
				),
			},
		},
		// When an update removes an item applied by apply-one,
		// it is removed from apply-one's managed field set, so
		// when apply-one does an extract apply
		// that item is not re-added if it is not in apply-one's
		// apply object.
		"update_removes_field_not_readded_by_apply": {
			Ops: []Operation{
				Apply{
					Manager: "apply-one",
					Object: `
						listOfLists:
						- name: a
						- name: b
						  value:
						  - c
					`,
					APIVersion: "v1",
				},
				Apply{
					Manager: "apply-two",
					Object: `
						listOfLists:
						- name: a
						  value:
						  - b
					`,
					APIVersion: "v2",
				},
				Update{
					Manager: "controller",
					Object: `
						listOfLists:
						- name: a
						  value:
						  - b
						- name: b
						  value:
						  - d
					`,
					APIVersion: "v2",
				},
				ExtractApply{
					Apply{
						Manager: "apply-one",
						Object: `
						listOfLists:
						- name: a
					`,
						APIVersion: "v3",
					},
				},
			},
			Object: `
				listOfLists:
				- name: a
				  value:
				  - b
				- name: b
				  value:
				  - d
			`,
			APIVersion: "v1",
			Managed: fieldpath.ManagedFields{
				"apply-one": fieldpath.NewVersionedSet(
					_NS(
						_P("listOfLists", _KBF("name", "a")),
						_P("listOfLists", _KBF("name", "a"), "name"),
						_P("listOfLists", _KBF("name", "b")),
						_P("listOfLists", _KBF("name", "b"), "name"),
					),
					"v3",
					true,
				),
				"apply-two": fieldpath.NewVersionedSet(
					_NS(
						_P("listOfLists", _KBF("name", "a")),
						_P("listOfLists", _KBF("name", "a"), "name"),
						_P("listOfLists", _KBF("name", "a"), "value", _V("b")),
					),
					"v2",
					false,
				),
				"controller": fieldpath.NewVersionedSet(
					_NS(
						_P("listOfLists", _KBF("name", "b"), "value", _V("d")),
					),
					"v2",
					false,
				),
			},
		},
		// this tests that for recursive maps, doing an extract apply on some
		// top-level path will extract any existing subfields and merge it
		// with the apply object.
		//
		// In this case, apply-one has the existing set
		// .mapsOfMapsRecursive.a.b
		// .mapsOfMapsRecursive.c.d
		//
		// This extracts everything .mOMR.a.b and .mOMR.c.d from the existing
		// object (which is everything) and tries to merge a subset of that
		// from the apply object (which is just .mOMR), because this is a subset
		// of the existing object, the object is not changed at all, but the
		// resulting managed field set of apply-one is every node in the object.
		"multiple_appliers_recursive_map_extract_manages_existing_object": {
			Ops: []Operation{
				Apply{
					Manager: "apply-one",
					Object: `
						mapOfMapsRecursive:
						  a:
						    b:
						  c:
						    d:
					`,
					APIVersion: "v1",
				},
				Apply{
					Manager: "apply-two",
					Object: `
						mapOfMapsRecursive:
						  a:
						  c:
						    d:
					`,
					APIVersion: "v2",
				},
				Update{
					Manager: "controller-one",
					Object: `
						mapOfMapsRecursive:
						  a:
						    b:
						      c:
						  c:
						    d:
						      e:
					`,
					APIVersion: "v3",
				},
				Update{
					Manager: "controller-two",
					Object: `
						mapOfMapsRecursive:
						  a:
						    b:
						      c:
						        d:
						  c:
						    d:
						      e:
						        f:
					`,
					APIVersion: "v2",
				},
				Update{
					Manager: "controller-one",
					Object: `
						mapOfMapsRecursive:
						  a:
						    b:
						      c:
						        d:
						          e:
						  c:
						    d:
						      e:
						        f:
						          g:
					`,
					APIVersion: "v3",
				},
				ExtractApply{
					Apply{
						Manager: "apply-one",
						Object: `
						mapOfMapsRecursive:
					`,
						APIVersion: "v4",
					},
				},
			},
			Object: `
				mapOfMapsRecursive:
				  a:
				    b:
				      c:
				        d:
				          e:
				  c:
				    d:
				      e:
				        f:
				          g:
			`,
			APIVersion: "v1",
			Managed: fieldpath.ManagedFields{
				"apply-one": fieldpath.NewVersionedSet(
					_NS(
						_P("mapOfMapsRecursive", "a"),
						_P("mapOfMapsRecursive", "a", "b"),
						_P("mapOfMapsRecursive", "a", "b", "c"),
						_P("mapOfMapsRecursive", "a", "b", "c", "d"),
						_P("mapOfMapsRecursive", "a", "b", "c", "d", "e"),
						_P("mapOfMapsRecursive", "c"),
						_P("mapOfMapsRecursive", "c", "d"),
						_P("mapOfMapsRecursive", "c", "d", "e"),
						_P("mapOfMapsRecursive", "c", "d", "e", "f"),
						_P("mapOfMapsRecursive", "c", "d", "e", "f", "g"),
					),
					"v4",
					false,
				),
				"apply-two": fieldpath.NewVersionedSet(
					_NS(
						_P("mapOfMapsRecursive", "a"),
						_P("mapOfMapsRecursive", "c"),
						_P("mapOfMapsRecursive", "c", "d"),
					),
					"v2",
					false,
				),
				"controller-one": fieldpath.NewVersionedSet(
					_NS(
						_P("mapOfMapsRecursive", "a", "b", "c"),
						_P("mapOfMapsRecursive", "a", "b", "c", "d", "e"),
						_P("mapOfMapsRecursive", "c", "d", "e"),
						_P("mapOfMapsRecursive", "c", "d", "e", "f", "g"),
					),
					"v3",
					false,
				),
				"controller-two": fieldpath.NewVersionedSet(
					_NS(
						_P("mapOfMapsRecursive", "a", "b", "c", "d"),
						_P("mapOfMapsRecursive", "c", "d", "e", "f"),
					),
					"v2",
					false,
				),
			},
		},
		// this does the same thing as the previous case, but demonstrates
		// that adding a new path will add this to the resultant object
		// (and add it to the field set of the manager as well).
		"multiple_appliers_recursive_map_extract_apply_new_field": {
			Ops: []Operation{
				Apply{
					Manager: "apply-one",
					Object: `
						mapOfMapsRecursive:
						  a:
						    b:
						  c:
						    d:
					`,
					APIVersion: "v1",
				},
				Apply{
					Manager: "apply-two",
					Object: `
						mapOfMapsRecursive:
						  a:
						  c:
						    d:
					`,
					APIVersion: "v2",
				},
				Update{
					Manager: "controller-one",
					Object: `
						mapOfMapsRecursive:
						  a:
						    b:
						      c:
						  c:
						    d:
						      e:
					`,
					APIVersion: "v3",
				},
				Update{
					Manager: "controller-two",
					Object: `
						mapOfMapsRecursive:
						  a:
						    b:
						      c:
						        d:
						  c:
						    d:
						      e:
						        f:
					`,
					APIVersion: "v2",
				},
				Update{
					Manager: "controller-one",
					Object: `
						mapOfMapsRecursive:
						  a:
						    b:
						      c:
						        d:
						          e:
						  c:
						    d:
						      e:
						        f:
						          g:
					`,
					APIVersion: "v3",
				},
				ExtractApply{
					Apply{
						Manager: "apply-one",
						Object: `
						mapOfMapsRecursive:
						  a:
						    b:
						      c:
						        d:
						          e:
						            f:
					`,
						APIVersion: "v4",
					},
				},
			},
			Object: `
				mapOfMapsRecursive:
				  a:
				    b:
				      c:
				        d:
				          e:
				            f:
				  c:
				    d:
				      e:
				        f:
				          g:
			`,
			APIVersion: "v1",
			Managed: fieldpath.ManagedFields{
				"apply-one": fieldpath.NewVersionedSet(
					_NS(
						_P("mapOfMapsRecursive", "a"),
						_P("mapOfMapsRecursive", "a", "b"),
						_P("mapOfMapsRecursive", "a", "b", "c"),
						_P("mapOfMapsRecursive", "a", "b", "c", "d"),
						_P("mapOfMapsRecursive", "a", "b", "c", "d", "e"),
						_P("mapOfMapsRecursive", "a", "b", "c", "d", "e", "f"),
						_P("mapOfMapsRecursive", "c"),
						_P("mapOfMapsRecursive", "c", "d"),
						_P("mapOfMapsRecursive", "c", "d", "e"),
						_P("mapOfMapsRecursive", "c", "d", "e", "f"),
						_P("mapOfMapsRecursive", "c", "d", "e", "f", "g"),
					),
					"v4",
					false,
				),
				"apply-two": fieldpath.NewVersionedSet(
					_NS(
						_P("mapOfMapsRecursive", "a"),
						_P("mapOfMapsRecursive", "c"),
						_P("mapOfMapsRecursive", "c", "d"),
					),
					"v2",
					false,
				),
				"controller-one": fieldpath.NewVersionedSet(
					_NS(
						_P("mapOfMapsRecursive", "a", "b", "c"),
						_P("mapOfMapsRecursive", "a", "b", "c", "d", "e"),
						_P("mapOfMapsRecursive", "c", "d", "e"),
						_P("mapOfMapsRecursive", "c", "d", "e", "f", "g"),
					),
					"v3",
					false,
				),
				"controller-two": fieldpath.NewVersionedSet(
					_NS(
						_P("mapOfMapsRecursive", "a", "b", "c", "d"),
						_P("mapOfMapsRecursive", "c", "d", "e", "f"),
					),
					"v2",
					false,
				),
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if err := test.Test(nestedTypeParser); err != nil {
				t.Fatal(err)
			}
		})
	}
}
