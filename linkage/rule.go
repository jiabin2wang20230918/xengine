package linkage

import (
	"errors"
	"fmt"
	"sync"
)

// DefaultMaxSceneId caps automatic SceneId allocation to the 1..255 range
// required by common Zigbee/PLC scene tables.
const DefaultMaxSceneId uint16 = 255

// ErrSceneIdExhausted is returned when all SceneIds in [1, maxSceneId] are in use.
var ErrSceneIdExhausted = errors.New("linkage: scene id pool exhausted")

type Rule struct {
	SceneId    uint16 `gorm:"primaryKey;autoIncrement:false"`
	RuleId     string `gorm:"uniqueIndex;not null"`
	Enable     bool
	Triggers   []byte
	Conditions []byte
	Actions    []byte
}

var (
	sceneIDMu    sync.Mutex
	maxSceneId   uint16 = DefaultMaxSceneId
	usedSceneIds        = make(map[uint16]struct{})
	nextSceneId  uint16 = 1
)

// SetMaxSceneId configures the upper bound (inclusive) for SceneId allocation.
func SetMaxSceneId(max uint16) {
	sceneIDMu.Lock()
	defer sceneIDMu.Unlock()
	if max == 0 {
		maxSceneId = DefaultMaxSceneId
	} else {
		maxSceneId = max
	}
	if nextSceneId == 0 || nextSceneId > maxSceneId {
		nextSceneId = 1
	}
}

func getMaxSceneId() uint16 {
	sceneIDMu.Lock()
	defer sceneIDMu.Unlock()
	return maxSceneId
}

func resetSceneIDPool(existing []uint16) {
	sceneIDMu.Lock()
	defer sceneIDMu.Unlock()
	usedSceneIds = make(map[uint16]struct{}, len(existing))
	for _, id := range existing {
		if id != 0 {
			usedSceneIds[id] = struct{}{}
		}
	}
	nextSceneId = 1
}

// AllocateSceneId reserves and returns the next available SceneId in [1, maxSceneId].
func AllocateSceneId() (uint16, error) {
	sceneIDMu.Lock()
	defer sceneIDMu.Unlock()

	limit := int(maxSceneId)
	if limit <= 0 {
		return 0, ErrSceneIdExhausted
	}
	if nextSceneId == 0 || nextSceneId > maxSceneId {
		nextSceneId = 1
	}

	candidate := nextSceneId
	for i := 0; i < limit; i++ {
		if _, used := usedSceneIds[candidate]; !used {
			usedSceneIds[candidate] = struct{}{}
			if candidate >= maxSceneId {
				nextSceneId = 1
			} else {
				nextSceneId = candidate + 1
			}
			return candidate, nil
		}
		if candidate >= maxSceneId {
			candidate = 1
		} else {
			candidate++
		}
	}
	return 0, ErrSceneIdExhausted
}

// GenerateSceneId returns the next available SceneId, or 0 if the pool is exhausted.
func GenerateSceneId() uint16 {
	id, err := AllocateSceneId()
	if err != nil {
		return 0
	}
	return id
}

// ReleaseSceneId returns a previously allocated SceneId to the pool for reuse.
func ReleaseSceneId(id uint16) {
	if id == 0 {
		return
	}
	sceneIDMu.Lock()
	defer sceneIDMu.Unlock()
	delete(usedSceneIds, id)
}

func markSceneIdUsed(id uint16) {
	if id == 0 {
		return
	}
	sceneIDMu.Lock()
	defer sceneIDMu.Unlock()
	usedSceneIds[id] = struct{}{}
}

func (r *Rule) SetRule() error {
	if r == nil {
		return errors.New("linkage: rule is nil")
	}
	if r.RuleId == "" {
		return errors.New("linkage: rule_id is empty")
	}

	allocated := false
	if r.SceneId == 0 {
		id, err := AllocateSceneId()
		if err != nil {
			return err
		}
		r.SceneId = id
		allocated = true
	} else {
		if max := getMaxSceneId(); r.SceneId > max {
			return fmt.Errorf("linkage: scene id %d exceeds max %d", r.SceneId, max)
		}
		markSceneIdUsed(r.SceneId)
	}

	err := upsertRuleTB(r)
	if err != nil && allocated {
		ReleaseSceneId(r.SceneId)
		r.SceneId = 0
	}
	return err
}

func (r *Rule) DeleteRule() error {
	if r == nil {
		return errors.New("linkage: rule is nil")
	}
	err := removeRuleTB(r.RuleId)
	if err == nil {
		ReleaseSceneId(r.SceneId)
	}
	return err
}

func (r *Rule) GetRule() error {
	return nil
}
