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

type removingWalker struct {
	value     value.Value
	out       interface{}
	schema    *schema.Schema
	toRemove  *fieldpath.Set
	allocator value.Allocator
}

func removeItemsWithSchema(val value.Value, toRemove *fieldpath.Set, schema *schema.Schema, typeRef schema.TypeRef) value.Value {
	w := &removingWalker{
		value:     val,
		schema:    schema,
		toRemove:  toRemove,
		allocator: value.NewFreelistAllocator(),
	}
	resolveSchema(schema, typeRef, val, w)
	fmt.Printf("final w.out = %+v\n", w.out)
	return value.NewValueInterface(w.out)
}

func (w *removingWalker) doScalar(t *schema.Scalar) ValidationErrors {
	fmt.Println("scalar rm called")
	w.out = w.value.Unstructured()
	fmt.Printf("rmScalar w.out = %+v\n", w.out)
	return nil
}

func (w *removingWalker) doList(t *schema.List) (errs ValidationErrors) {
	fmt.Println("list rm called")
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
		hasPath := w.toRemove.Has(path)
		fmt.Printf("pe = %+v\n", pe)
		fmt.Printf("path = %+v\n", path)
		fmt.Printf("hasPath = %+v\n", hasPath)
		fmt.Printf("w.toRemove = %+v\n", w.toRemove)
		if hasPath {
			continue
		}
		if subset := w.toRemove.WithPrefix(pe); !subset.Empty() {
			item = removeItemsWithSchema(item, subset, w.schema, t.ElementType)
		}
		newItems = append(newItems, item.Unstructured())
	}
	fmt.Printf("newItems = %+v\n", newItems)
	if len(newItems) > 0 {
		w.out = newItems
	}
	fmt.Printf("rmList w.out = %+v\n", w.out)
	return nil
}

func (w *removingWalker) doMap(t *schema.Map) ValidationErrors {
	fmt.Println("map rm called")
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
	m.Iterate(func(k string, val value.Value) bool {
		pe := fieldpath.PathElement{FieldName: &k}
		path, _ := fieldpath.MakePath(pe)
		hasPath := w.toRemove.Has(path)
		fieldType := t.ElementType
		fmt.Printf("pe = %+v\n", pe)
		fmt.Printf("path = %+v\n", path)
		fmt.Printf("fieldType = %+v\n", fieldType)
		fmt.Printf("w.toRemove = %+v\n", w.toRemove)
		if ft, ok := fieldTypes[k]; ok {
			fieldType = ft
		}
		if hasPath {
			return true
		}
		if subset := w.toRemove.WithPrefix(pe); !subset.Empty() {
			val = removeItemsWithSchema(val, subset, w.schema, fieldType)
		}
		newMap[k] = val.Unstructured()
		return true
	})
	if len(newMap) > 0 {
		w.out = newMap
	}
	fmt.Printf("rmMap w.out = %+v\n", w.out)
	return nil
}
