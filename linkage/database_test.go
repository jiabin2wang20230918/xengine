package linkage

import (
	"encoding/json"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"log"
	"os"
	"testing"
)

func TestRule_SetRule(t *testing.T) {
	triggers, _ := json.Marshal(`[{}]`)
	conditions, _ := json.Marshal(`[{}]`)
	actions, _ := json.Marshal(`[{}]`)

	pwd, _ := os.Getwd()
	db, err := gorm.Open("sqlite3", pwd+"_test.db")
	if err != nil {
		log.Fatalf("Open database failed:%v", err)
		return
	}
	InitRuleDb(db)

	r0 := &Rule{
		RuleId:     "abcdef1122334455",
		Triggers:   triggers,
		Conditions: conditions,
		Actions:    actions,
	}

	r0.SetRule()

	r1 := &Rule{
		RuleId:     "123456789abcdef",
		Triggers:   triggers,
		Conditions: conditions,
		Actions:    actions,
	}
	r1.SetRule()

	rules, err := LoadRuleTB()
	if err != nil {
		log.Fatalf("%v", err)
		return
	}
	for _, r := range rules {
		log.Println(r.RuleId)
	}
}
