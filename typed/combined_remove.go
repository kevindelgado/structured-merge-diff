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

package typed

import (
	"fmt"

	"sigs.k8s.io/structured-merge-diff/v4/fieldpath"
	"sigs.k8s.io/structured-merge-diff/v4/schema"
	"sigs.k8s.io/structured-merge-diff/v4/value"
)

type combinedRemovingWalker struct {
	value         value.Value
	out           interface{}
	schema        *schema.Schema
	toRemove      *fieldpath.Set
	allocator     value.Allocator
	shouldExtract bool
}

// combinedRemoveItemsWithSchema will walk the given value and look for items from the toRemove set.
// Depending on whether shouldExtract is set true or false, it will return a modified version
// of the input value with either:
// 1. only the items in the toRemove set (when shouldExtract is true) or
// 2. the items from the toRemove set combinedRemoved from the value (when shouldExtract is false).
func combinedRemoveItemsWithSchema(val value.Value, toRemove *fieldpath.Set, schema *schema.Schema, typeRef schema.TypeRef, shouldExtract bool) value.Value {
	w := &combinedRemovingWalker{
		value:         val,
		schema:        schema,
		toRemove:      toRemove,
		allocator:     value.NewFreelistAllocator(),
		shouldExtract: shouldExtract,
	}
	resolveSchema(schema, typeRef, val, w)
	return value.NewValueInterface(w.out)
}

func (w *combinedRemovingWalker) doScalar(t *schema.Scalar) ValidationErrors {
	fmt.Println("doScalar")
	w.out = w.value.Unstructured()
	return nil
}

func (w *combinedRemovingWalker) doList(t *schema.List) (errs ValidationErrors) {
	if w.shouldExtract {
		fmt.Println("extract list")
	}
	fmt.Println("doList")
	l := w.value.AsListUsing(w.allocator)
	defer w.allocator.Free(l)
	// If list is null, empty, or atomic just return
	if l == nil || l.Length() == 0 || t.ElementRelationship == schema.Atomic {
		return nil
	}

	var newItems []interface{}
	iter := l.RangeUsing(w.allocator)
	defer w.allocator.Free(iter)
	for iter.Next() {
		i, item := iter.Item()
		fmt.Printf("list i = %+v\n", i)
		fmt.Printf("iter item.Unstructured() = %+v\n", item.Unstructured())
		// Ignore error because we have already validated this list
		pe, _ := listItemToPathElement(w.allocator, w.schema, t, i, item)
		path, _ := fieldpath.MakePath(pe)
		fmt.Printf("path = %+v\n", path)
		// save items that do have the path when we shouldExtract
		// but ignore it when we are combinedRemoving (i.e. !w.shouldExtract)
		if w.toRemove.Has(path) {
			fmt.Println("hasPath")
			if w.shouldExtract {
				newItems = append(newItems, item.Unstructured())
				fmt.Printf("item.Unstructured() = %+v\n", item.Unstructured())
				fmt.Printf("newItems = %+v\n", newItems)
			} else {
				continue
			}
		} else {
			fmt.Println("noPath")
		}

		if subset := w.toRemove.WithPrefix(pe); !subset.Empty() {
			fmt.Println("subset not empty")

			item = combinedRemoveItemsWithSchema(item, subset, w.schema, t.ElementType, w.shouldExtract)
			fmt.Printf("subitem item.Unstructured() = %+v\n", item.Unstructured())
			// new code
			//if w.shouldExtract {
			//	// we need to store the item with the parent here
			//	newItems = append(newItems,  item.Unstructured())
			//}
			// end new
		}
		// save items that do not have the path only when combinedRemoving (i.e. !w.shouldExtract)
		if !w.shouldExtract {
			newItems = append(newItems, item.Unstructured())
			fmt.Printf("!!list newItems = %+v\n", newItems)
		}
	}
	fmt.Printf("len(newItems) = %+v\n", len(newItems))
	fmt.Printf("pre list w.out = %+v\n", w.out)
	if len(newItems) > 0 {
		w.out = newItems
	}
	fmt.Printf("list w.out = %+v\n", w.out)
	return nil
}

func (w *combinedRemovingWalker) doMap(t *schema.Map) ValidationErrors {
	if w.shouldExtract {
		fmt.Println("extract map")
	}
	fmt.Println("doMap")
	m := w.value.AsMapUsing(w.allocator)
	if m != nil {
		defer w.allocator.Free(m)
	}
	// If map is null, empty, or atomic just return
	if m == nil || m.Empty() || t.ElementRelationship == schema.Atomic {
		return nil
	}

	fieldTypes := map[string]schema.TypeRef{}
	for _, structField := range t.Fields {
		fieldTypes[structField.Name] = structField.Type
	}

	newMap := map[string]interface{}{}
	i := -1
	m.Iterate(func(k string, val value.Value) bool {
		i++
		fmt.Printf("map i = %+v\n", i)
		fmt.Printf("k = %+v\n", k)
		fmt.Printf("iter val.Unstructured() = %+v\n", val.Unstructured())
		pe := fieldpath.PathElement{FieldName: &k}
		path, _ := fieldpath.MakePath(pe)
		fieldType := t.ElementType
		if ft, ok := fieldTypes[k]; ok {
			fieldType = ft
		}
		fmt.Printf("path = %+v\n", path)
		// save items on the path only when extracting.
		if w.toRemove.Has(path) {
			fmt.Println("hasPath")
			if w.shouldExtract {
				newMap[k] = val.Unstructured()
				fmt.Printf("k = %+v\n", k)
				fmt.Printf("val.Unstrutured = %+v\n", val.Unstructured())
				fmt.Printf("newMap = %+v\n", newMap)
			}
			return true
		}
		if subset := w.toRemove.WithPrefix(pe); !subset.Empty() {
			fmt.Println("subset not empty")
			val = combinedRemoveItemsWithSchema(val, subset, w.schema, fieldType, w.shouldExtract)
			fmt.Printf("NOT EMPTY val.Unstructured() = %+v\n", val.Unstructured())
		} else {
			fmt.Println("subset IS empty")
			// don't save items not on the path when extracting.
			if w.shouldExtract {
				return true
			}
		}
		newMap[k] = val.Unstructured()
		fmt.Printf("final k = %+v\n", k)
		fmt.Printf("final val.Unstrutured = %+v\n", val.Unstructured())
		fmt.Printf("final newMap = %+v\n", newMap)
		return true
	})
	fmt.Printf("pre map w.out = %+v\n", w.out)
	if len(newMap) > 0 {
		w.out = newMap
	}
	fmt.Printf("map w.out = %+v\n", w.out)
	return nil
}
