package callback

import (
	"hotkey-client/rule"
)

// ValueModel 值模型
type ValueModel struct {
	CreateTime int64       `json:"createTime"` // 创建时间（毫秒）
	Duration   int64       `json:"duration"`   // 持续时间（毫秒）
	Value      interface{} `json:"value"`      // 实际值
}

// DefaultValue 创建默认值模型
func DefaultValue(key string) *ValueModel {
	duration := rule.GetInstance().Duration(key)
	if duration <= 0 {
		return nil
	}

	return &ValueModel{
		CreateTime: 0,                      // 暂时设为0，实际使用时应该设置当前时间
		Duration:   int64(duration) * 1000, // 转换为毫秒
		Value:      nil,
	}
}

// GetCreateTime 获取创建时间
func (vm *ValueModel) GetCreateTime() int64 {
	return vm.CreateTime
}

// SetCreateTime 设置创建时间
func (vm *ValueModel) SetCreateTime(createTime int64) {
	vm.CreateTime = createTime
}

// GetDuration 获取持续时间
func (vm *ValueModel) GetDuration() int64 {
	return vm.Duration
}

// SetDuration 设置持续时间
func (vm *ValueModel) SetDuration(duration int64) {
	vm.Duration = duration
}

// GetValue 获取值
func (vm *ValueModel) GetValue() interface{} {
	return vm.Value
}

// SetValue 设置值
func (vm *ValueModel) SetValue(value interface{}) {
	vm.Value = value
}
