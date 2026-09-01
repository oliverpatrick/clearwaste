extends RefCounted

const HEADER_SIZE := 6
const MAX_PAYLOAD := 65536
const CLIENT_HELLO := 1
const SERVER_HELLO := 2
const LOGIN_REQUEST := 3
const LOGIN_ACCEPTED := 4
const LOGIN_REJECTED := 5
const MOVE_REQUEST := 6
const SET_RUN_ENABLED := 7
const INTERACT := 8
const WORLD_BOOTSTRAP := 9
const CONNECT_ACK := 0x0101
const ENTITY_SPAWN := 0x0103
const ENTITY_DESPAWN := 0x0104
const ENTITY_MOVE := 0x0105
const ENTITY_HEALTH := 0x0127
const ENTITY_FACE := 0x0126
const PLAYER_ACTION := 0x0125
const GROUND_ITEM := 0x0122
const RESOURCE_STATE := 0x0124

static func encode_frame(opcode: int, payload: PackedByteArray) -> PackedByteArray:
	if payload.size() > MAX_PAYLOAD:
		return PackedByteArray()
	var result := PackedByteArray()
	_put_u16(result, opcode)
	_put_u32(result, payload.size())
	result.append_array(payload)
	return result

static func decode_frame(data: PackedByteArray):
	if data.size() < HEADER_SIZE:
		return null
	var size := _u32(data, 2)
	if size > MAX_PAYLOAD or data.size() != HEADER_SIZE + size:
		return null
	return {"id": _u16(data, 0), "payload": data.slice(HEADER_SIZE)}

static func encode_client_hello(version: int) -> PackedByteArray:
	var payload := PackedByteArray()
	_put_u16(payload, version)
	return encode_frame(CLIENT_HELLO, payload)

static func encode_login_request(ticket: String) -> PackedByteArray:
	var bytes := ticket.to_utf8_buffer()
	if bytes.size() > 65535:
		return PackedByteArray()
	var payload := PackedByteArray()
	_put_u16(payload, bytes.size())
	payload.append_array(bytes)
	return encode_frame(LOGIN_REQUEST, payload)

static func decode_server_hello(payload: PackedByteArray):
	if payload.size() != 3 or payload[0] > 1:
		return null
	return {"accepted": payload[0] == 1, "version": _u16(payload, 1)}

static func decode_bootstrap(payload: PackedByteArray):
	if payload.size() < 21: return null
	var offset := 0
	var local_id := _u64(payload, offset); offset += 8
	var region_x := _s32(payload, offset); offset += 4
	var region_z := _s32(payload, offset); offset += 4
	var plane := payload[offset]; offset += 1
	var count := _u16(payload, offset); offset += 2
	var entities: Array = []
	for _i in range(count):
		if offset + 30 > payload.size(): return null
		var item := {"id": _u64(payload, offset), "kind": payload[offset + 8], "definition_id": _u16(payload, offset + 9), "character_id": _u64(payload, offset + 11), "x": _s32(payload, offset + 19), "z": _s32(payload, offset + 23), "plane": payload[offset + 27], "appearance_id": _u16(payload, offset + 28)}
		offset += 30
		entities.append(item)
	if offset != payload.size(): return null
	return {"local_entity_id": local_id, "region_x": region_x, "region_z": region_z, "plane": plane, "entities": entities}

static func encode_move(x: int, z: int, plane: int, mode: int, sequence: int) -> PackedByteArray:
	var payload := PackedByteArray()
	_put_u32(payload, x); _put_u32(payload, z); payload.append(plane); payload.append(mode); _put_u32(payload, sequence)
	return encode_frame(MOVE_REQUEST, payload)

static func decode_message(_id: int, _payload: PackedByteArray):
	return null

static func _put_u16(bytes: PackedByteArray, value: int) -> void:
	bytes.append((value >> 8) & 0xff); bytes.append(value & 0xff)

static func _put_u32(bytes: PackedByteArray, value: int) -> void:
	bytes.append((value >> 24) & 0xff); bytes.append((value >> 16) & 0xff); bytes.append((value >> 8) & 0xff); bytes.append(value & 0xff)

static func _u16(bytes: PackedByteArray, offset: int) -> int:
	return (bytes[offset] << 8) | bytes[offset + 1]

static func _u32(bytes: PackedByteArray, offset: int) -> int:
	return (bytes[offset] << 24) | (bytes[offset + 1] << 16) | (bytes[offset + 2] << 8) | bytes[offset + 3]

static func _s32(bytes: PackedByteArray, offset: int) -> int:
	var value := _u32(bytes, offset)
	return value - 4294967296 if value >= 2147483648 else value

static func _u64(bytes: PackedByteArray, offset: int) -> int:
	return (_u32(bytes, offset) << 32) | _u32(bytes, offset + 4)
