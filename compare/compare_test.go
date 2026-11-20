package compare

import (
	"encoding/json"
	"testing"
)

func TestPropertyMatchOperators(t *testing.T) {
	tests := []struct {
		name   string
		val    interface{}
		target interface{}
		symbol string
		want   bool
	}{
		{"int <", 1, 2, "<", true},
		{"int <=", 1, 2, "<=", true},
		{"int > false", 1, 2, ">", false},
		{"int >= false", 1, 2, ">=", false},
		{"int == false", 1, 2, "==", false},
		{"int !=", 1, 2, "!=", true},
		{"int == true", 2, 2, "==", true},
		{"float <", 1.1, 2.2, "<", true},
		{"float ==", 2.2, 2.2, "==", true},
		{"string <", "1671688678770", "1671688778770", "<", true},
		{"string ==", "abc", "abc", "==", true},
		{"like match", "qqqabceeee", "abc", "like", true},
		{"like mismatch", "qqqeeee", "abc", "like", false},
		{"between match", 5, []interface{}{1, 7}, "between", true},
		{"between out of range", 9, []interface{}{1, 7}, "between", false},
		{"between wrong length", 5, []interface{}{1}, "between", false},
		{"between wrong type", 5, "1:7", "between", false},
		{"in match", 2, []interface{}{1, 2, 4}, "in", true},
		{"in mismatch", 3, []interface{}{1, 2, 4}, "in", false},
		{"not in match", 3, []interface{}{1, 2, 4}, "not in", true},
		{"not in mismatch", 2, []interface{}{1, 2, 4}, "not in", false},
		{"unsupported symbol", 1, 1, "===", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := PropertyMatch(tc.val, tc.target, tc.symbol); got != tc.want {
				t.Fatalf("PropertyMatch(%v, %v, %q) = %v, want %v", tc.val, tc.target, tc.symbol, got, tc.want)
			}
		})
	}
}

func TestPropertyMatchJSONDecodedTargets(t *testing.T) {
	raw := []byte(`{"svalue": ["1", "7"], "ivalue": [1, 7]}`)
	var decoded struct {
		SValue interface{} `json:"svalue"`
		IValue interface{} `json:"ivalue"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if !PropertyMatch(float64(5), decoded.IValue, "between") {
		t.Fatal("expected 5 between [1, 7]")
	}
	if !PropertyMatch("5", decoded.SValue, "between") {
		t.Fatal("expected \"5\" between [\"1\", \"7\"]")
	}
	if !PropertyMatch(float64(1), decoded.IValue, "in") {
		t.Fatal("expected 1 in [1, 7]")
	}
	if !PropertyMatch("7", decoded.SValue, "in") {
		t.Fatal("expected \"7\" in [\"1\", \"7\"]")
	}
}
