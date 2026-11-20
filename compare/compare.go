// Package compare evaluates property comparisons via github.com/antonmedv/expr.
//
// Supported operators: "==", "!=", ">=", "<=", ">", "<", "between", "in", "not in", "like".
package compare

import (
	"fmt"

	"github.com/antonmedv/expr"

	"github.com/jiabin2wang20230918/xengine/utils/log"
)

// PropertyMatch compares propertyValue against targetPropertyValue using symbol.
// Any unsupported symbol or evaluation error returns false.
func PropertyMatch(propertyValue interface{}, targetPropertyValue interface{}, symbol string) bool {
	env := map[string]interface{}{}
	var input string
	switch symbol {
	case "between":
		env["p"] = propertyValue
		arr, ok := targetPropertyValue.([]interface{})
		if !ok {
			log.Errorf("parse error, should be []interface{}")
			return false
		}
		if len(arr) != 2 {
			log.Errorf("between arr length should be 2")
			return false
		}
		env["small"] = arr[0]
		env["big"] = arr[1]
		input = "p >= small && p <= big"
	case "in":
		env["p"] = propertyValue
		env["t"] = targetPropertyValue
		input = "p in t"
	case "not in":
		env["p"] = propertyValue
		env["t"] = targetPropertyValue
		input = "p not in t"
	case "==", "!=", ">=", "<=", ">", "<":
		env["p"] = propertyValue
		env["t"] = targetPropertyValue
		input = fmt.Sprintf("p %s t", symbol)
	case "like":
		env["p"] = propertyValue
		env["t"] = targetPropertyValue
		input = "p contains t"
	default:
		log.Errorf("unsupported symbol: %s", symbol)
		return false
	}

	log.Debugf("property value:%v symbol:%v target:%v", propertyValue, symbol, targetPropertyValue)
	eval, err := expr.Eval(input, env)
	if err != nil {
		log.Errorf("err:%v", err)
		return false
	}
	result, ok := eval.(bool)
	return ok && result
}
