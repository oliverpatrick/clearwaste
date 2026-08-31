class_name ProtocolEncoder
extends RefCounted

static func encode_frame(id: int, payload: PackedByteArray) -> PackedByteArray:
	if payload.size() > MAX_PAYLOAD:
		return PackedByteArray()
	var result := PackedByteArray()
	_put_u16(result, id)
	_put_u16(result, payload.size())
	result.append_array(payload)
	return result