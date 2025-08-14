package serializer

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/jd/platform/hotkey/client-go/model"
)

// Protobuf wire types
const (
	WireTypeVarint          = 0
	WireTypeFixed64         = 1
	WireTypeLengthDelimited = 2
	WireTypeStartGroup      = 3
	WireTypeEndGroup        = 4
	WireTypeFixed32         = 5
)

// writeFieldHeader 写入字段头部（字段号 + wire type）
func (ps *ProtostuffSerializer) writeFieldHeader(buf *bytes.Buffer, fieldNum int, wireType int) {
	tag := (fieldNum << 3) | wireType
	ps.writeVarint(buf, int64(tag))
}

// readFieldHeader 读取字段头部
func (ps *ProtostuffSerializer) readFieldHeader(buf *bytes.Reader) (fieldNum int, wireType int, err error) {
	tag, err := ps.readVarint(buf)
	if err != nil {
		return 0, 0, err
	}
	fieldNum = int(tag >> 3)
	wireType = int(tag & 0x7)
	return fieldNum, wireType, nil
}

// writeVarint 写入变长整数
func (ps *ProtostuffSerializer) writeVarint(buf *bytes.Buffer, value int64) {
	for value >= 0x80 {
		buf.WriteByte(byte(value) | 0x80)
		value >>= 7
	}
	buf.WriteByte(byte(value))
}

// readVarint 读取变长整数
func (ps *ProtostuffSerializer) readVarint(buf *bytes.Reader) (int64, error) {
	var result int64
	var shift uint
	for {
		b, err := buf.ReadByte()
		if err != nil {
			return 0, err
		}
		result |= int64(b&0x7F) << shift
		if b&0x80 == 0 {
			break
		}
		shift += 7
		if shift >= 64 {
			return 0, fmt.Errorf("varint overflow")
		}
	}
	return result, nil
}

// writeString 写入字符串
func (ps *ProtostuffSerializer) writeString(buf *bytes.Buffer, value string) {
	data := []byte(value)
	ps.writeVarint(buf, int64(len(data)))
	buf.Write(data)
}

// readString 读取字符串
func (ps *ProtostuffSerializer) readString(buf *bytes.Reader) (string, error) {
	length, err := ps.readVarint(buf)
	if err != nil {
		return "", err
	}
	data := make([]byte, length)
	_, err = io.ReadFull(buf, data)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// writeBytes 写入字节数组
func (ps *ProtostuffSerializer) writeBytes(buf *bytes.Buffer, data []byte) {
	ps.writeVarint(buf, int64(len(data)))
	buf.Write(data)
}

// readBytes 读取字节数组
func (ps *ProtostuffSerializer) readBytes(buf *bytes.Reader) ([]byte, error) {
	length, err := ps.readVarint(buf)
	if err != nil {
		return nil, err
	}
	data := make([]byte, length)
	_, err = io.ReadFull(buf, data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// writeFixed32 写入32位固定长度整数
func (ps *ProtostuffSerializer) writeFixed32(buf *bytes.Buffer, value uint32) {
	binary.LittleEndian.PutUint32(buf.Bytes()[buf.Len():buf.Len()+4], value)
}

// readFixed32 读取32位固定长度整数
func (ps *ProtostuffSerializer) readFixed32(buf *bytes.Reader) (uint32, error) {
	data := make([]byte, 4)
	_, err := io.ReadFull(buf, data)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(data), nil
}

// writeFixed64 写入64位固定长度整数
func (ps *ProtostuffSerializer) writeFixed64(buf *bytes.Buffer, value uint64) {
	binary.LittleEndian.PutUint64(buf.Bytes()[buf.Len():buf.Len()+8], value)
}

// readFixed64 读取64位固定长度整数
func (ps *ProtostuffSerializer) readFixed64(buf *bytes.Reader) (uint64, error) {
	data := make([]byte, 8)
	_, err := io.ReadFull(buf, data)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(data), nil
}

// skipField 跳过字段
func (ps *ProtostuffSerializer) skipField(buf *bytes.Reader, wireType int) error {
	switch wireType {
	case WireTypeVarint:
		_, err := ps.readVarint(buf)
		return err
	case WireTypeFixed64:
		_, err := ps.readFixed64(buf)
		return err
	case WireTypeLengthDelimited:
		length, err := ps.readVarint(buf)
		if err != nil {
			return err
		}
		data := make([]byte, length)
		_, err = io.ReadFull(buf, data)
		return err
	case WireTypeFixed32:
		_, err := ps.readFixed32(buf)
		return err
	default:
		return fmt.Errorf("unknown wire type: %d", wireType)
	}
}

// EncodeWithDelimiter 使用分隔符编码消息，对应Java的MsgEncoder
func (ps *ProtostuffSerializer) EncodeWithDelimiter(obj interface{}) ([]byte, error) {
	// 序列化对象
	data, err := ps.Serialize(obj)
	if err != nil {
		return nil, err
	}

	// 添加分隔符
	delimiter := []byte("$(* *)$") // 对应Java的Constant.DELIMITER
	total := make([]byte, len(data)+len(delimiter))
	copy(total, data)
	copy(total[len(data):], delimiter)

	return total, nil
}

// DecodeWithDelimiter 使用分隔符解码消息，对应Java的MsgDecoder
func (ps *ProtostuffSerializer) DecodeWithDelimiter(data []byte, objType interface{}) (interface{}, error) {
	// 移除分隔符
	delimiter := []byte("$(* *)$")
	if len(data) < len(delimiter) {
		return nil, fmt.Errorf("data too short")
	}

	// 检查分隔符
	dataLen := len(data) - len(delimiter)
	if !bytes.Equal(data[dataLen:], delimiter) {
		return nil, fmt.Errorf("invalid delimiter")
	}

	// 反序列化
	actualData := data[:dataLen]
	switch objType.(type) {
	case **model.HotKeyMsg:
		return ps.deserializeHotKeyMsg(actualData)
	default:
		return nil, fmt.Errorf("unsupported type")
	}
}

// ValidateMessage 验证消息格式
func (ps *ProtostuffSerializer) ValidateMessage(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("empty data")
	}
	if len(data) > 4*1024*1024 { // 4MB limit
		return fmt.Errorf("data too large: %d bytes", len(data))
	}
	return nil
}
