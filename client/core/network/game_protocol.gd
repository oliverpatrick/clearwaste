class_name GameProtocol
extends RefCounted


static func _strict_utf8(bytes: PackedByteArray):
	var text := bytes.get_string_from_utf8()
	if text.to_utf8_buffer() != bytes:
		return null
	return text

static func _put_u16(bytes: PackedByteArray, value: int) -> void:
	bytes.append((value >> 8) & 0xff)
	bytes.append(value & 0xff)

static func _put_u32(bytes: PackedByteArray, value: int) -> void:
	bytes.append((value >> 24) & 0xff)
	bytes.append((value >> 16) & 0xff)
	bytes.append((value >> 8) & 0xff)
	bytes.append(value & 0xff)

static func _u16(bytes: PackedByteArray, offset: int) -> int:
	return (bytes[offset] << 8) | bytes[offset + 1]

static func _s16(bytes: PackedByteArray, offset: int) -> int:
	var value := _u16(bytes, offset)
	return value - 65536 if value >= 32768 else value

static func _u32(bytes: PackedByteArray, offset: int) -> int:
	return (bytes[offset] << 24) | (bytes[offset + 1] << 16) | (bytes[offset + 2] << 8) | bytes[offset + 3]

static func _s32(bytes: PackedByteArray, offset: int) -> int:
	var value := _u32(bytes, offset)
	return value - 4294967296 if value >= 2147483648 else value

static func _u64(bytes: PackedByteArray, offset: int) -> int:
	return (_u32(bytes, offset) << 32) | _u32(bytes, offset + 4)
