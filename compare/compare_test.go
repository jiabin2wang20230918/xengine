package compare

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestCom(t *testing.T) {
	t.Log(PropertyMatch(1, 2, "<"))
	t.Log(PropertyMatch(1, 2, "<="))
	t.Log(!PropertyMatch(1, 2, ">"))
	t.Log(!PropertyMatch(1, 2, ">="))
	t.Log(!PropertyMatch(1, 2, "=="))
	t.Log(PropertyMatch(1, 2, "!="))
	t.Log(PropertyMatch(2, 2, "=="))
	t.Log(PropertyMatch(1.1, 2.2, "<"))
	t.Log(PropertyMatch(1.1, 2.2, "<="))
	t.Log(!PropertyMatch(1.1, 2.2, ">"))
	t.Log(!PropertyMatch(1.1, 2.2, ">="))
	t.Log(!PropertyMatch(1.1, 2.2, "=="))
	t.Log(PropertyMatch(1.1, 2.2, "!="))
	t.Log(PropertyMatch(2.2, 2.2, "=="))
	t.Log(PropertyMatch("1671688678770", "1671688778770", "<"))
	t.Log(PropertyMatch("1671688678770", "1671688778770", "<="))
	t.Log(!PropertyMatch("1671688678770", "1671688778770", ">"))
	t.Log(!PropertyMatch("1671688678770", "1671688778770", ">="))
	t.Log(!PropertyMatch("1671688678770", "1671688778770", "=="))
	t.Log(PropertyMatch("1671688678770", "1671688778770", "!="))
	t.Log(PropertyMatch("1671688778770", "1671688778770", "=="))
	t.Log(PropertyMatch("abc", "abc", "=="))
	t.Log(PropertyMatch("qqqabceeee", "abc", "like"))
	//t.Log(PropertyMatch(5, `1:5`, "between"))
	//t.Log(PropertyMatch("5", `"1":"5"`, "between"))
	//t.Log(PropertyMatch(4, "[1, 2, 4]", "in"))
	//t.Log(PropertyMatch(3, "[1, 2, 4]", "not in"))
	//t.Log(PropertyMatch("main", `["default", "case", "main"]`, "in"))
}

func TestName(t *testing.T) {
	j := []byte(`{ "onOff": 1, "svalue": ["1", "7"], "ivalue": [1, 7]}`)
	type Temp struct {
		OnOff  interface{} `json:"onOff"`
		SValue interface{} `json:"svalue"`
		IValue interface{} `json:"ivalue"`
	}
	temp := &Temp{}
	err := json.Unmarshal(j, temp)
	if err != nil {
		t.Logf("err:%v", err)
		return
	}

	t.Log(temp.OnOff)
	t.Log(temp.SValue)
	t.Log(temp.IValue)

	sv1, ok := temp.SValue.([]interface{})
	if ok {
		t.Log(sv1)
		t.Log(reflect.TypeOf(sv1[0]))
	}
	sv2, ok := temp.IValue.([]interface{})
	if ok {
		t.Log(sv2)
		t.Log(reflect.TypeOf(sv2[0]))
	}

	t.Log(PropertyMatch(5, temp.IValue, "between"))
	t.Log(PropertyMatch("5", temp.SValue, "between"))
	t.Log(PropertyMatch(1, temp.IValue, "in"))
	t.Log(PropertyMatch("7", temp.SValue, "in"))
}
