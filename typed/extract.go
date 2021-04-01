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

type extractingWalker struct {
	value         value.Value
	out           interface{}
	schema        *schema.Schema
	toExtract     *fieldpath.Set
	allocator     value.Allocator
	shouldExtract bool
	matchedPrev   bool
}

// extractItemsWithSchema will walk the given value and look for items from the toExtract set.
// Depending on whether shouldExtract is set true or false, it will return a modified version
// of the input value with either:
// 1. only the items in the toExtract set (when shouldExtract is true) or
// 2. the items from the toExtract set removed from the value (when shouldExtract is false).
func extractItemsWithSchema(val value.Value, toExtract *fieldpath.Set, schema *schema.Schema, typeRef schema.TypeRef, shouldExtract, matchedPrev bool) value.Value {
	w := &extractingWalker{
		value:         val,
		schema:        schema,
		toExtract:     toExtract,
		allocator:     value.NewFreelistAllocator(),
		shouldExtract: shouldExtract,
		matchedPrev:   matchedPrev,
	}
	resolveSchema(schema, typeRef, val, w)
	out := value.NewValueInterface(w.out)
	fmt.Printf("out = %+v\n", out)
	fmt.Printf("out.IsList() = %+v\n", out.IsList())
	return out
}

func (w *extractingWalker) doScalar(t *schema.Scalar) ValidationErrors {
	w.out = w.value.Unstructured()
	return nil
}

func (w *extractingWalker) doList(t *schema.List) (errs ValidationErrors) {
	fmt.Println("doList")
	l := w.value.AsListUsing(w.allocator)
	defer w.allocator.Free(l)
	if w.shouldExtract && t.ElementRelationship == schema.Atomic {
		w.out = w.value.Unstructured()
		return nil
	}
	//if w.matchedPrev {
	//	//fmt.Printf("list w.matchedPrev = %+v\n", w.matchedPrev)
	//	////fmt.Printf("l.IsList()) = %+v\n", l.IsList())
	//	//fmt.Printf("l = %+v\n", l)
	//	if t.ElementRelationship == schema.Atomic {
	//		w.out = l
	//	}
	//	return nil
	//}

	// If list is null, empty, or atomic just return
	if l == nil || l.Length() == 0 || t.ElementRelationship == schema.Atomic {
		return nil
	}

	var newItems []interface{}
	iter := l.RangeUsing(w.allocator)
	defer w.allocator.Free(iter)
	for iter.Next() {
		i, item := iter.Item()
		// Ignore error because we have already validated this list
		pe, _ := listItemToPathElement(w.allocator, w.schema, t, i, item)
		path, _ := fieldpath.MakePath(pe)
		// save items on the path when we shouldExtract
		// but ignore them when we are removing (i.e. !w.shouldExtract)
		if w.toExtract.Has(path) {
			fmt.Printf("path = %+v\n", path)
			if w.shouldExtract {
				itemIsAtomic, err := isAtomic(item, w.schema, t.ElementType)
				if err != nil {
					return err
				}
				if !itemIsAtomic && item.IsMap() {
					retainOnlyListKeys(t.Keys, item.AsMap())
				}
				newItems = append(newItems, item.Unstructured())
			} else {
				continue
			}
		}
		if subset := w.toExtract.WithPrefix(pe); !subset.Empty() {
			item = extractItemsWithSchema(item, subset, w.schema, t.ElementType, w.shouldExtract, false)
		} else {
			// don't save items not on the path when we shouldExtract.
			if w.shouldExtract {
				continue
			}
		}
		newItems = append(newItems, item.Unstructured())
	}
	if len(newItems) > 0 {
		w.out = newItems
	}
	return nil
}

func (w *extractingWalker) doMap(t *schema.Map) ValidationErrors {
	fmt.Println("doMap")
	if w.matchedPrev {
		fmt.Printf("map w.matchedPrev = %+v\n", w.matchedPrev)
		if t.ElementRelationship == schema.Atomic {
			w.out = w.value.Unstructured()
		}
		return nil
	}
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
	var errors ValidationErrors
	m.Iterate(func(k string, val value.Value) bool {
		fmt.Printf("k = %+v\n", k)
		fmt.Printf("val = %+v\n", val)
		pe := fieldpath.PathElement{FieldName: &k}
		fmt.Printf("pe = %+v\n", pe)
		path, _ := fieldpath.MakePath(pe)
		fieldType := t.ElementType
		if ft, ok := fieldTypes[k]; ok {
			fieldType = ft
		}
		// save values on the path when we shouldExtract
		// but ignore them when we are removing (i.e. !w.shouldExtract)
		fmt.Printf("w.toExtract = %+v\n", w.toExtract)
		if w.toExtract.Has(path) {
			fmt.Printf("path = %+v\n", path)
			if w.shouldExtract {
				subset := w.toExtract.WithPrefix(pe)
				fmt.Printf("subset = %+v\n", subset)
				//// OLD
				//valIsAtomic, err := isAtomic(val, w.schema, fieldType)
				//if err != nil {
				//	errors = err
				//	return false
				//}

				//if !valIsAtomic && (val.IsMap() || val.IsList()) {
				//	newMap[k] = nil
				//} else {
				//	fmt.Printf("val.Unstructured() = %+v\n", val.Unstructured())
				//	fmt.Printf("val = %+v\n", val)
				//	newMap[k] = val.Unstructured()
				//}

				// NEW
				fmt.Println("matchedExtract")
				exVal := extractItemsWithSchema(val, w.toExtract, w.schema, fieldType, w.shouldExtract, true)
				fmt.Printf("exVal = %+v\n", exVal)
				fmt.Printf("exVal.AsList() = %+v\n", exVal.IsList())
				unstructured := exVal.Unstructured()
				fmt.Printf("unstructured = %+v\n", unstructured)
				newMap[k] = unstructured

				fmt.Printf("newMap = %+v\n", newMap)
			}
			return true
		}
		if subset := w.toExtract.WithPrefix(pe); !subset.Empty() {
			val = extractItemsWithSchema(val, subset, w.schema, fieldType, w.shouldExtract, false)
		} else {
			// don't save values not on the path when we shouldExtract.
			if w.shouldExtract {
				return true
			}
		}
		newMap[k] = val.Unstructured()
		return true
	})
	if errors != nil {
		return errors
	}
	if len(newMap) > 0 {
		w.out = newMap
	}
	return nil
}
