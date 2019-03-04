package note

import (
	"strings"
	"testing"
)

func Test_Result_String(t *testing.T) {
	result := &Result{
		ID:   1,
		Body: `{"Entities":[{"Attributes":[{"BeginOffset":1,"EndOffset":1,"Id":1,"RelationshipScore":1,"Score":1,"Text":"string","Traits":[{"Name":"string","Score":1}],"Type":"string"}],"BeginOffset":1,"Category":"string","EndOffset":1,"Id":1,"Score":1,"Text":"string","Traits":[{"Name":"string","Score":1}],"Type":"string"}],"PaginationToken":"string"}`,
	}
	str := result.String()

	if !strings.Contains(str, "\"Id\":1") {
		t.Errorf("unexpected result id string: %s", str)
	}

	if !strings.Contains(str, result.Body) {
		t.Errorf("body not found: %s", str)
	}
}
