package domain

import (
	"reflect"
	"strings"
	"testing"
)

func TestProductAttributeValues_UsesJSONSerializer(t *testing.T) {
	tpe := reflect.TypeOf(ProductAttribute{})
	field, ok := tpe.FieldByName("Values")
	if !ok {
		t.Fatalf("field Values not found")
	}

	gormTag := field.Tag.Get("gorm")
	if !strings.Contains(gormTag, "type:jsonb") {
		t.Fatalf("expected type:jsonb in gorm tag, got %q", gormTag)
	}
	if !strings.Contains(gormTag, "serializer:json") {
		t.Fatalf("expected serializer:json in gorm tag, got %q", gormTag)
	}
}

func TestProductVariantAttributeValues_UsesJSONSerializer(t *testing.T) {
	tpe := reflect.TypeOf(ProductVariant{})
	field, ok := tpe.FieldByName("AttributeValues")
	if !ok {
		t.Fatalf("field AttributeValues not found")
	}

	gormTag := field.Tag.Get("gorm")
	if !strings.Contains(gormTag, "type:jsonb") {
		t.Fatalf("expected type:jsonb in gorm tag, got %q", gormTag)
	}
	if !strings.Contains(gormTag, "serializer:json") {
		t.Fatalf("expected serializer:json in gorm tag, got %q", gormTag)
	}
}
