package utils

const (

	// SceneActionRecallOperation 别的模块请求recall的operation
	SceneActionRecallOperation = "recall"
)

const (
	PublishToDeviceSceneStoreTopic  = "/scene/store"
	PublishToDeviceSceneRemoveTopic = "/scene/remove"
	PublishToDeviceSceneRecallTopic = "/scene/recall"
)

const (
	CloudTriggerTypeTimer    = "timer"
	CloudTriggerTypeProperty = "property"
	CloudTriggerTypeEvent    = "event"

	CloudConditionTypeTimer    = "timer"
	CloudConditionTypeProperty = "property"
)
