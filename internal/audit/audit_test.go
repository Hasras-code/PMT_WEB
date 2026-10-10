package audit

import (
	"testing"

	"github.com/google/uuid"
)

func TestUUIDParamAllowsNullableEmptyValue(t *testing.T) {
	got, err := uuidParam("", true)
	if err != nil {
		t.Fatalf("uuidParam returned an error: %v", err)
	}
	if got.Valid {
		t.Fatal("empty nullable UUID should map to SQL NULL")
	}
}

func TestUUIDParamParsesRequiredValue(t *testing.T) {
	want := uuid.MustParse("f86cc4ee-29ad-452a-822b-e040140472f9")
	got, err := uuidParam(want.String(), false)
	if err != nil {
		t.Fatalf("uuidParam returned an error: %v", err)
	}
	if !got.Valid || uuid.UUID(got.Bytes) != want {
		t.Fatalf("uuidParam = %v, want %v", got, want)
	}
}

func TestUUIDParamRejectsEmptyRequiredValue(t *testing.T) {
	if _, err := uuidParam("", false); err == nil {
		t.Fatal("uuidParam accepted an empty required UUID")
	}
}
