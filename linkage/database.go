package linkage

import (
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/jiabin2wang20230918/xengine/utils/log"
)

var linkageDb *gorm.DB

func InitRuleDb(sDb *gorm.DB) {
	linkageDb = sDb
	linkageDb.AutoMigrate(&Rule{})
}

func upsertRuleTB(con *Rule) error {

	if linkageDb != nil && con != nil {
		var err error

		err = linkageDb.Save(con).Error
		if err != nil {
			log.Errorf("UpsertRule %v failed:%v, maybe key existed", con, err)
		}

		return err
	}
	err1 := fmt.Errorf("db error or key nil")
	log.Warnf("UpsertRule failed: %v", err1)
	return err1
}

func LoadRuleTB() ([]Rule, error) {
	if linkageDb != nil {
		var p []Rule
		err := linkageDb.Find(&p).Error
		if err != nil {
			log.Errorf("%v", err)
			return nil, err
		}
		return p, nil
	}
	err1 := fmt.Errorf("linkageDb nil")
	log.Warnf("LoadRule %v failed:%v", err1)
	return nil, err1
}

func removeRuleTB(ruleid string) error {
	if linkageDb != nil && ruleid != "" {
		var err error
		err = linkageDb.Delete(Rule{}, "rule_id = ?", ruleid).Error
		if err != nil {
			log.Errorf("RemoveRuleTB %s failed:%v, ", ruleid, err)
		}
		return err
	}

	err1 := fmt.Errorf("db error or key nil")
	log.Errorf("RemoveRuleTB %s failed:%v, ", ruleid, err1)
	return err1
}

func QueryRuleBySceneId(sceneId uint16) (*Rule, error) {
	if linkageDb != nil {
		p := &Rule{}
		err := linkageDb.Find(&p, "scene_id = ?", sceneId).Error
		if err != nil {
			log.Errorf("%v", err)
			return nil, err
		}
		return p, nil
	}
	err1 := fmt.Errorf("linkageDb nil")
	log.Warnf("queryRuleBySceneId %v failed:%v", sceneId, err1)
	return nil, err1
}
