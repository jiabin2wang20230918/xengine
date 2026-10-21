package linkage

type Rule struct {
	SceneId    uint16 `gorm:"primary_key"` //需确认zigbee sceneId采用自增长是否OK，超过255是否会出问题??
	RuleId     string `gorm:"primary_key"`
	Enable     bool
	Triggers   []byte `gorm:"serializer:json"`
	Conditions []byte `gorm:"serializer:json"`
	Actions    []byte `gorm:"serializer:json"`
}

var lastDeleteSceneId uint16 = 0

func GenerateSceneId() uint16 {
	return lastDeleteSceneId
}

func (r *Rule) SetRule() error {
	err := upsertRuleTB(r)
	if r.SceneId == lastDeleteSceneId {
		lastDeleteSceneId = 0
	}
	return err
}

func (r *Rule) DeleteRule() error {
	lastDeleteSceneId = r.SceneId
	err := removeRuleTB(r.RuleId)
	return err
}

func (r *Rule) GetRule() error {
	return nil
}
