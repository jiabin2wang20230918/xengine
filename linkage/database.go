package linkage

import (
	"errors"
	"fmt"
	"sync"

	"gorm.io/gorm"

	"github.com/jiabin2wang20230918/xengine/utils/log"
)

var (
	dbMu      sync.RWMutex
	linkageDb *gorm.DB
)

func getDB() *gorm.DB {
	dbMu.RLock()
	defer dbMu.RUnlock()
	return linkageDb
}

// InitRuleDb migrates the rules schema and seeds the in-memory SceneId pool from existing rows.
func InitRuleDb(sDb *gorm.DB) error {
	if sDb == nil {
		dbMu.Lock()
		linkageDb = nil
		dbMu.Unlock()
		resetSceneIDPool(nil)
		return errors.New("linkage: nil db")
	}

	if err := sDb.AutoMigrate(&Rule{}); err != nil {
		log.Errorf("AutoMigrate Rule failed: %v", err)
		return err
	}

	var existing []Rule
	if err := sDb.Select("scene_id").Find(&existing).Error; err != nil {
		log.Errorf("load existing scene ids failed: %v", err)
		return err
	}
	ids := make([]uint16, 0, len(existing))
	for _, r := range existing {
		ids = append(ids, r.SceneId)
	}

	dbMu.Lock()
	linkageDb = sDb
	dbMu.Unlock()
	resetSceneIDPool(ids)
	return nil
}

func upsertRuleTB(con *Rule) error {
	db := getDB()
	if db == nil || con == nil {
		err := fmt.Errorf("db error or rule nil")
		log.Warnf("UpsertRule failed: %v", err)
		return err
	}
	if err := db.Save(con).Error; err != nil {
		log.Errorf("UpsertRule %v failed:%v", con, err)
		return err
	}
	return nil
}

func LoadRuleTB() ([]Rule, error) {
	db := getDB()
	if db == nil {
		err := fmt.Errorf("linkageDb nil")
		log.Warnf("LoadRule failed:%v", err)
		return nil, err
	}
	var p []Rule
	if err := db.Find(&p).Error; err != nil {
		log.Errorf("%v", err)
		return nil, err
	}
	return p, nil
}

func removeRuleTB(ruleid string) error {
	db := getDB()
	if db == nil || ruleid == "" {
		err := fmt.Errorf("db error or key nil")
		log.Errorf("RemoveRuleTB %s failed:%v", ruleid, err)
		return err
	}
	if err := db.Where("rule_id = ?", ruleid).Delete(&Rule{}).Error; err != nil {
		log.Errorf("RemoveRuleTB %s failed:%v", ruleid, err)
		return err
	}
	return nil
}

func QueryRuleBySceneId(sceneId uint16) (*Rule, error) {
	db := getDB()
	if db == nil {
		err := fmt.Errorf("linkageDb nil")
		log.Warnf("queryRuleBySceneId %v failed:%v", sceneId, err)
		return nil, err
	}
	p := &Rule{}
	if err := db.First(p, "scene_id = ?", sceneId).Error; err != nil {
		log.Errorf("%v", err)
		return nil, err
	}
	return p, nil
}
