package workers

import (
	"reflect"
	"testing"
)

func TestFailedItemCodesListsEveryAffectedItemOnce(t *testing.T) {
	got := failedItemCodes([]string{
		"6.1: selector no encontrado",
		"7.2.1: sesión de Zajuna expirada o página de login",
		"6.1: timeout",
		"captura cancelada sin código",
	})
	want := []string{"6.1", "7.2.1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("failedItemCodes = %v, want %v", got, want)
	}
}
