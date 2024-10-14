package firestore

import (
	"cloud.google.com/go/firestore"
	"context"
	"errors"
	"fmt"
	"google.golang.org/api/iterator"
)

func count(ctx context.Context, client *firestore.Client, collectionPath string) int {
	cr := client.Collection(collectionPath)
	if cr == nil {
		return 0
	}

	iter := cr.Documents(ctx)
	c := 0
	for {
		_, err := iter.Next()
		if err != nil {
			break
		}
		c++
	}
	return c
}

func create[T any](ctx context.Context, client *firestore.Client, documentPath string, t *T) error {
	dr := client.Doc(documentPath)
	if dr == nil {
		return fmt.Errorf("invalid document path, %s", documentPath)
	}

	if _, err := dr.Create(ctx, t); err != nil {
		return fmt.Errorf("error creating document, %s", err)
	}

	return nil
}

func get[T any](ctx context.Context, client *firestore.Client, documentPath string) (*T, error) {
	dr := client.Doc(documentPath)
	if dr == nil {
		return nil, fmt.Errorf("invalid document path, %s", documentPath)
	}

	ds, err := dr.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting document, %s", err)
	}

	t := new(T)
	if err = ds.DataTo(t); err != nil {
		return nil, fmt.Errorf("error decoding document, %s", err)
	}

	return t, nil
}

func set[T any](ctx context.Context, client *firestore.Client, documentPath string, t *T) error {
	dr := client.Doc(documentPath)
	if dr == nil {
		return fmt.Errorf("invalid document path, %s", documentPath)
	}

	if _, err := dr.Set(ctx, t); err != nil {
		return fmt.Errorf("error setting document contents, %s", err)
	}

	return nil
}

func update(ctx context.Context, client *firestore.Client, documentPath string, fields map[string]any) error {
	dr := client.Doc(documentPath)
	if dr == nil {
		return fmt.Errorf("invalid document path, %s", documentPath)
	}

	updates := make([]firestore.Update, 0)
	for k, v := range fields {
		updates = append(updates, firestore.Update{Path: k, Value: v})
	}

	if _, err := dr.Update(ctx, updates); err != nil {
		return fmt.Errorf("error updating document, %s", err)
	}

	return nil
}

func remove(ctx context.Context, client *firestore.Client, path string) error {
	cr := client.Collection(path)
	dr := client.Doc(path)

	if cr != nil {
		err := removeCollection(ctx, client, cr)
		if err != nil {
			return err
		}
		return nil
	}

	if dr != nil {
		err := removeDocument(ctx, client, dr)
		if err != nil {
			return err
		}
		return nil
	}

	return fmt.Errorf("invalid path format, %s", path)
}

func removeCollection(ctx context.Context, client *firestore.Client, cr *firestore.CollectionRef) error {
	if cr == nil {
		return fmt.Errorf("invalid collection reference, %s", cr.Path)
	}

	removed := 0

	iter := cr.Documents(ctx)
	for {
		ds, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}

		if err = removeDocument(ctx, client, ds.Ref); err != nil {
			return fmt.Errorf("error deleting document, %s", err)
		}

		removed++
	}

	if removed == 0 {
		return fmt.Errorf("invalid collection reference, %s", cr.Path)
	}

	return nil
}

func removeDocument(ctx context.Context, client *firestore.Client, dr *firestore.DocumentRef) error {
	if dr == nil {
		return fmt.Errorf("invalid document reference, %s", dr.Path)
	}

	iter := dr.Collections(ctx)
	for {
		cr, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return err
		}

		if err = removeCollection(ctx, client, cr); err != nil {
			return fmt.Errorf("error deleting collection, %s", err)
		}
	}

	if _, err := dr.Delete(ctx); err != nil {
		return fmt.Errorf("error deleting document, %s", err)
	}

	return nil
}

func list[T any](ctx context.Context, client *firestore.Client, collectionPath string) ([]*T, error) {
	cr := client.Collection(collectionPath)
	if cr == nil {
		return nil, fmt.Errorf("invalid collection path, %s", collectionPath)
	}

	iter := cr.Documents(ctx)
	ds, err := iter.GetAll()
	if err != nil {
		return nil, fmt.Errorf("error listing documents, %s", err)
	}

	documents := make([]*T, 0)
	for _, d := range ds {
		t := new(T)
		if err = d.DataTo(t); err != nil {
			return nil, fmt.Errorf("error decoding document, %s", err)
		}
		documents = append(documents, t)
	}

	return documents, nil
}

func query[T any](ctx context.Context, client *firestore.Client, criteria QueryCriteria) ([]*T, error) {
	cr := client.Collection(criteria.Path)
	if cr == nil {
		return nil, fmt.Errorf("invalid collection path, %s", criteria.Path)
	}

	var q firestore.Query
	if criteria.Filter == nil {
		q = cr.Offset(criteria.Offset)
	} else {
		q = cr.WhereEntity(criteria.Filter).Offset(criteria.Offset)
	}

	if len(criteria.OrderBy) > 0 {
		for _, o := range criteria.OrderBy {
			q = q.OrderBy(o.Field, o.Direction)
		}
	}

	if criteria.Limit > 0 {
		q = q.Limit(criteria.Limit)
	}

	iter := q.Documents(ctx)
	ds, err := iter.GetAll()
	if err != nil {
		return nil, fmt.Errorf("error querying documents, %s", err)
	}

	documents := make([]*T, 0)
	for _, d := range ds {
		t := new(T)
		if err = d.DataTo(t); err != nil {
			return nil, fmt.Errorf("error decoding document, %s", err)
		}
		documents = append(documents, t)
	}

	return documents, nil
}

func exists[T any](ctx context.Context, client *firestore.Client, criteria QueryCriteria) (bool, error) {
	cr := client.Collection(criteria.Path)
	if cr == nil {
		return false, fmt.Errorf("invalid collection path, %s", criteria.Path)
	}

	var q firestore.Query
	if criteria.Filter == nil {
		q = cr.Offset(criteria.Offset)
	} else {
		q = cr.WhereEntity(criteria.Filter).Offset(criteria.Offset)
	}

	if len(criteria.OrderBy) > 0 {
		for _, o := range criteria.OrderBy {
			q = q.OrderBy(o.Field, o.Direction)
		}
	}

	q = q.Limit(1)

	iter := q.Documents(ctx)
	ds, err := iter.GetAll()
	if err != nil {
		return false, fmt.Errorf("error querying documents, %s", err)
	}

	if len(ds) == 0 {
		return false, nil
	}

	t := new(T)
	if err = ds[0].DataTo(t); err != nil {
		return false, fmt.Errorf("error decoding document, %s", err)
	}

	return t != nil, nil
}

type QueryCriteria struct {
	Path    string
	Filter  firestore.EntityFilter
	OrderBy []OrderBy
	Limit   int
	Offset  int
}

type OrderBy struct {
	Field     string
	Direction firestore.Direction
}

const (
	Equal              string = "=="
	NotEqual           string = "!="
	GreaterThan        string = ">"
	GreaterThanOrEqual string = ">="
	LessThan           string = "<"
	LessThanOrEqual    string = "<="
	In                 string = "in"
	NotIn              string = "not-in"
	ArrayContains      string = "array-contains"
	ArrayContainsAny   string = "array-contains-any"
)

func createQueryCriteria(collection, path, operator string, value any) QueryCriteria {
	return QueryCriteria{
		Path:   collection,
		Filter: createPropertyFilter(path, operator, value),
	}
}

func createPropertyFilter(path, operator string, value any) firestore.PropertyFilter {
	return firestore.PropertyFilter{
		Path:     path,
		Operator: operator,
		Value:    value,
	}
}
