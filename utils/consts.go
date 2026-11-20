package utils

const (

	// SceneActionRecallOperation is the beehive operation used to request a scene recall.
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
