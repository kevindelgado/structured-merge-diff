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
	value     value.Value
	out       interface{}
	schema    *schema.Schema
	toExtract *fieldpath.Set
	allocator value.Allocator
}

func extractItemsWithSchema(val value.Value, toExtract *fieldpath.Set, schema *schema.Schema, typeRef schema.TypeRef) value.Value {
	w := &extractingWalker{
		value:     val,
		schema:    schema,
		toExtract: toExtract,
		allocator: value.NewFreelistAllocator(),
	}
	resolveSchema(schema, typeRef, val, w)
	fmt.Printf("final w.out = %+v\n", w.out)
	return value.NewValueInterface(w.out)
}

func (w *extractingWalker) doScalar(t *schema.Scalar) ValidationErrors {
	fmt.Println("scalar ex called")
	w.out = w.value.Unstructured()
	fmt.Printf("exScalar w.out = %+v\n", w.out)
	return nil
}

func (w *extractingWalker) doList(t *schema.List) (errs ValidationErrors) {
	fmt.Println("list ex called")
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
		// Ignore error because we have already validated this list
		pe, _ := listItemToPathElement(w.allocator, w.schema, t, i, item)
		path, _ := fieldpath.MakePath(pe)
		hasPath := w.toExtract.Has(path)
		fmt.Printf("pe = %+v\n", pe)
		fmt.Printf("path = %+v\n", path)
		fmt.Printf("hasPath = %+v\n", hasPath)
		fmt.Printf("w.toExtract = %+v\n", w.toExtract)
		if hasPath {
			newItems = append(newItems, item.Unstructured())
			//continue
		}
		if subset := w.toExtract.WithPrefix(pe); !subset.Empty() {
			item = extractItemsWithSchema(item, subset, w.schema, t.ElementType)
		}
		//newItems = append(newItems, item.Unstructured())
	}
	fmt.Printf("newItems = %+v\n", newItems)
	if len(newItems) > 0 {
		w.out = newItems
	}
	fmt.Printf("exList w.out = %+v\n", w.out)
	return nil
}

func (w *extractingWalker) doMap(t *schema.Map) ValidationErrors {
	fmt.Println("map ex called")
	m := w.value.AsMapUsing(w.allocator)
	if m != nil {
		defer w.allocator.Free(m)
	}
	// If map is null, empty, or atomic just return
	if m == nil || m.Empty() || t.ElementRelationship == schema.Atomic {
		fmt.Println("NULL RETURN")
		return nil
	}

	fieldTypes := map[string]schema.TypeRef{}
	for _, structField := range t.Fields {
		fmt.Printf("structField = %+v\n", structField)
		fieldTypes[structField.Name] = structField.Type
	}

	newMap := map[string]interface{}{}
	m.Iterate(func(k string, val value.Value) bool {
		pe := fieldpath.PathElement{FieldName: &k}
		path, _ := fieldpath.MakePath(pe)
		hasPath := w.toExtract.Has(path)
		fieldType := t.ElementType
		fmt.Printf("pe = %+v\n", pe)
		fmt.Printf("path = %+v\n", path)
		fmt.Printf("fieldType = %+v\n", fieldType)
		fmt.Printf("w.toExtract = %+v\n", w.toExtract)
		if ft, ok := fieldTypes[k]; ok {
			fieldType = ft
		}
		// what does it  mean to not have path?
		// why does w.toExtract not Have the .list path?
		// Do I need to do some sort of is subpath or something? (Maybe I should check how remove works)
		if hasPath {
			//fmt.Println("toExtract doesn't have path, returning true")
			// why return true
			newMap[k] = val.Unstructured()
			return true
			//return true
			//fmt.Println("toExtract doesn't have path, recursing")
			//val = extractItemsWithSchema(val, w.toExtract, w.schema, fieldType)
		}
		//  what are we doing with prefix?
		subset := w.toExtract.WithPrefix(pe)
		if subset.Empty() {
			return true
		}
		val = extractItemsWithSchema(val, subset, w.schema, fieldType)
		newMap[k] = val.Unstructured()
		// why return true?
		return true
	})
	if len(newMap) > 0 {
		w.out = newMap
	}
	fmt.Printf("exMap w.out = %+v\n", w.out)
	return nil
}
