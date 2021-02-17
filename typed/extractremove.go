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
	"sigs.k8s.io/structured-merge-diff/v4/fieldpath"
	"sigs.k8s.io/structured-merge-diff/v4/schema"
	"sigs.k8s.io/structured-merge-diff/v4/value"
)

type extractOrRemoveWalker struct {
	value             value.Value
	out               interface{}
	schema            *schema.Schema
	toExtractOrRemove *fieldpath.Set
	allocator         value.Allocator
	shouldExtract     bool
}

func extractOrRemoveItemsWithSchema(val value.Value, toExtract *fieldpath.Set, schema *schema.Schema, typeRef schema.TypeRef, shouldExtract bool) value.Value {
	w := &extractOrRemoveWalker{
		value:             val,
		schema:            schema,
		toExtractOrRemove: toExtract,
		allocator:         value.NewFreelistAllocator(),
		shouldExtract:     shouldExtract,
	}
	resolveSchema(schema, typeRef, val, w)
	return value.NewValueInterface(w.out)
}

func (w *extractOrRemoveWalker) doScalar(t *schema.Scalar) ValidationErrors {
	w.out = w.value.Unstructured()
	return nil
}

func (w *extractOrRemoveWalker) doList(t *schema.List) (errs ValidationErrors) {
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
		// save items that do have the path when we shouldExtract
		// but ignore it when we are removing (i.e. !w.shouldExtract)
		if w.toExtractOrRemove.Has(path) {
			if w.shouldExtract {
				newItems = append(newItems, item.Unstructured())
			} else {
				continue
			}
		}
		if subset := w.toExtractOrRemove.WithPrefix(pe); !subset.Empty() {
			item = extractOrRemoveItemsWithSchema(item, subset, w.schema, t.ElementType, w.shouldExtract)
		}
		// save items that do not have the path only when removing (i.e. !w.shouldExtract)
		if !w.shouldExtract {
			newItems = append(newItems, item.Unstructured())
		}
	}
	if len(newItems) > 0 {
		w.out = newItems
	}
	return nil
}

func (w *extractOrRemoveWalker) doMap(t *schema.Map) ValidationErrors {
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
		hasPath := w.toExtractOrRemove.Has(path)
		fieldType := t.ElementType
		if ft, ok := fieldTypes[k]; ok {
			fieldType = ft
		}
		if hasPath {
			if w.shouldExtract {
				newMap[k] = val.Unstructured()
			}
			return true
		}
		if subset := w.toExtractOrRemove.WithPrefix(pe); !subset.Empty() {
			val = extractOrRemoveItemsWithSchema(val, subset, w.schema, fieldType, w.shouldExtract)
		} else {
			if w.shouldExtract {
				return true
			}
		}
		newMap[k] = val.Unstructured()
		return true
	})
	if len(newMap) > 0 {
		w.out = newMap
	}
	return nil
}
