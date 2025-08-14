package serializer

import (
	"bytes"
	"fmt"
	"reflect"

	"github.com/jd/platform/hotkey/client-go/model"
)

// ProtostuffSerializer Protostuff序列化器，兼容Java版本
type ProtostuffSerializer struct{}

// NewProtostuffSerializer 创建新的Protostuff序列化器
func NewProtostuffSerializer() *ProtostuffSerializer {
	return &ProtostuffSerializer{}
}

// Serialize 序列化对象为字节数组，兼容Java的ProtostuffUtils.serialize
func (ps *ProtostuffSerializer) Serialize(obj interface{}) ([]byte, error) {
	switch v := obj.(type) {
	case *model.HotKeyMsg:
		return ps.serializeHotKeyMsg(v)
	default:
		return nil, fmt.Errorf("unsupported type: %T", obj)
	}
}

// Deserialize 反序列化字节数组为对象，兼容Java的ProtostuffUtils.deserialize
func (ps *ProtostuffSerializer) Deserialize(data []byte, objType reflect.Type) (interface{}, error) {
	switch objType {
	case reflect.TypeOf(&model.HotKeyMsg{}):
		return ps.deserializeHotKeyMsg(data)
	default:
		return nil, fmt.Errorf("unsupported type: %v", objType)
	}
}

// serializeHotKeyMsg 序列化HotKeyMsg
// 需要与Java的HotKeyMsg字段顺序和编号保持一致
func (ps *ProtostuffSerializer) serializeHotKeyMsg(msg *model.HotKeyMsg) ([]byte, error) {
	buf := &bytes.Buffer{}

	// 字段1: magicNumber (int32)
	if msg.MagicNumber != 0 {
		ps.writeFieldHeader(buf, 1, WireTypeVarint)
		ps.writeVarint(buf, int64(msg.MagicNumber))
	}

	// 字段2: appName (string)
	if msg.AppName != "" {
		ps.writeFieldHeader(buf, 2, WireTypeLengthDelimited)
		ps.writeString(buf, msg.AppName)
	}

	// 字段3: messageType (enum as int)
	if msg.MessageType != 0 {
		ps.writeFieldHeader(buf, 3, WireTypeVarint)
		ps.writeVarint(buf, int64(msg.MessageType))
	}

	// 字段4: body (string)
	if msg.Body != "" {
		ps.writeFieldHeader(buf, 4, WireTypeLengthDelimited)
		ps.writeString(buf, msg.Body)
	}

	// 字段5: hotKeyModels (repeated)
	if len(msg.HotKeyModels) > 0 {
		for _, hotKey := range msg.HotKeyModels {
			ps.writeFieldHeader(buf, 5, WireTypeLengthDelimited)
			hotKeyData, err := ps.serializeHotKeyModel(hotKey)
			if err != nil {
				return nil, err
			}
			ps.writeBytes(buf, hotKeyData)
		}
	}

	// 字段6: keyCountModels (repeated)
	if len(msg.KeyCountModels) > 0 {
		for _, countModel := range msg.KeyCountModels {
			ps.writeFieldHeader(buf, 6, WireTypeLengthDelimited)
			countData, err := ps.serializeKeyCountModel(countModel)
			if err != nil {
				return nil, err
			}
			ps.writeBytes(buf, countData)
		}
	}

	return buf.Bytes(), nil
}

// serializeHotKeyModel 序列化HotKeyModel
func (ps *ProtostuffSerializer) serializeHotKeyModel(hotKey *model.HotKeyModel) ([]byte, error) {
	buf := &bytes.Buffer{}

	// 继承BaseModel的字段
	// 字段1: id (string)
	if hotKey.ID != "" {
		ps.writeFieldHeader(buf, 1, WireTypeLengthDelimited)
		ps.writeString(buf, hotKey.ID)
	}

	// 字段2: createTime (int64)
	if hotKey.CreateTime != 0 {
		ps.writeFieldHeader(buf, 2, WireTypeVarint)
		ps.writeVarint(buf, hotKey.CreateTime)
	}

	// 字段3: key (string)
	if hotKey.Key != "" {
		ps.writeFieldHeader(buf, 3, WireTypeLengthDelimited)
		ps.writeString(buf, hotKey.Key)
	}

	// 字段4: count (int64) - 对应Java的LongAdder
	count := hotKey.GetCount()
	if count != 0 {
		ps.writeFieldHeader(buf, 4, WireTypeVarint)
		ps.writeVarint(buf, count)
	}

	// HotKeyModel特有字段
	// 字段5: appName (string)
	if hotKey.AppName != "" {
		ps.writeFieldHeader(buf, 5, WireTypeLengthDelimited)
		ps.writeString(buf, hotKey.AppName)
	}

	// 字段6: keyType (enum as int)
	ps.writeFieldHeader(buf, 6, WireTypeVarint)
	ps.writeVarint(buf, int64(hotKey.KeyType))

	// 字段7: remove (bool)
	if hotKey.Remove {
		ps.writeFieldHeader(buf, 7, WireTypeVarint)
		ps.writeVarint(buf, 1)
	}

	return buf.Bytes(), nil
}

// serializeKeyCountModel 序列化KeyCountModel
func (ps *ProtostuffSerializer) serializeKeyCountModel(countModel *model.KeyCountModel) ([]byte, error) {
	buf := &bytes.Buffer{}

	// 字段1: ruleKey (string)
	if countModel.RuleKey != "" {
		ps.writeFieldHeader(buf, 1, WireTypeLengthDelimited)
		ps.writeString(buf, countModel.RuleKey)
	}

	// 字段2: totalHitCount (int32)
	if countModel.TotalHitCount != 0 {
		ps.writeFieldHeader(buf, 2, WireTypeVarint)
		ps.writeVarint(buf, int64(countModel.TotalHitCount))
	}

	// 字段3: hotHitCount (int32)
	if countModel.HotHitCount != 0 {
		ps.writeFieldHeader(buf, 3, WireTypeVarint)
		ps.writeVarint(buf, int64(countModel.HotHitCount))
	}

	// 字段4: createTime (int64)
	if countModel.CreateTime != 0 {
		ps.writeFieldHeader(buf, 4, WireTypeVarint)
		ps.writeVarint(buf, countModel.CreateTime)
	}

	return buf.Bytes(), nil
}

// deserializeHotKeyMsg 反序列化HotKeyMsg
func (ps *ProtostuffSerializer) deserializeHotKeyMsg(data []byte) (*model.HotKeyMsg, error) {
	msg := &model.HotKeyMsg{}
	buf := bytes.NewReader(data)

	for buf.Len() > 0 {
		fieldNum, wireType, err := ps.readFieldHeader(buf)
		if err != nil {
			return nil, err
		}

		switch fieldNum {
		case 1: // magicNumber
			if wireType != WireTypeVarint {
				return nil, fmt.Errorf("invalid wire type for magicNumber")
			}
			val, err := ps.readVarint(buf)
			if err != nil {
				return nil, err
			}
			msg.MagicNumber = int(val)

		case 2: // appName
			if wireType != WireTypeLengthDelimited {
				return nil, fmt.Errorf("invalid wire type for appName")
			}
			val, err := ps.readString(buf)
			if err != nil {
				return nil, err
			}
			msg.AppName = val

		case 3: // messageType
			if wireType != WireTypeVarint {
				return nil, fmt.Errorf("invalid wire type for messageType")
			}
			val, err := ps.readVarint(buf)
			if err != nil {
				return nil, err
			}
			msg.MessageType = model.MessageType(val)

		case 4: // body
			if wireType != WireTypeLengthDelimited {
				return nil, fmt.Errorf("invalid wire type for body")
			}
			val, err := ps.readString(buf)
			if err != nil {
				return nil, err
			}
			msg.Body = val

		case 5: // hotKeyModels
			if wireType != WireTypeLengthDelimited {
				return nil, fmt.Errorf("invalid wire type for hotKeyModels")
			}
			data, err := ps.readBytes(buf)
			if err != nil {
				return nil, err
			}
			hotKey, err := ps.deserializeHotKeyModel(data)
			if err != nil {
				return nil, err
			}
			msg.HotKeyModels = append(msg.HotKeyModels, hotKey)

		case 6: // keyCountModels
			if wireType != WireTypeLengthDelimited {
				return nil, fmt.Errorf("invalid wire type for keyCountModels")
			}
			data, err := ps.readBytes(buf)
			if err != nil {
				return nil, err
			}
			countModel, err := ps.deserializeKeyCountModel(data)
			if err != nil {
				return nil, err
			}
			msg.KeyCountModels = append(msg.KeyCountModels, countModel)

		default:
			// 跳过未知字段
			err := ps.skipField(buf, wireType)
			if err != nil {
				return nil, err
			}
		}
	}

	return msg, nil
}

// deserializeHotKeyModel 反序列化HotKeyModel
func (ps *ProtostuffSerializer) deserializeHotKeyModel(data []byte) (*model.HotKeyModel, error) {
	hotKey := &model.HotKeyModel{}
	buf := bytes.NewReader(data)

	for buf.Len() > 0 {
		fieldNum, wireType, err := ps.readFieldHeader(buf)
		if err != nil {
			return nil, err
		}

		switch fieldNum {
		case 1: // id
			if wireType != WireTypeLengthDelimited {
				return nil, fmt.Errorf("invalid wire type for id")
			}
			val, err := ps.readString(buf)
			if err != nil {
				return nil, err
			}
			hotKey.ID = val

		case 2: // createTime
			if wireType != WireTypeVarint {
				return nil, fmt.Errorf("invalid wire type for createTime")
			}
			val, err := ps.readVarint(buf)
			if err != nil {
				return nil, err
			}
			hotKey.CreateTime = val

		case 3: // key
			if wireType != WireTypeLengthDelimited {
				return nil, fmt.Errorf("invalid wire type for key")
			}
			val, err := ps.readString(buf)
			if err != nil {
				return nil, err
			}
			hotKey.Key = val

		case 4: // count
			if wireType != WireTypeVarint {
				return nil, fmt.Errorf("invalid wire type for count")
			}
			val, err := ps.readVarint(buf)
			if err != nil {
				return nil, err
			}
			hotKey.SetCount(val)

		case 5: // appName
			if wireType != WireTypeLengthDelimited {
				return nil, fmt.Errorf("invalid wire type for appName")
			}
			val, err := ps.readString(buf)
			if err != nil {
				return nil, err
			}
			hotKey.AppName = val

		case 6: // keyType
			if wireType != WireTypeVarint {
				return nil, fmt.Errorf("invalid wire type for keyType")
			}
			val, err := ps.readVarint(buf)
			if err != nil {
				return nil, err
			}
			hotKey.KeyType = model.KeyType(val)

		case 7: // remove
			if wireType != WireTypeVarint {
				return nil, fmt.Errorf("invalid wire type for remove")
			}
			val, err := ps.readVarint(buf)
			if err != nil {
				return nil, err
			}
			hotKey.Remove = val != 0

		default:
			// 跳过未知字段
			err := ps.skipField(buf, wireType)
			if err != nil {
				return nil, err
			}
		}
	}

	return hotKey, nil
}

// deserializeKeyCountModel 反序列化KeyCountModel
func (ps *ProtostuffSerializer) deserializeKeyCountModel(data []byte) (*model.KeyCountModel, error) {
	countModel := &model.KeyCountModel{}
	buf := bytes.NewReader(data)

	for buf.Len() > 0 {
		fieldNum, wireType, err := ps.readFieldHeader(buf)
		if err != nil {
			return nil, err
		}

		switch fieldNum {
		case 1: // ruleKey
			if wireType != WireTypeLengthDelimited {
				return nil, fmt.Errorf("invalid wire type for ruleKey")
			}
			val, err := ps.readString(buf)
			if err != nil {
				return nil, err
			}
			countModel.RuleKey = val

		case 2: // totalHitCount
			if wireType != WireTypeVarint {
				return nil, fmt.Errorf("invalid wire type for totalHitCount")
			}
			val, err := ps.readVarint(buf)
			if err != nil {
				return nil, err
			}
			countModel.TotalHitCount = int(val)

		case 3: // hotHitCount
			if wireType != WireTypeVarint {
				return nil, fmt.Errorf("invalid wire type for hotHitCount")
			}
			val, err := ps.readVarint(buf)
			if err != nil {
				return nil, err
			}
			countModel.HotHitCount = int(val)

		case 4: // createTime
			if wireType != WireTypeVarint {
				return nil, fmt.Errorf("invalid wire type for createTime")
			}
			val, err := ps.readVarint(buf)
			if err != nil {
				return nil, err
			}
			countModel.CreateTime = val

		default:
			// 跳过未知字段
			err := ps.skipField(buf, wireType)
			if err != nil {
				return nil, err
			}
		}
	}

	return countModel, nil
}
